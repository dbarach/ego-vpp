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
#include <skipnat/skipnat.h>

typedef struct
{
  u32 sw_if_index;
  u32 next_index;
} skipnat_trace_t;

#ifndef CLIB_MARCH_VARIANT

/* packet trace format function */
static u8 * format_skipnat_trace (u8 * s, va_list * args)
{
  CLIB_UNUSED (vlib_main_t * vm) = va_arg (*args, vlib_main_t *);
  CLIB_UNUSED (vlib_node_t * node) = va_arg (*args, vlib_node_t *);
  skipnat_trace_t * t = va_arg (*args, skipnat_trace_t *);

  s = format (s, "SKIPNAT: sw_if_index %d, %s",
              t->sw_if_index, t->next_index == 0 ? "skip nat" : "do nat");
  return s;
}

vlib_node_registration_t skipnat_node;

#endif /* CLIB_MARCH_VARIANT */

#define foreach_skipnat_error                   \
_(SKIPPED, "Skip-nat packets")                  \
_(NATTED, "Do-nat packets")

typedef enum
{
#define _(sym,str) SKIPNAT_ERROR_##sym,
  foreach_skipnat_error
#undef _
  SKIPNAT_N_ERROR,
} skipnat_error_t;

#ifndef CLIB_MARCH_VARIANT
static char * skipnat_error_strings[] =
{
#define _(sym,string) string,
  foreach_skipnat_error
#undef _
};
#endif /* CLIB_MARCH_VARIANT */

typedef enum
{
  SKIPNAT_NEXT_LOOKUP,
  SKIPNAT_N_NEXT,
} skipnat_next_t;

/*
 * Don't NAT home network inside addrs 192.168.1.*, 192.168.2.*,
 * 192.168.3.*, and 192.168.40.*
 * And don't nat to-openvpn-tunnel pkts on 192.168.10.*
 */
static inline int skip_nat_by_addr (u32 addr)
{
  u32 sum;

  addr &= clib_host_to_net_u32 (0xFFFFFF00);

  sum = (addr ^ clib_host_to_net_u32(0xc0a80100)) != 0;
  sum += (addr ^ clib_host_to_net_u32(0xc0a80200)) != 0;
  sum += (addr ^ clib_host_to_net_u32(0xc0a80300)) != 0;
  sum += (addr ^ clib_host_to_net_u32(0xc0a80a00)) != 0;
  sum += (addr ^ clib_host_to_net_u32(0xc0a82800)) != 0;

  return sum < 5;
}

