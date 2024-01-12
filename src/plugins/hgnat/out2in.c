/*
 * out2in.c - hgw out2in translation
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
#include <hgnat/hgnat.h>

static f64 hgnat_out2in_now;

typedef struct
{
  u32 protocol;
  u32 session_index;
  u32 next_index;
} hgnat_out2in_trace_t;

#ifndef CLIB_MARCH_VARIANT

/* packet trace format function */
static u8 * format_hgnat_out2in_trace (u8 * s, va_list * args)
{
  CLIB_UNUSED (vlib_main_t * vm) = va_arg (*args, vlib_main_t *);
  CLIB_UNUSED (vlib_node_t * node) = va_arg (*args, vlib_node_t *);
  hgnat_out2in_trace_t * t = va_arg (*args, hgnat_out2in_trace_t *);

  if (t->session_index != ~0)
    s = format (s, "hgnat_out2in: next index %d, session index %d, protocol %s",
                t->next_index, t->session_index,
                hgnat_protocol_strings[t->protocol]);
  else
    s = format (s, "hgnat_out2in: next index %d, no session, protocol %d",
                t->next_index, hgnat_protocol_strings[t->protocol]);
  return s;
}
vlib_node_registration_t out2in_node;
#endif /*  CLIB_MARCH_VARIANT */

#define foreach_hgnat_error \
_(HGNAT_OUT2IN_TRANSLATED, "Translated Packets") \
_(HGNAT_OUT2IN_DROP, "Dropped Packets")

typedef enum
{
#define _(sym,str) HGNAT_ERROR_##sym,
  foreach_hgnat_error
#undef _
  HGNAT_N_ERROR,
} hgnat_error_t;

#ifndef CLIB_MARCH_VARIANT
static char * hgnat_out2in_error_strings[] =
{
#define _(sym,string) string,
  foreach_hgnat_error
#undef _
};
#endif /*  CLIB_MARCH_VARIANT */

typedef enum
{
  HGNAT_OUT2IN_NEXT_DROP,
  HGNAT_OUT2IN_NEXT_LOOKUP,
  HGNAT_OUT2IN_N_NEXT,
} hgnat_next_t;

