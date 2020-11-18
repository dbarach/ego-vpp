/*
 * node.c - skeleton vpp engine plug-in dual-loop node skeleton
 *
 * Copyright (c) <current-year> <your-organization>
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
#include <vlib/vlib.h>
#include <vnet/vnet.h>
#include <vnet/pg/pg.h>
#include <vppinfra/error.h>
#include <vnet/tcp/tcp_inlines.h>
#include <tcptrack/tcptrack.h>

typedef struct
{
  u8 in_order;
  u8 out_of_order;
  u8 new;
  u8 deleted;
  u8 skipped;
} tcptrack_trace_t;

#ifndef CLIB_MARCH_VARIANT

/* packet trace format function */
static u8 * format_tcptrack_trace (u8 * s, va_list * args)
{
  CLIB_UNUSED (vlib_main_t * vm) = va_arg (*args, vlib_main_t *);
  CLIB_UNUSED (vlib_node_t * node) = va_arg (*args, vlib_node_t *);
  tcptrack_trace_t * t = va_arg (*args, tcptrack_trace_t *);

  s = format (s,
              "TCPTRACK: skip %d in_order %d out_of_order %d new %d deleted %d",
              (u32)(t->skipped), (u32)(t->in_order), (u32)(t->out_of_order),
              (u32)(t->new), (u32)(t->deleted));

  return s;
}

vlib_node_registration_t tcptrack_node;

#endif /* CLIB_MARCH_VARIANT */

#define foreach_tcptrack_error                          \
_(TOTAL_PKTS, "Total TCP packets")                      \
_(IN_ORDER_PKTS, "Total in-order TCP packets")          \
_(OUT_OF_ORDER_PKTS, "Total out-of-order TCP packets")

typedef enum
{
#define _(sym,str) TCPTRACK_ERROR_##sym,
  foreach_tcptrack_error
#undef _
  TCPTRACK_N_ERROR,
} tcptrack_error_t;

#ifndef CLIB_MARCH_VARIANT
static char * tcptrack_error_strings[] =
{
#define _(sym,string) string,
  foreach_tcptrack_error
#undef _
};
#endif /* CLIB_MARCH_VARIANT */

typedef enum
{
  TCPTRACK_NEXT_DROP,
  TCPTRACK_N_NEXT,
} tcptrack_next_t;