always_inline uword
skipnat_inline (vlib_main_t * vm,
     		 vlib_node_runtime_t * node, vlib_frame_t * frame,
		 int is_ip4, int is_trace)
{
  u32 n_left_from, *from;
  vlib_buffer_t *bufs[VLIB_FRAME_SIZE], **b;
  u16 nexts[VLIB_FRAME_SIZE], *next;
  u32 arc_next0, arc_next1, arc_next2, arc_next3;
  ip4_header_t *ip40, *ip41, *ip42, *ip43;
  int skip0, skip1, skip2, skip3;
  int skipped = 0;

  from = vlib_frame_vector_args (frame);
  n_left_from = frame->n_vectors;

  vlib_get_buffers (vm, from, bufs, n_left_from);
  b = bufs;
  next = nexts;


  while (n_left_from >= 4)
    {
      /* Prefetch next iteration. */
      if (PREDICT_TRUE (n_left_from >= 8))
	{
	  vlib_prefetch_buffer_header (b[4], STORE);
	  vlib_prefetch_buffer_header (b[5], STORE);
	  vlib_prefetch_buffer_header (b[6], STORE);
	  vlib_prefetch_buffer_header (b[7], STORE);
          CLIB_PREFETCH (b[4]->data, CLIB_CACHE_LINE_BYTES, STORE);
          CLIB_PREFETCH (b[5]->data, CLIB_CACHE_LINE_BYTES, STORE);
          CLIB_PREFETCH (b[6]->data, CLIB_CACHE_LINE_BYTES, STORE);
          CLIB_PREFETCH (b[7]->data, CLIB_CACHE_LINE_BYTES, STORE);
	}

      vnet_feature_next (&arc_next0, b[0]);
      vnet_feature_next (&arc_next1, b[1]);
      vnet_feature_next (&arc_next2, b[2]);
      vnet_feature_next (&arc_next3, b[3]);

      next[0] = arc_next0;
      next[1] = arc_next1;
      next[2] = arc_next2;
      next[3] = arc_next3;

      ip40 = vlib_buffer_get_current (b[0]);
      ip41 = vlib_buffer_get_current (b[1]);
      ip42 = vlib_buffer_get_current (b[2]);
      ip43 = vlib_buffer_get_current (b[3]);

      skip0 = skip_nat_by_addr (ip40->dst_address.as_u32);
      skip1 = skip_nat_by_addr (ip41->dst_address.as_u32);
      skip2 = skip_nat_by_addr (ip42->dst_address.as_u32);
      skip3 = skip_nat_by_addr (ip43->dst_address.as_u32);
      skipped += skip0 + skip1 + skip2 + skip3;

      if (skip0)
        next[0] = SKIPNAT_NEXT_LOOKUP;
      if(skip1)
        next[1] = SKIPNAT_NEXT_LOOKUP;
      if(skip2)
        next[2] = SKIPNAT_NEXT_LOOKUP;
      if(skip3)
        next[3] = SKIPNAT_NEXT_LOOKUP;

      if (is_trace)
	{
	  if (b[0]->flags & VLIB_BUFFER_IS_TRACED)
	    {
	      skipnat_trace_t *t =
                   vlib_add_trace (vm, node, b[0], sizeof (*t));
	      t->next_index = next[0];
              t->sw_if_index = vnet_buffer(b[0])->sw_if_index[VLIB_RX];
	    }
	  if (b[1]->flags & VLIB_BUFFER_IS_TRACED)
	    {
	      skipnat_trace_t *t =
                    vlib_add_trace (vm, node, b[1], sizeof (*t));
	      t->next_index = next[1];
              t->sw_if_index = vnet_buffer(b[1])->sw_if_index[VLIB_RX];
	    }
	  if (b[2]->flags & VLIB_BUFFER_IS_TRACED)
	    {
	      skipnat_trace_t *t =
                    vlib_add_trace (vm, node, b[2], sizeof (*t));
	      t->next_index = next[2];
              t->sw_if_index = vnet_buffer(b[2])->sw_if_index[VLIB_RX];
	    }
	  if (b[3]->flags & VLIB_BUFFER_IS_TRACED)
	    {
	      skipnat_trace_t *t =
                    vlib_add_trace (vm, node, b[3], sizeof (*t));
	      t->next_index = next[3];
              t->sw_if_index = vnet_buffer(b[3])->sw_if_index[VLIB_RX];
	    }
	}

      b += 4;
      next += 4;
      n_left_from -= 4;
    }

  while (n_left_from > 0)
    {

      vnet_feature_next (&arc_next0, b[0]);
      next[0] = arc_next0;
      ip40 = vlib_buffer_get_current (b[0]);
      skip0 = skip_nat_by_addr (ip40->dst_address.as_u32);

      skipped += skip0;
      if (skip0)
        next[0] = SKIPNAT_NEXT_LOOKUP;

      if (is_trace)
	{
	  if (b[0]->flags & VLIB_BUFFER_IS_TRACED)
	    {
	      skipnat_trace_t *t =
                    vlib_add_trace (vm, node, b[0], sizeof (*t));
	      t->next_index = next[0];
              t->sw_if_index = vnet_buffer(b[0])->sw_if_index[VLIB_RX];
	    }
	}

      b += 1;
      next += 1;
      n_left_from -= 1;
    }

  vlib_buffer_enqueue_to_next (vm, node, from, nexts, frame->n_vectors);
  vlib_node_increment_counter (vm, node->node_index,
			       SKIPNAT_ERROR_SKIPPED, skipped);
  vlib_node_increment_counter (vm, node->node_index,
			       SKIPNAT_ERROR_NATTED,
                               frame->n_vectors - skipped);
  return frame->n_vectors;
}

VLIB_NODE_FN (skipnat_node) (vlib_main_t * vm, vlib_node_runtime_t * node,
                             vlib_frame_t * frame)
{
  if (PREDICT_FALSE (node->flags & VLIB_NODE_FLAG_TRACE))
    return skipnat_inline (vm, node, frame, 1 /* is_ip4 */ ,
			    1 /* is_trace */ );
  else
    return skipnat_inline (vm, node, frame, 1 /* is_ip4 */ ,
			    0 /* is_trace */ );
}

/* *INDENT-OFF* */
#ifndef CLIB_MARCH_VARIANT
VLIB_REGISTER_NODE (skipnat_node) =
{
  .name = "skipnat",
  .vector_size = sizeof (u32),
  .format_trace = format_skipnat_trace,
  .type = VLIB_NODE_TYPE_INTERNAL,

  .n_errors = ARRAY_LEN(skipnat_error_strings),
  .error_strings = skipnat_error_strings,

  .n_next_nodes = SKIPNAT_N_NEXT,

  /* edit / add dispositions here */
  .next_nodes = {
        [SKIPNAT_NEXT_LOOKUP] = "ip4-lookup",
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
