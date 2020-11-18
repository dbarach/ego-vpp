
/*
 * tcptrack.h - skeleton vpp engine plug-in header file
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
#ifndef __included_tcptrack_h__
#define __included_tcptrack_h__

#include <vnet/vnet.h>
#include <vnet/ip/ip.h>

#include <vppinfra/hash.h>
#include <vppinfra/error.h>
#include <vnet/ip/ip4_packet.h>
#include <vppinfra/bihash_8_8.h>
#include <vppinfra/bihash_template.h>

typedef struct
{
  u32 src_address;
  u32 dst_port;
  f64 last_heard;
  u32 next_seq_expected;
  u32 out_of_order_pkts;
  u32 in_order_pkts;
} tcp_conn_t;


typedef struct {
  tcp_conn_t **connections;

  /* connection tracking table */
  clib_bihash_8_8_t ct4;

  /* API message ID base */
  u16 msg_id_base;

  /* on/off switch for the periodic function */
  u8 periodic_timer_enabled;

  /* Node index, non-zero if the periodic process has been created */
  u32 periodic_node_index;

    /* convenience */
  vlib_main_t * vlib_main;
  vnet_main_t * vnet_main;
} tcptrack_main_t;

extern tcptrack_main_t tcptrack_main;

void tcptrack_create_periodic_process (tcptrack_main_t *tmp);

#define TCPTRACK_EVENT_PERIODIC_ENABLE_DISABLE 1

#endif /* __included_tcptrack_h__ */

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
