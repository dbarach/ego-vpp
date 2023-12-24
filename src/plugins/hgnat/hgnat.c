/*
 * hgnat.c - skeleton vpp engine plug-in
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

#include <vnet/vnet.h>
#include <vnet/plugin/plugin.h>
#include <hgnat/hgnat.h>

#include <vlibapi/api.h>
#include <vlibmemory/api.h>
#include <vpp/app/version.h>
#include <stdbool.h>

#include <hgnat/hgnat.api_enum.h>
#include <hgnat/hgnat.api_types.h>

#define REPLY_MSG_ID_BASE hgm->msg_id_base
#include <vlibapi/api_helper_macros.h>

hgnat_main_t hgnat_main;

char *hgnat_protocol_strings[] =
  {
    "udp",
    "tcp",
    "icmp",
    "unknown",
  };

/* API message handler */
static void vl_api_hgnat_periodic_enable_disable_t_handler
(vl_api_hgnat_periodic_enable_disable_t * mp)
{
  vl_api_hgnat_periodic_enable_disable_reply_t * rmp;
  hgnat_main_t * hgm = &hgnat_main;
  int rv=0;

  REPLY_MACRO(VL_API_HGNAT_PERIODIC_ENABLE_DISABLE_REPLY);
}

static clib_error_t *
hgnat_config (vlib_main_t * vm, unformat_input_t * input)
{
  hgnat_main_t * hgm = &hgnat_main;

  uword hash_buckets = HGNAT_HASH_BUCKET_DEFAULT;
  uword hash_memory_size = HGNAT_HASH_MEMORY_DEFAULT;
  f64 translation_timeout = HGNAT_TRANSLATION_TIMEOUT_DEFAULT;
  uword translation_reclaim_if_expired = HGNAT_TRANSLATION_RECLAIM_IF_EXPIRED_DEFAULT;
  uword translation_limit = HGNAT_TRANSLATION_LIMIT_DEFAULT;

  while (unformat_check_input (input) != UNFORMAT_END_OF_INPUT)
    {
      if (unformat (input, "hash buckets %d", &hash_buckets))
        ;
      else if (unformat (input, "translation memory size %U",
                         unformat_memory_size, &hash_memory_size))
        ;
      else if (unformat (input, "translation timeout %f",
                         &translation_timeout))
        ;
      else if (unformat (input, "translation limit %d",
                         &translation_limit))
        ;
      else if (unformat (input, "translation reclaim expired %d",
                         &translation_reclaim_if_expired))
        ;
      else
	return clib_error_return (0, "unknown input '%U'",
				  format_unformat_error, input);
    }

  /* for show commands, etc. */
  hgm->hash_buckets = hash_buckets;
  hgm->hash_memory_size = hash_memory_size;
  hgm->translation_timeout = translation_timeout;
  hgm->translation_reclaim_if_expired = translation_reclaim_if_expired;
  hgm->translation_limit = translation_limit;
  return 0;
}

VLIB_CONFIG_FUNCTION (hgnat_config, "hgnat");


/* API definitions */
#include <hgnat/hgnat.api.c>

static clib_error_t * hgnat_init (vlib_main_t * vm)
{
  hgnat_main_t * hgm = &hgnat_main;
  clib_error_t * error = 0;

  hgm->vlib_main = vm;
  hgm->vnet_main = vnet_get_main();
  hgm->ip4_main = &ip4_main;

  hgm->msg_id_base = setup_message_id_table ();

  hgm->hash_buckets = HGNAT_HASH_BUCKET_DEFAULT;
  hgm->hash_memory_size = HGNAT_HASH_MEMORY_DEFAULT;
  hgm->translation_timeout = HGNAT_TRANSLATION_TIMEOUT_DEFAULT;
  hgm->translation_reclaim_if_expired = HGNAT_TRANSLATION_RECLAIM_IF_EXPIRED_DEFAULT;
  hgm->translation_limit = HGNAT_TRANSLATION_LIMIT_DEFAULT;
  return error;
}

VLIB_INIT_FUNCTION (hgnat_init);

/* *INDENT-OFF* */
/* Hook up input features */
VNET_FEATURE_INIT (ip4_hgnat_in2out, static) = {
  .arc_name = "ip4-unicast",
  .node_name = "hgnat-in2out",
  .runs_before = VNET_FEATURES ("hgnat-out2in"),
};
VNET_FEATURE_INIT (ip4_hgnat_out2in, static) = {
  .arc_name = "ip4-unicast",
  .node_name = "hgnat-out2in",
  .runs_before = VNET_FEATURES ("ip4-lookup"),
};
/* *INDENT-ON */