always_inline uword
tcptrack_inline (vlib_main_t * vm,
     		 vlib_node_runtime_t * node, vlib_frame_t * frame,
		 int is_ip4, int is_trace)
{
  tcptrack_main_t *ttm = &tcptrack_main;
  u32 n_left_from, *from;
  vlib_buffer_t *bufs[VLIB_FRAME_SIZE], **b;
  u16 nexts[VLIB_FRAME_SIZE], *next;
  u32 tcp_pkts = 0;
  u32 in_order_pkts = 0;
  u32 out_of_order_pkts = 0;
  f64 now = vlib_time_now(vm);

  from = vlib_frame_vector_args (frame);
  n_left_from = frame->n_vectors;

  vlib_get_buffers (vm, from, bufs, n_left_from);
  b = bufs;
  next = nexts;

  while (n_left_from > 0)
    {
      ip4_header_t *ip0;
      tcp_header_t *tcp0;
      u32 next0;
      u32 seq_number;
      tcp_conn_t *tc;
      clib_bihash_kv_8_8_t kv;
      int n_advance_bytes0, n_data_bytes0, ip_hdr_bytes0;
      u8 in_order = 0;
      u8 out_of_order = 0;
      u8 new =0;
      u8 deleted = 0;
      u8 skipped = 0;

      /* Are we having fun yet? */
      vnet_feature_next (&next0, b[0]);
      next[0] = next0;

      ip0 = vlib_buffer_get_current (b[0]);

      /* Must be ip4, or how did we get here? */
      ASSERT ((ip0->ip_version_and_header_length & 0xF0) == 0x40);

      /* TCP? */
      if (ip0->protocol != 6)
        {
          skipped = 1;
          goto trace0;
        }

      tcp0 = ip4_next_header (ip0);

      kv.key = ((u64) ip0->src_address.as_u32 << 32) | tcp0->dst_port;

      seq_number = clib_net_to_host_u32 (tcp0->seq_number);
      ip_hdr_bytes0 = ip4_header_bytes (ip0);
      n_advance_bytes0 = (ip_hdr_bytes0 + tcp_header_bytes (tcp0));
      n_data_bytes0 = clib_net_to_host_u16 (ip0->length) - n_advance_bytes0;

      if (clib_bihash_search_8_8 (&ttm->ct4, &kv, &kv))
        {
          /* Connection going away? Don't bother making a new entry... */
          if (tcp_is_fin (tcp0) || !!tcp_rst(tcp0))
            goto trace0;

          /* Set up new connection */
          new = 1;
          pool_get (ttm->connections[vm->thread_index], tc);
          tc->src_address = ip0->src_address.as_u32;
          tc->dst_port = tcp0->dst_port;
          tc->last_heard = now;
          tc->out_of_order_pkts = 0;
          tc->in_order_pkts = 0;
          if (tcp_is_syn(tcp0))
            tc->next_seq_expected = seq_number + 1;
          else
            tc->next_seq_expected = seq_number + n_data_bytes0;
          kv.value = tc - ttm->connections[vm->thread_index];
          (void) clib_bihash_add_del_8_8 (&ttm->ct4, &kv, 1 /* is_add */);
        }
      else
        {
          tc = pool_elt_at_index (ttm->connections[vm->thread_index],
                                  kv.value);
          /* Connection going away? Turf the (bihash, pool) entries */
          if (tcp_is_fin (tcp0) || !!tcp_rst(tcp0))
            {
              if (clib_bihash_add_del_8_8 (&ttm->ct4, &kv, 0 /* is_add */))
                {
                  /* Damn, we just found this one! */
                  clib_warning ("Src %U dst_port %u AWOL!",
                                format_ip4_address, &ip0->src_address,
                                clib_net_to_host_u16(tcp0->dst_port));
                }
              pool_put (ttm->connections[vm->thread_index], tc);
              deleted = 1;
            }

          if (tc->next_seq_expected != seq_number)
            {
              tc->out_of_order_pkts ++;
              out_of_order = 1;
            }
          else
            {
              tc->in_order_pkts ++;
              in_order = 1;
            }
          /* Obviously an approximation... */
          tc->next_seq_expected = seq_number + n_data_bytes0;
          tc->last_heard = now;
        }

    trace0:
      if (is_trace)
	{
	  if (b[0]->flags & VLIB_BUFFER_IS_TRACED)
	    {
	      tcptrack_trace_t *t =
                    vlib_add_trace (vm, node, b[0], sizeof (*t));
              t->in_order = in_order;
              t->out_of_order = out_of_order;
              t->new = new;
              t->deleted = deleted;
              t->skipped = skipped;
	    }
	}

      tcp_pkts += skipped ^ 1;
      in_order_pkts += in_order;
      out_of_order_pkts += out_of_order;

      b += 1;
      next += 1;
      n_left_from -= 1;
    }

  vlib_buffer_enqueue_to_next (vm, node, from, nexts, frame->n_vectors);

  vlib_node_increment_counter (vm, node->node_index,
                               TCPTRACK_ERROR_TOTAL_PKTS,
                               tcp_pkts);
  vlib_node_increment_counter (vm, node->node_index,
                               TCPTRACK_ERROR_IN_ORDER_PKTS,
                               in_order_pkts);
  vlib_node_increment_counter (vm, node->node_index,
                               TCPTRACK_ERROR_OUT_OF_ORDER_PKTS,
                               out_of_order_pkts);

  return frame->n_vectors;
}

VLIB_NODE_FN (tcptrack_node) (vlib_main_t * vm, vlib_node_runtime_t * node,
                             vlib_frame_t * frame)
{
  if (PREDICT_FALSE (node->flags & VLIB_NODE_FLAG_TRACE))
    return tcptrack_inline (vm, node, frame, 1 /* is_ip4 */ ,
			    1 /* is_trace */ );
  else
    return tcptrack_inline (vm, node, frame, 1 /* is_ip4 */ ,
			    0 /* is_trace */ );
}

/* *INDENT-OFF* */
#ifndef CLIB_MARCH_VARIANT
VLIB_REGISTER_NODE (tcptrack_node) =
{
  .name = "tcptrack",
  .vector_size = sizeof (u32),
  .format_trace = format_tcptrack_trace,
  .type = VLIB_NODE_TYPE_INTERNAL,

  .n_errors = ARRAY_LEN(tcptrack_error_strings),
  .error_strings = tcptrack_error_strings,

  .n_next_nodes = TCPTRACK_N_NEXT,

  /* edit / add dispositions here */
  .next_nodes = {
        [TCPTRACK_NEXT_DROP] = "error-drop",
  },
};
#endif /* CLIB_MARCH_VARIANT */
/* *INDENT-ON* */
/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