static never_inline hgnat_session_t *
out2in_slowpath (hgnat_main_t *hgm, vlib_buffer_t *b,
                 ip4_header_t *ip, udp_header_t *udp,
                 tcp_header_t *tcp,
                 icmp46_header_t *icmp, icmp_echo_header_t *echo,
                 u32 proto)
{
  clib_bihash_kv_8_8_t kv, value;
  hgnat_session_key_t key;
  hgnat_session_t *s;
  u32 new_addr, old_addr;
  ip_csum_t sum;
  dlist_elt_t *lru;

  /*
   * This routine deals with special cases which arise when
   * the out2in hash lookup fails.
   */

  /*
   * Traceroute sends UDP datagrams with manipulated TTLs
   * which result in intentional icmp type 11 (ttl exceeded) code 0
   * errors. Before we turf the icmp, let's see if that's
   * what happened..
   */
  if (proto == HGNAT_PROTOCOL_ICMP)
    {
      ip4_header_t *inner_ip;
      udp_header_t *inner_udp;
      u8 *tmp;
      u16 sum16;
      /* Skip 4-octet pad after icmp header */
      tmp = (u8 *)(icmp+1);
      tmp += 4;

      inner_ip = (ip4_header_t *)(tmp);
      inner_udp = (udp_header_t *)(inner_ip+1);

      /* Try to find a translation for the original UDP packet */
      key.as_u64 = 0;
      key.addr = inner_ip->dst_address;
      key.port = inner_udp->src_port;
      key.protocol = HGNAT_PROTOCOL_UDP;
      kv.key = key.as_u64;
      if (clib_bihash_search_8_8 (&hgm->out2in, &kv, &value) == 0)
        {
          s = pool_elt_at_index (hgm->session_pool, value.value);

          /* Fix outer IP dst address and checksum */
          old_addr = ip->dst_address.as_u32;
          ip->dst_address = s->in2out.addr;
          new_addr = ip->dst_address.as_u32;
          sum = ip->checksum;
          sum = ip_csum_update (sum, old_addr, new_addr,
                                ip4_header_t,
                                dst_address /* changed member */);
          ip->checksum = ip_csum_fold (sum);

          /* Fix inner IP src address */
          old_addr = inner_ip->src_address.as_u32;
          inner_ip->src_address = s->in2out.addr;
          new_addr = inner_ip->src_address.as_u32;
          sum = inner_ip->checksum;
          sum = ip_csum_update (sum, old_addr, new_addr,
                                ip4_header_t,
                                src_address /* changed member */);

          inner_ip->checksum = ip_csum_fold (sum);

          /* Fix the inner IP dst address */
          old_addr = inner_ip->dst_address.as_u32;

          /* The inner ip dst must be the original dst address */
          inner_ip->dst_address.as_u32 = s->out2in.addr.as_u32;
          new_addr = inner_ip->dst_address.as_u32;
          sum = inner_ip->checksum;
          sum = ip_csum_update (sum, old_addr, new_addr,
                                ip4_header_t,
                                dst_address /* changed member */);
          inner_ip->checksum = ip_csum_fold (sum);

          /* Fix inner UDP datagram */
          inner_udp->src_port = s->orig_src_port;
          inner_udp->dst_port = s->orig_dst_port;

          inner_udp->checksum = 0;

          /* Fix the icmp checksum. */
          icmp->checksum = 0;
	  sum16 = hgnat_icmp_compute_checksum (hgm->vlib_main, b, ip);
	  icmp->checksum = sum16;
	  return s;
	}
    }

  /* Look for a static mapping */
  key.as_u64 = 0;
  /* addr == 0.0.0.0 for static mappings */

  if (ip->protocol == IP_PROTOCOL_TCP)
    key.protocol = HGNAT_PROTOCOL_TCP;
  else if (ip->protocol == IP_PROTOCOL_UDP)
    key.protocol = HGNAT_PROTOCOL_UDP;
  else /* e.g. unfixable ICMP, done */
    return 0;

  key.port = udp->dst_port;
  kv.key = key.as_u64;

  if (clib_bihash_search_8_8 (&hgm->out2in, &kv, &value) == 0)
    {
      s = pool_elt_at_index (hgm->session_pool, value.value);
      /* Note: the session out2in member has the desired inside address */
      old_addr = ip->dst_address.as_u32;
      ip->dst_address = s->out2in.addr;
      new_addr = ip->dst_address.as_u32;

      /* fix this packet (e.g. TCP SYN) so it will work. */
      sum = ip->checksum;
      sum = ip_csum_update (sum, old_addr, new_addr,
                            ip4_header_t,
                            dst_address /* changed member */);
      ip->checksum = ip_csum_fold (sum);

      if (ip->protocol == IP_PROTOCOL_TCP)
        {
          sum = tcp->checksum;
          sum = ip_csum_update (sum, old_addr, new_addr,
                                ip4_header_t,
                                dst_address /* changed member */);
          tcp->checksum = ip_csum_fold(sum);
        }
      else if (ip->protocol == IP_PROTOCOL_UDP)
        {
          udp->checksum = 0;
        }

      /* Set up the real mappings */
      key.as_u64 = 0;
      key.addr = ip->dst_address;
      key.port = udp->dst_port;
      key.protocol = ip->protocol == IP_PROTOCOL_TCP ? HGNAT_PROTOCOL_TCP :
        HGNAT_PROTOCOL_UDP;
      kv.key = key.as_u64;

      /* The mapping already exists. */
      if (clib_bihash_search_8_8 (&hgm->in2out, &kv, &value) == 0)
        {
	  s = pool_elt_at_index (hgm->session_pool, value.value);
          return s;
        }

      hgnat_maybe_reclaim_translations(hgm, 0 /* all */);
      /* No such translation (yet), add one */
      pool_get (hgm->session_pool, s);
      memset (s, 0, sizeof (*s));
      kv.value = s - hgm->session_pool;

      memset(s, 0, sizeof(*s));
      s->in2out = key;
      clib_bihash_add_del_8_8 (&hgm->in2out, &kv, 1 /* is_add */);

      /* Form out-to-in translation */
      key.addr = ip->src_address;
      key.port = udp->dst_port;
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
      return s;
    }
  return 0;
}


