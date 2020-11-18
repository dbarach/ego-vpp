/*
 * tcptrack.c - skeleton vpp engine plug-in
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
#include <tcptrack/tcptrack.h>

#include <vlibapi/api.h>
#include <vlibmemory/api.h>
#include <vpp/app/version.h>
#include <stdbool.h>

#define REPLY_MSG_ID_BASE tmp->msg_id_base
#include <vlibapi/api_helper_macros.h>

tcptrack_main_t tcptrack_main;

/* Action function shared between message handler and debug CLI */

int tcptrack_enable_disable (tcptrack_main_t * tmp, u32 sw_if_index,
                                   int enable_disable)
{
  vlib_thread_main_t *tm = vlib_get_thread_main ();
  tcptrack_main_t *ttm = &tcptrack_main;
  vnet_sw_interface_t * sw;
  int i;
  int rv = 0;

  /* Utterly wrong? */
  if (pool_is_free_index (tmp->vnet_main->interface_main.sw_interfaces,
                          sw_if_index))
    return VNET_API_ERROR_INVALID_SW_IF_INDEX;

  /* Not a physical port? */
  sw = vnet_get_sw_interface (tmp->vnet_main, sw_if_index);
  if (sw->type != VNET_SW_INTERFACE_TYPE_HARDWARE)
    return VNET_API_ERROR_INVALID_SW_IF_INDEX;

  if (enable_disable)
    {
      clib_bihash_init2_args_8_8_t _a, *a = &_a;
      memset (a, 0, sizeof (*a));

      a->h = &ttm->ct4;
      a->name = "tcp order tracker";
      a->nbuckets = 65536;

      u32 num_threads = 1 + tm->n_threads;
      vec_validate (ttm->connections, num_threads);

      clib_bihash_init2_8_8 (a);
      vnet_feature_enable_disable ("ip4-unicast", "tcptrack",
                                   sw_if_index, enable_disable, 0, 0);

      tcptrack_create_periodic_process (ttm);

      /* Send an event to enable/disable the periodic scanner process */
      vlib_process_signal_event (ttm->vlib_main,
                                 ttm->periodic_node_index,
                                 TCPTRACK_EVENT_PERIODIC_ENABLE_DISABLE,
                                 (uword)enable_disable);
    }
  else
    {
      /* Send an event to enable/disable the periodic scanner process */
      vlib_process_signal_event (ttm->vlib_main,
                                 ttm->periodic_node_index,
                                 TCPTRACK_EVENT_PERIODIC_ENABLE_DISABLE,
                                 (uword)enable_disable);

      vnet_feature_enable_disable ("ip4-unicast", "tcptrack",
                                   sw_if_index, enable_disable, 0, 0);
      clib_bihash_free_8_8 (&ttm->ct4);
      for (i = 0; i < vec_len (ttm->connections); i++)
          pool_free (ttm->connections[i]);
    }

  return rv;
}

static clib_error_t *
tcptrack_enable_disable_command_fn (vlib_main_t * vm,
                                   unformat_input_t * input,
                                   vlib_cli_command_t * cmd)
{
  tcptrack_main_t * tmp = &tcptrack_main;
  u32 sw_if_index = ~0;
  int enable_disable = 1;

  int rv;

  while (unformat_check_input (input) != UNFORMAT_END_OF_INPUT)
    {
      if (unformat (input, "disable"))
        enable_disable = 0;
      else if (unformat (input, "%U", unformat_vnet_sw_interface,
                         tmp->vnet_main, &sw_if_index))
        ;
      else
        break;
  }

  if (sw_if_index == ~0)
    return clib_error_return (0, "Please specify an interface...");

  rv = tcptrack_enable_disable (tmp, sw_if_index, enable_disable);

  switch(rv)
    {
  case 0:
    break;

  case VNET_API_ERROR_INVALID_SW_IF_INDEX:
    return clib_error_return
      (0, "Invalid interface, only works on physical ports");
    break;

  case VNET_API_ERROR_UNIMPLEMENTED:
    return clib_error_return (0, "Device driver doesn't support redirection");
    break;

  default:
    return clib_error_return (0, "tcptrack_enable_disable returned %d",
                              rv);
    }
  return 0;
}

/* *INDENT-OFF* */
VLIB_CLI_COMMAND (tcptrack_enable_disable_command, static) =
{
  .path = "tcptrack enable-disable",
  .short_help =
  "tcptrack enable-disable <interface-name> [disable]",
  .function = tcptrack_enable_disable_command_fn,
};
/* *INDENT-ON* */


static clib_error_t *
tcptrack_show_command_fn (vlib_main_t * vm,
                                   unformat_input_t * input,
                                   vlib_cli_command_t * cmd)
{
  tcptrack_main_t *ttm = &tcptrack_main;
  tcp_conn_t *tc;
  int verbose = 0;
  int detail = 0;
  f64 now = vlib_time_now(vm);
  u32 in_order_pkts=0, out_of_order_pkts=0;
  int i;

  if (unformat (input, "verbose %d", &verbose))
    ;
  else if (unformat (input, "verbose"))
    verbose = 1;
  else if (unformat (input, "detail"))
    detail = 1;

  for (i = 0; i < vec_len (ttm->connections); i++)
    {
      /* *INDENT-OFF* */
      pool_foreach (tc, ttm->connections[i],
      ({
        in_order_pkts += tc->in_order_pkts;
        out_of_order_pkts += tc->out_of_order_pkts;
        if (verbose || (detail && tc->in_order_pkts) ||
            (tc->in_order_pkts + tc->out_of_order_pkts > 1000))
          {
            vlib_cli_output (vm, "%U:%d ooo rate %.4f ooo %d total %d last heard %.3fs ago",
                             format_ip4_address, &tc->src_address,
                             tc->dst_port,
                             (f64)(tc->out_of_order_pkts) /
                             (f64)(tc->out_of_order_pkts + tc->in_order_pkts),
                             tc->out_of_order_pkts,
                             (tc->out_of_order_pkts + tc->in_order_pkts),
                             now - tc->last_heard);
          }}));
      /* *INDENT-ON* */
    }

  vlib_cli_output (vm, "IP4 connection tracking hash table:");
  vlib_cli_output (vm, "%U", format_bihash_8_8, &ttm->ct4, verbose);
  return 0;
}

/* *INDENT-OFF* */
VLIB_CLI_COMMAND (tcptrack_show_command, static) =
{
  .path = "show tcptrack",
  .short_help = "show tcptrack [verbose [nn]] [detail]",
  .function = tcptrack_show_command_fn,
};
/* *INDENT-ON* */


static clib_error_t * tcptrack_init (vlib_main_t * vm)
{
  tcptrack_main_t * tmp = &tcptrack_main;
  clib_error_t * error = 0;

  tmp->vlib_main = vm;
  tmp->vnet_main = vnet_get_main();

  /* Add our API messages to the global name_crc hash table */
  // tmp->msg_id_base = setup_message_id_table ();

  return error;
}

VLIB_INIT_FUNCTION (tcptrack_init);

/* *INDENT-OFF* */
VNET_FEATURE_INIT (tcptrack, static) =
{
  .arc_name = "ip4-unicast",
  .node_name = "tcptrack",
  .runs_before = VNET_FEATURES ("nat44-out2in-fast"),
};
/* *INDENT-ON */

/* *INDENT-OFF* */
VLIB_PLUGIN_REGISTER () =
{
  .version = VPP_BUILD_VER,
  .description = "TCP sequence number tracker",
};
/* *INDENT-ON* */

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