static int hgnat_interface_add_del (u32 sw_if_index, u8 is_inside, int is_del)
{
  if (is_inside)
    vnet_feature_enable_disable ("ip4-unicast", "hgnat-in2out", sw_if_index,
                                 !is_del, 0, 0);
  else
    vnet_feature_enable_disable ("ip4-unicast", "hgnat-out2in", sw_if_index,
                                 !is_del, 0, 0);
  return 0;
}


static clib_error_t *
hgnat_feature_command_fn (vlib_main_t * vm,
                          unformat_input_t * input,
                          vlib_cli_command_t * cmd)
{
  unformat_input_t _line_input, *line_input = &_line_input;
  hgnat_main_t * hgm = &hgnat_main;
  vnet_main_t * vnm = vnet_get_main();
  clib_error_t * error = 0;
  u32 sw_if_index;
  u32 * inside_sw_if_indices = 0;
  u32 * outside_sw_if_indices = 0;
  int is_del = 0;
  int i;

  sw_if_index = ~0;

  /* Get a line of input. */
  if (!unformat_user (input, unformat_line_input, line_input))
    return 0;

  while (unformat_check_input (line_input) != UNFORMAT_END_OF_INPUT)
    {
      if (unformat (line_input, "in %U", unformat_vnet_sw_interface,
                    vnm, &sw_if_index))
        vec_add1 (inside_sw_if_indices, sw_if_index);
      else if (unformat (line_input, "out %U", unformat_vnet_sw_interface,
                         vnm, &sw_if_index))
        vec_add1 (outside_sw_if_indices, sw_if_index);
      else if (unformat (line_input, "del"))
        is_del = 1;
      else
        return clib_error_return (0, "unknown input '%U'",
          format_unformat_error, input);
    }
  unformat_free (line_input);

  if (vec_len (inside_sw_if_indices))
    {
      for (i = 0; i < vec_len(inside_sw_if_indices); i++)
        {
          sw_if_index = inside_sw_if_indices[i];
          hgnat_interface_add_del (sw_if_index, 1 /* inside */, is_del);
        }
    }

  if (vec_len (outside_sw_if_indices))
    {
      for (i = 0; i < vec_len(outside_sw_if_indices); i++)
        {
          sw_if_index = outside_sw_if_indices[i];
          hgm->outside_sw_if_index = sw_if_index;
          hgnat_interface_add_del (sw_if_index, 0 /* outside */, is_del);
        }
    }

  vec_free (inside_sw_if_indices);
  vec_free (outside_sw_if_indices);

  if (error == 0)
    {
      if (hgm->table_init_done == 0)
        {
          dlist_elt_t *lru_listhead;
          clib_bihash_init_8_8 (&hgm->in2out, "hgnat-in2out", hgm->hash_buckets,
                                hgm->hash_memory_size);
          clib_bihash_init_8_8 (&hgm->out2in, "hgnat-out2in", hgm->hash_buckets,
                                hgm->hash_memory_size);
          pool_get (hgm->list_pool, lru_listhead);
          clib_dlist_init (hgm->list_pool, lru_listhead - hgm->list_pool);
          hgm->global_lru_list_head_index = lru_listhead - hgm->list_pool;
          hgm->table_init_done = 1;
        }
    }

  return error;
}

VLIB_CLI_COMMAND (set_interface_hgnat_command, static) = {
  .path = "set interface hgnat",
  .function = hgnat_feature_command_fn,
  .short_help = "set interface hgnat in <intfc> out <intfc> [del]",
};

