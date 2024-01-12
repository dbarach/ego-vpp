/*
 * in2out.c - hgw in2out translation
 *
 * Copyright (c) 2023 by David R. Barach
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
#include <vppinfra/random.h>
#include <hgnat/hgnat.h>

static f64 hgnat_in2out_now;

typedef struct
{
  u32 protocol;
  u32 session_index;
  u32 next_index;
} hgnat_in2out_trace_t;

#ifndef CLIB_MARCH_VARIANT
/* packet trace format function */
static u8 * format_hgnat_in2out_trace (u8 * s, va_list * args)
{
  CLIB_UNUSED (vlib_main_t * vm) = va_arg (*args, vlib_main_t *);
  CLIB_UNUSED (vlib_node_t * node) = va_arg (*args, vlib_node_t *);
  hgnat_in2out_trace_t * t = va_arg (*args, hgnat_in2out_trace_t *);


  if (t->session_index != ~0)
    s = format (s, "hgnat_in2out: next index %d, session index %d, protocol %s",
                t->next_index, t->session_index,
                hgnat_protocol_strings[t->protocol]);
  else
    s = format (s, "hgnat_in2out: next index %d, no session, protocol %d",
                t->next_index, hgnat_protocol_strings[t->protocol]);

  return s;
}
#endif /*  CLIB_MARCH_VARIANT */

#define foreach_hgnat_error                                                   \
  _ (HGNAT_IN2OUT_TRANSLATED, "Translated Packets")                           \
  _ (HGNAT_IN2OUT_DROP, "Dropped Packets")                                    \
  _ (HGNAT_IN2OUT_NO_OUTSIDE_ADDR, "Trunk port address unresolved")           \
  _ (HGNAT_IN2OUT_UNKNOWN_PROTOCOL, "Unknown protocol not translated")        \
  _ (HGNAT_IN2OUT_SESSIONS_ADDED, "Sessions added")                           \
  _ (HGNAT_IN2OUT_TTL_DROP, "Traceroute first-hop pkts (ttl drop)")

typedef enum
{
#define _(sym,str) HGNAT_ERROR_##sym,
  foreach_hgnat_error
#undef _
  HGNAT_N_ERROR,
} hgnat_error_t;

#ifndef CLIB_MARCH_VARIANT
static char * hgnat_error_strings[] =
{
#define _(sym,string) string,
  foreach_hgnat_error
#undef _
};
#endif /*  CLIB_MARCH_VARIANT */

typedef enum
{
  HGNAT_NEXT_DROP,
  HGNAT_NEXT_LOOKUP,
  HGNAT_NEXT_ICMP_ERROR,
  HGNAT_N_NEXT,
} hgnat_next_t;

static u16 hgm_alloc_port (ip4_address_t addr, u8 proto)
{

  int i;
  u32 rn;
  hgnat_main_t *hgm = &hgnat_main;
  hgnat_session_key_t key;
  clib_bihash_kv_8_8_t kv, value;

  for (i = 0; i < 65536 - 1024; i++)
    {
      while (1)
        {
          rn = random_u32(&hgm->port_random_seed);
          rn &= 0xFFFF;
          if (rn > 1024 && rn < 0xFFFF)
            break;
        }

      key.as_u64 = 0;
      key.addr = addr;
      key.port = (u16) rn;
      key.protocol = proto;
      kv.key = key.as_u64;

      if (clib_bihash_search_8_8 (&hgm->out2in, &kv, &value))
        {
          /* lookup failed, we found a usable translation */
          return clib_host_to_net_u16((u16) rn);
        }
    }
  /* Completely out of ports on the outside address... */
  abort();
  /* not so much */
  return (0xffff);
}

