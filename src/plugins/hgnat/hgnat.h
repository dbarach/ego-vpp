
/*
 * hgnat.h - skeleton vpp engine plug-in header file
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
#ifndef __included_hgnat_h__
#define __included_hgnat_h__

#include <vnet/vnet.h>
#include <vnet/ip/ip.h>
#include <vnet/ethernet/ethernet.h>

#include <vppinfra/hash.h>
#include <vppinfra/error.h>
#include <vnet/api_errno.h>
#include <vppinfra/bihash_8_8.h>
#include <vppinfra/dlist.h>
#include <vppinfra/error.h>
#include <vlibapi/api.h>

typedef enum {
  HGNAT_PROTOCOL_UDP = 0,
  HGNAT_PROTOCOL_TCP,
  HGNAT_PROTOCOL_ICMP,
  HGNAT_PROTOCOL_UNKNOWN,
} hgnat_protocol_t;

typedef struct {
  union
  {
    struct
    {
      /* Fields in network byte order */
      ip4_address_t addr;
      u16 port;
      /* End fields in network byte order */
      u16 protocol:3,
      pad:13;
    };
    u64 as_u64;
  };
} hgnat_session_key_t;

typedef CLIB_PACKED(struct {
  hgnat_session_key_t out2in;
  hgnat_session_key_t in2out;

  u16 orig_src_port;
  u16 orig_dst_port;

  /* Last heard timer */
  f64 last_heard;

  u32 global_lru_list_index;

  u64 total_bytes;
  u32 total_pkts;
}) hgnat_session_t;

typedef struct {
  /* API message ID base */
  u16 msg_id_base;

  /* Non-zero if the periodic process has been created */
  u32 ager_process_node_index;

  /* convenience */
  vlib_main_t * vlib_main;
  vnet_main_t * vnet_main;
  ethernet_main_t * ethernet_main;
  ip4_main_t *ip4_main;

  /* Pool of doubly-linked list elements */
  dlist_elt_t * list_pool;
  u32 global_lru_list_head_index;

  /* "fast" path translation tables */
  clib_bihash_8_8_t in2out;
  clib_bihash_8_8_t out2in;
  u32 port_random_seed;
  u32 outside_sw_if_index;
  ip4_address_t outside_addr;
  hgnat_session_t *session_pool;

  /* config params */
  u32 hash_buckets;
  u32 hash_memory_size;
  u32 translation_reclaim_if_expired;
  u32 translation_limit;
  f64 translation_timeout;

  /* Stats */
  u64 active_translations;
  u64 reclaims;

  /* Table / ager process init complete */
  u8 table_init_done;
} hgnat_main_t;

typedef struct
{
  u16 identifier;
  u16 sequence;
} icmp_echo_header_t;


#define HGNAT_HASH_BUCKET_DEFAULT 8192
#define HGNAT_HASH_MEMORY_DEFAULT (128<<20)
#define HGNAT_TRANSLATION_TIMEOUT_DEFAULT 600.0 /* ten mins */
#define HGNAT_TRANSLATION_RECLAIM_IF_EXPIRED_DEFAULT                          \
  (2 << 10) /* 2K active                                                      \
	     */
#define HGNAT_TRANSLATION_LIMIT_DEFAULT (128<<10)

/* ager process event */
#define HGNAT_AGER_TRANSLATION_TIMEOUT_RESET 1

void hgnat_maybe_reclaim_translations (hgnat_main_t *hgm, int all);
u16 hgnat_icmp_compute_checksum (vlib_main_t *vm, vlib_buffer_t *p0,
				 ip4_header_t *ip0);

extern hgnat_main_t hgnat_main;
extern char *hgnat_protocol_strings[];
extern vlib_node_registration_t out2in_node;
extern vlib_node_registration_t in2out_node;

#endif /* __included_hgnat_h__ */

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