static clib_error_t *
show_hgnat_command_fn (vlib_main_t * vm,
                       unformat_input_t * input,
                       vlib_cli_command_t * cmd)
{
  hgnat_main_t * hgm = &hgnat_main;
  unformat_input_t _line_input, *line_input = &_line_input;
  int verbose = 0;
  int sessions = 0;
  int tables = 0;
  int static_sessions = 0;
  clib_error_t * error = 0;
  u32 one_session = ~0;
  hgnat_session_t *s;
  f64 now = vlib_time_now(vm);

  /* Get a line of input. */
  if (!unformat_user (input, unformat_line_input, line_input))
    goto show_something;

  while (unformat_check_input (line_input) != UNFORMAT_END_OF_INPUT)
    {
      if (unformat(line_input, "tables"))
        tables = 1;
      else if (unformat(line_input, "sessions"))
        sessions = 1;
      else if (unformat (line_input, "session %d", &one_session))
	;
      else if (unformat (line_input, "verbose %d", &verbose))
        ;
      else if (unformat (line_input, "verbose"))
        verbose = 1;
      else if (unformat (line_input, "static"))
        static_sessions = 1;
      else
        return clib_error_return (0, "unknown input '%U'",
                                  format_unformat_error, input);
    }
  unformat_free (line_input);

  /* Always show something... */
 show_something:
  vlib_cli_output (vm, "%d active hgnat sessions",
		   pool_elts (hgm->session_pool));

  if (one_session != ~0)
    {
      if (pool_is_free_index (hgm->session_pool, one_session))
	return clib_error_return (0, "Session %d is inactive", one_session);

      s = pool_elt_at_index (hgm->session_pool, one_session);

      vlib_cli_output (vm, "Last heard %f seconds ago",
		       now - s->last_heard);
      vlib_cli_output (vm, "%d packets %d bytes",
		       s->total_pkts, s->total_bytes);
      vlib_cli_output (vm, "out2in %U port %d proto %s",
		       format_ip4_address, &s->out2in.addr,
		       clib_net_to_host_u16 (s->out2in.port),
		       hgnat_protocol_strings[s->out2in.protocol]);
      vlib_cli_output (vm, "in2out %U port %d proto %s",
		       format_ip4_address, &s->in2out.addr,
		       clib_net_to_host_u16 (s->in2out.port),
		       hgnat_protocol_strings[s->in2out.protocol]);
    }

  if (sessions)
    {
      vlib_cli_output (vm, "%d reclaims", hgm->reclaims);
      if (hgm->active_translations != pool_elts (hgm->session_pool))
        vlib_cli_output (vm, "WARNING: %ld active translations != %d pool elts",
                         hgm->active_translations,
                         pool_elts (hgm->session_pool));
      if (verbose)
        {
          vlib_cli_output (vm, "%s", static_sessions ? "Static Sessions" :
                           "All Sessions");
          /* *INDENT-OFF* */
          pool_foreach (s, hgm->session_pool)
            {
              if (static_sessions && (s->global_lru_list_index != ~0))
                continue;

	      vlib_cli_output (vm, "Index %d, last heard %f seconds ago",
			       s - hgm->session_pool,
			       now - s->last_heard);
	      vlib_cli_output (vm, "%d packets %d bytes",
			       s->total_pkts, s->total_bytes);
              vlib_cli_output (vm, "out2in %U port %d proto %s",
                               format_ip4_address, &s->out2in.addr,
                               clib_net_to_host_u16 (s->out2in.port),
                               hgnat_protocol_strings[s->out2in.protocol]);
              vlib_cli_output (vm, "in2out %U port %d proto %s",
                               format_ip4_address, &s->in2out.addr,
                               clib_net_to_host_u16 (s->in2out.port),
                               hgnat_protocol_strings[s->in2out.protocol]);
	      vlib_cli_output (vm, "----");

            }
          /* *INDENT-OFF* */
        }
    }
  if (tables)
    {
      vlib_cli_output (vm, "in2out table: %U", format_bihash_8_8,
                       &hgm->in2out, verbose);
      vlib_cli_output (vm, "out2in table: %U", format_bihash_8_8,
                       &hgm->out2in, verbose);
    }
  return error;
}

VLIB_CLI_COMMAND (show_hgnat_command, static) = {
  .path = "show hgnat",
  .function = show_hgnat_command_fn,
  .short_help = "show hgnat [sessions] [tables] [static] [[verbose] nn]",
};

static clib_error_t *
clear_hgnat_command_fn (vlib_main_t * vm,
                        unformat_input_t * input,
                        vlib_cli_command_t * cmd)
{
  hgnat_main_t * hgm = &hgnat_main;
  unformat_input_t _line_input, *line_input = &_line_input;
  int clear_all = 0;

  /* Get a line of input. */
  if (!unformat_user (input, unformat_line_input, line_input))
    goto clear_expired;

  while (unformat_check_input (line_input) != UNFORMAT_END_OF_INPUT)
    {
      if (unformat(line_input, "expired"))
        clear_all = 0;
      else if (unformat(line_input, "all"))
        clear_all = 1;
      else
        return clib_error_return (0, "unknown input '%U'",
                                  format_unformat_error, input);
    }
  unformat_free (line_input);

 clear_expired:
  hgnat_maybe_reclaim_translations(hgm, clear_all);
  return 0;
}

VLIB_CLI_COMMAND (clear_hgnat_command, static) = {
  .path = "clear hgnat",
  .function = clear_hgnat_command_fn,
  .short_help = "clear hgnat [expired] | [all]",
};