always_inline uword
in2out_inline (vlib_main_t * vm, vlib_node_runtime_t * node,
               vlib_buffer_t *b, int is_trace)
{
  hgnat_main_t *hgm = &hgnat_main;
  u32 translated = 0;
  u32 dropped = 0;
  u32 no_outside_address = 0;
  u32 unknown_protocol = 0;
  u32 sessions_added = 0;
  u32 ttl_drops = 0;
  ip4_header_t * ip;
  u32 next;
  ip_csum_t sum;
  u32 new_addr, old_addr;
  u16 old_port, new_port;
  udp_header_t * udp;
  tcp_header_t * tcp;
  icmp46_header_t * icmp;
  icmp_echo_header_t *echo;
  hgnat_session_key_t key;
  u32 proto;
  hgnat_session_t * s;
  clib_bihash_kv_8_8_t kv, value;
  dlist_elt_t *lru;
  u32 feature_next;


  /* Initial guess: send to next on feature arc */
  vnet_feature_next(&feature_next, b);
  next = feature_next;

  ip = vlib_buffer_get_current (b);
  udp = ip4_next_header (ip);
  tcp = (tcp_header_t *) udp;
  icmp = (icmp46_header_t *) udp;
  echo = (icmp_echo_header_t *)(icmp + 1);
  s = 0; /* for tracing */

  /* Waiting on DHCP resolution? */
  if (PREDICT_FALSE (hgm->outside_addr.as_u32 == 0))
    {
      b->error = HGNAT_ERROR_HGNAT_IN2OUT_NO_OUTSIDE_ADDR;
      next = HGNAT_NEXT_DROP;
      no_outside_address++;
      goto trace;
    }

  proto = HGNAT_PROTOCOL_UNKNOWN;
  proto = (ip->protocol == IP_PROTOCOL_UDP)
    ? HGNAT_PROTOCOL_UDP : proto;
  proto = (ip->protocol == IP_PROTOCOL_TCP)
    ? HGNAT_PROTOCOL_TCP : proto;
  proto = (ip->protocol == IP_PROTOCOL_ICMP)
    ? HGNAT_PROTOCOL_ICMP : proto;

  /* Not a protocol we deal with? Don't translate */
  if (PREDICT_FALSE(proto == HGNAT_PROTOCOL_UNKNOWN))
    {
      unknown_protocol++;
      next = HGNAT_NEXT_LOOKUP;
      goto trace;
    }

  /* TTL drop (traceroute)? */
  if (PREDICT_FALSE (ip->ttl == 1))
    {
      ttl_drops++;
      icmp4_error_set_vnet_buffer (b, 11 /* TTL exceeded */,
                                   0 /* code 0 = "in transit" */,
                                   0 /* data unused */);
      next = HGNAT_NEXT_ICMP_ERROR;
      goto trace;
    }

  /*
   * Untranslated cases: unknown protocols, don't nat inside network
   * addresses. Ought to work harder on the netmask, someday.
   */
  if (PREDICT_FALSE
      ((clib_net_to_host_u32 (ip->dst_address.as_u32) & 0xFFFFFF00)
       ==
       (clib_net_to_host_u32 (ip->src_address.as_u32) & 0xFFFFFF00)))
    {
      /* next set to next feature on arc... */
      goto trace;
    }

  key.as_u64 = 0;
  key.addr = ip->src_address;
  key.port = (proto != HGNAT_PROTOCOL_ICMP) ? udp->src_port : echo->identifier;
  key.protocol = proto;
  kv.key = key.as_u64;

  if (clib_bihash_search_8_8 (&hgm->in2out, &kv, &value))
    {
      /* Clean up stale translations? */
      hgnat_maybe_reclaim_translations(hgm, 0 /* all */);
      /* No such translation (yet), add one */
      pool_get (hgm->session_pool, s);
      memset (s, 0, sizeof (*s));
      kv.value = s - hgm->session_pool;

      memset(s, 0, sizeof(*s));
      s->in2out = key;
      clib_bihash_add_del_8_8 (&hgm->in2out, &kv, 1 /* is_add */);

      /* Form out-to-in translation */
      key.addr = ip->dst_address;
      key.port = hgm_alloc_port(ip->dst_address, proto);
      kv.key = key.as_u64;
      s->out2in = key;
      if (0)
        clib_warning ("add out2in map for %U, port %d",
                      format_ip4_address, &key.addr,
                      clib_net_to_host_u16(key.port));

      clib_bihash_add_del_8_8 (&hgm->out2in, &kv, 1 /* is_add */);

      pool_get (hgm->list_pool, lru);
      clib_dlist_init (hgm->list_pool, lru - hgm->list_pool);
      lru->value = s - hgm->session_pool;
      s->global_lru_list_index = lru - hgm->list_pool;
      clib_dlist_addtail (hgm->list_pool, hgm->global_lru_list_head_index,
                          s->global_lru_list_index);

      /* for convenience when fixing ICMP error replies */
      s->orig_src_port = udp->src_port;
      s->orig_dst_port = udp->dst_port;
      hgm->active_translations ++;
      sessions_added ++;
    }
  else
    s = pool_elt_at_index (hgm->session_pool, value.value);

  old_addr = ip->src_address.as_u32;
  ip->src_address = hgm->outside_addr;
  new_addr = ip->src_address.as_u32;

  sum = ip->checksum;
  sum = ip_csum_update (sum, old_addr, new_addr,
                        ip4_header_t,
                        src_address /* changed member */);
  ip->checksum = ip_csum_fold (sum);

  if (PREDICT_TRUE(proto == HGNAT_PROTOCOL_TCP))
    {
      old_port = tcp->src_port;
      tcp->src_port = s->out2in.port;
      new_port = tcp->src_port;

      sum = tcp->checksum;
      sum = ip_csum_update (sum, old_addr, new_addr,
                            ip4_header_t,
                            dst_address /* changed member */);
      sum = ip_csum_update (sum, old_port, new_port,
                            ip4_header_t /* cheat */,
                            length /* changed member */);
      tcp->checksum = ip_csum_fold(sum);
    }
  else if (proto == HGNAT_PROTOCOL_UDP)
    {
      old_port = udp->src_port;
      udp->src_port = s->out2in.port;
      new_port = udp->src_port;
      udp->checksum = 0;
    }
  else /* HGNAT_PROTO_ICMP */
    {
      u16 old_id, new_id;
      old_id = echo->identifier;
      new_id = s->out2in.port;
      echo->identifier = new_id;

      sum = icmp->checksum;
      sum = ip_csum_update (sum, old_id, new_id, icmp_echo_header_t,
                            identifier);
      icmp->checksum = ip_csum_fold (sum);
    }

  translated++;

  s->last_heard = hgnat_in2out_now;
  s->total_pkts ++;
  s->total_bytes += vlib_buffer_length_in_chain (hgm->vlib_main, b);
  clib_dlist_remove (hgm->list_pool, s->global_lru_list_index);
  clib_dlist_addtail (hgm->list_pool, hgm->global_lru_list_head_index,
                      s->global_lru_list_index);

 trace:

  if (is_trace)
    {
      if (b->flags & VLIB_BUFFER_IS_TRACED)
        {
          hgnat_in2out_trace_t *t =
            vlib_add_trace (vm, node, b, sizeof (*t));
          t->session_index = s ? s - hgm->session_pool : ~0;
          t->next_index = next;
          t->protocol = proto;
        }
    }

  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_IN2OUT_TRANSLATED, translated);
  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_IN2OUT_DROP, dropped);
  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_IN2OUT_NO_OUTSIDE_ADDR,
			       no_outside_address);
  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_IN2OUT_UNKNOWN_PROTOCOL,
			       unknown_protocol);
  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_IN2OUT_TTL_DROP, ttl_drops);
  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_IN2OUT_SESSIONS_ADDED,
			       sessions_added);
  return next;
}