always_inline uword
out2in_inline (vlib_main_t * vm, vlib_node_runtime_t * node,
               vlib_buffer_t *b, int is_trace)
{
  hgnat_main_t *hgm = &hgnat_main;
  u32 translated = 0;
  u32 dropped = 0;
  u32 next;

  ip4_header_t * ip;
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
  u32 feature_next;

  /* Initial guess: send to next on feature arc */
  vnet_feature_next(&feature_next, b);
  next = feature_next;

  ip = vlib_buffer_get_current (b);
  udp = ip4_next_header (ip);
  tcp = (tcp_header_t *) udp;
  icmp = (icmp46_header_t *) udp;
  echo = (icmp_echo_header_t *)(icmp + 1);

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
      next = HGNAT_OUT2IN_NEXT_LOOKUP;
      goto trace;
    }
  s = 0; /* for trace */

  key.as_u64 = 0;
  key.addr = ip->src_address;
  key.port = (proto != HGNAT_PROTOCOL_ICMP) ?
    udp->dst_port : echo->identifier;
  key.protocol = proto;
  kv.key = key.as_u64;

  if (clib_bihash_search_8_8 (&hgm->out2in, &kv, &value))
    {
      s = out2in_slowpath (hgm, b, ip, udp, tcp, icmp, echo, proto);

      if (s)
        goto send_packet;

      if (0)
        clib_warning("Couldn't find addr %U port %d",
                     format_ip4_address, &ip->src_address,
                     clib_net_to_host_u16(key.port));
      /* No such translation, packet goes to the shitter */
      next = HGNAT_OUT2IN_NEXT_DROP;
      dropped++;
      goto trace;
    }
  else
    s = pool_elt_at_index (hgm->session_pool, value.value);

  old_addr = ip->dst_address.as_u32;
  ip->dst_address = s->in2out.addr;
  new_addr = ip->dst_address.as_u32;

  sum = ip->checksum;
  sum = ip_csum_update (sum, old_addr, new_addr,
                        ip4_header_t,
                        dst_address /* changed member */);
  ip->checksum = ip_csum_fold (sum);

  if (PREDICT_TRUE(proto == HGNAT_PROTOCOL_TCP))
    {
      old_port = tcp->dst_port;
      tcp->dst_port = s->in2out.port;
      new_port = tcp->dst_port;

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
      old_port = udp->dst_port;
      udp->dst_port = s->in2out.port;
      udp->checksum = 0;
    }
  else /* HGNAT_PROTO_ICMP */
    {
      u16 old_id, new_id;

      old_id = echo->identifier;
      new_id = s->in2out.port;
      echo->identifier = new_id;

      sum = icmp->checksum;
      sum = ip_csum_update (sum, old_id, new_id, icmp_echo_header_t,
                            identifier);
      icmp->checksum = ip_csum_fold (sum);
    }

 send_packet:
  s->last_heard = hgnat_out2in_now;
  s->total_pkts ++;
  s->total_bytes += vlib_buffer_length_in_chain (hgm->vlib_main, b);
  /* Static sessions don't have LRU list elements */
  ASSERT (s->global_lru_list_index != ~0);
  clib_dlist_remove (hgm->list_pool, s->global_lru_list_index);
  clib_dlist_addtail (hgm->list_pool, hgm->global_lru_list_head_index,
                      s->global_lru_list_index);
  translated++;

 trace:

  if (is_trace)
    {
      if (b->flags & VLIB_BUFFER_IS_TRACED)
        {
          hgnat_out2in_trace_t *t =
            vlib_add_trace (vm, node, b, sizeof (*t));
          t->protocol = proto;
          t->session_index = s ? s - hgm->session_pool : ~0;
          t->next_index = next;
        }
    }

  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_OUT2IN_TRANSLATED, translated);
  vlib_node_increment_counter (vm, node->node_index,
			       HGNAT_ERROR_HGNAT_OUT2IN_DROP, dropped);
  return next;
}

static never_inline u32
last_stage (vlib_main_t *vm, vlib_node_runtime_t *node, vlib_buffer_t *b)
{
  if (PREDICT_FALSE (node->flags & VLIB_NODE_FLAG_TRACE))
    return out2in_inline (vm, node, b, 1 /* is_trace */ );
  else
    return out2in_inline (vm, node, b, 0 /* is_trace */ );
}

#define NSTAGES 2
#define stage0 generic_stage0

#include <vnet/pipeline.h>

VLIB_NODE_FN (out2in_node) (vlib_main_t * vm, vlib_node_runtime_t * node,
                            vlib_frame_t * frame)
{
  hgnat_out2in_now = vlib_time_now(vm);
  return dispatch_pipeline(vm, node, frame);
}

/* *INDENT-OFF* */
#ifndef CLIB_MARCH_VARIANT
VLIB_REGISTER_NODE (out2in_node) =
{
  .name = "hgnat-out2in",
  .vector_size = sizeof (u32),
  .format_trace = format_hgnat_out2in_trace,
  .type = VLIB_NODE_TYPE_INTERNAL,

  .n_errors = ARRAY_LEN(hgnat_out2in_error_strings),
  .error_strings = hgnat_out2in_error_strings,

  .n_next_nodes = HGNAT_OUT2IN_N_NEXT,

  /* edit / add dispositions here */
  .next_nodes = {
    [HGNAT_OUT2IN_NEXT_LOOKUP] = "ip4-lookup",
    [HGNAT_OUT2IN_NEXT_DROP] = "error-drop",
  },
};
#endif /*  CLIB_MARCH_VARIANT */

#ifndef CLIB_MARCH_VARIANT
/* Dummy init function to force the linker to include this file */
static clib_error_t *
hgnat_dummy_init (vlib_main_t * vm)
{
  return 0;
}

VLIB_INIT_FUNCTION (hgnat_dummy_init);
#endif /* CLIB_MARCH_VARIANT */


/* *INDENT-ON* */
/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