static clib_error_t *
set_hgnat_command_fn (vlib_main_t * vm,
                      unformat_input_t * input,
                      vlib_cli_command_t * cmd)
{
  hgnat_main_t * hgm = &hgnat_main;
  unformat_input_t _line_input, *line_input = &_line_input;
  u16 port = ~0;
  u32 protocol = HGNAT_PROTOCOL_UNKNOWN;
  ip4_address_t addr;
  int addr_set = 0;
  int is_add = 1;
  u32 tmp;
  hgnat_session_key_t key;
  hgnat_session_t * s;
  clib_bihash_kv_8_8_t kv, value;



  /* Get a line of input. */
  if (!unformat_user (input, unformat_line_input, line_input))
    return clib_error_return (0, "no input");

  while (unformat_check_input (line_input) != UNFORMAT_END_OF_INPUT)
    {
      if (unformat(line_input, "port %d", &tmp))
        {
          port = tmp;
          if (port < 1 || port >= 65535)
            return clib_error_return (0, "Invalid port number %d", port);
          port = clib_host_to_net_u16 (port);
        }
      else if (unformat(line_input, "proto tcp"))
        protocol = HGNAT_PROTOCOL_TCP;
      else if (unformat (line_input, "proto udp"))
        protocol = HGNAT_PROTOCOL_UDP;
      else if (unformat (line_input, "del"))
        is_add = 0;
      else if (unformat (line_input, "dst %U",
                         unformat_ip4_address, &addr))
        addr_set = 1;
      else
        return clib_error_return (0, "unknown input '%U'",
                                  format_unformat_error, input);
    }
  unformat_free (line_input);

  if (port == 0xFFFF)
    return clib_error_return (0, "outside port must be specified");
  if (addr_set == 0)
    return clib_error_return (0, "inside address must be specified");
  if (protocol == HGNAT_PROTOCOL_UNKNOWN)
    return clib_error_return (0, "protocol must be specified");

  /* Add the requested translation to the out2in table */
  key.as_u64 = 0;
  /* Add key.addr as 0.0.0.0, so the src address doesn't matter */
  key.port = port;
  key.protocol = protocol;
  kv.key = key.as_u64;

  if (is_add)
    {
      pool_get (hgm->session_pool, s);
      memset (s, 0, sizeof (*s));
      kv.value = s - hgm->session_pool;

      clib_bihash_add_del_8_8 (&hgm->out2in, &kv, 1 /* is_add */);
      /*
       * Save the statically-mapped dst address in the session
       * No LRU list, keep these away from the ager.
       * No in2out mapping at all. The resultant e.g. TCP SYN-ACK will
       * nail up a per-session out2in mapping. We hope.
       */
      key.addr = addr;
      s->out2in = key;
      /* No LRU list action for static sessions */
      s->global_lru_list_index = ~0;
      hgm->active_translations ++;
    }
  else
    {
      kv.key = key.as_u64;
      if (clib_bihash_search_8_8 (&hgm->out2in, &kv, &value))
        return clib_error_return (0,
                                  "Couldn't find mapping for port %d proto %s",
                                  clib_host_to_net_u16 (port),
                                  (protocol == HGNAT_PROTOCOL_UDP)
                                  ? "udp" : "tcp");
      s = pool_elt_at_index (hgm->session_pool, value.value);
      clib_bihash_add_del_8_8 (&hgm->out2in, &kv, 0 /* is_add */);
      pool_put (hgm->session_pool, s);
      hgm->active_translations --;
      hgm->reclaims ++;
    }
  return 0;
}

VLIB_CLI_COMMAND (set_hgnat_command, static) = {
  .path = "set hgnat static",
  .function = set_hgnat_command_fn,
  .short_help = "set hgnat static port NN proto [tcp | udp] dst <ip4> [del]"
};

#if 0
/* Horrifying checksum ordeal */
static u8 csum_test[68] =
{
    0x03, 0x03, 0 /* 0x5b */, 0 /*0xad*/, 0x00, 0x00, 0x00, 0x00,
    0x45, 0x00, 0x00, 0x3c, 0xd5, 0xc8, 0x00, 0x00,
    0x02, 0x11, 0xd2, 0xf0, 0x0a, 0x1a, 0x44, 0x2c,
    0xc0, 0xa8, 0x01, 0x0a, 0xd8, 0x37, 0x82, 0xa1,
    0x00, 0x28, 0x00, 0x00, 0x40, 0x41, 0x42, 0x43,
    0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b,
    0x4c, 0x4d, 0x4e, 0x4f, 0x50, 0x51, 0x52, 0x53,
    0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x5b,
    0x5c, 0x5d, 0x5e, 0x5f,
};


static clib_error_t *test_hgnat_command_fn (vlib_main_t * vm,
                       unformat_input_t * input,
                       vlib_cli_command_t * cmd)
{
  ip_csum_t sum = 0;
  u16 sum16;

  sum = ip_incremental_checksum (sum, csum_test, 68);
  sum16 = ~ip_csum_fold(sum);
  vlib_cli_output (vm, "checksum %x", clib_host_to_net_u16(sum16));
  return 0;
}

VLIB_CLI_COMMAND (test_hgnat_command, static) = {
  .path = "test hgnat",
  .function = test_hgnat_command_fn,
  .short_help = "test hgnat",
};
#endif




/* *INDENT-OFF* */
VLIB_PLUGIN_REGISTER () =
{
  .version = VPP_BUILD_VER,
  .description = "Home Gateway NAT44",
};
/* *INDENT-ON* */

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