static never_inline u32
last_stage (vlib_main_t *vm, vlib_node_runtime_t *node, vlib_buffer_t *b)
{
  if (PREDICT_FALSE (node->flags & VLIB_NODE_FLAG_TRACE))
    return in2out_inline (vm, node, b, 1 /* is_trace */ );
  else
    return in2out_inline (vm, node, b, 0 /* is_trace */ );
}

#define NSTAGES 2
#define stage0 generic_stage0

#include <vnet/pipeline.h>

VLIB_NODE_FN (in2out_node) (vlib_main_t * vm, vlib_node_runtime_t * node,
                            vlib_frame_t * frame)
{
  hgnat_main_t *hgm = &hgnat_main;
  hgnat_in2out_now = vlib_time_now(vm);
  ip4_address_t *addr;

  /* lookup the outside interface address */
  addr = ip4_interface_first_address (hgm->ip4_main,
                                      hgm->outside_sw_if_index, 0);
  hgm->outside_addr.as_u32 = addr ? addr->as_u32 : 0;

  return dispatch_pipeline(vm, node, frame);
}

/* *INDENT-OFF* */
#ifndef CLIB_MARCH_VARIANT
VLIB_REGISTER_NODE (in2out_node) =
{
  .name = "hgnat-in2out",
  .vector_size = sizeof (u32),
  .format_trace = format_hgnat_in2out_trace,
  .type = VLIB_NODE_TYPE_INTERNAL,

  .n_errors = ARRAY_LEN(hgnat_error_strings),
  .error_strings = hgnat_error_strings,

  .n_next_nodes = HGNAT_N_NEXT,

  /* edit / add dispositions here */
  .next_nodes = {
    [HGNAT_NEXT_LOOKUP] = "ip4-lookup",
    [HGNAT_NEXT_DROP] = "error-drop",
    [HGNAT_NEXT_ICMP_ERROR] = "ip4-icmp-error",
  },
};
#endif /*  CLIB_MARCH_VARIANT */

/* *INDENT-ON* */

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
