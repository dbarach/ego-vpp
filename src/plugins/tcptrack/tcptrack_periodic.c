/*
 * tcptrack_periodic.c - skeleton plug-in periodic function
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
#include <vppinfra/error.h>
#include <tcptrack/tcptrack.h>

static void
handle_periodic_enable_disable (tcptrack_main_t *pm, f64 now, uword event_data)
{
   pm->periodic_timer_enabled = event_data;
}

static void
handle_timeout (tcptrack_main_t *ttm, f64 now)
{
  static u32 *timed_out_indices;
  tcp_conn_t *tc;
  int i, j;

  vlib_worker_thread_barrier_sync (ttm->vlib_main);

  for (i = 0; i < vec_len (ttm->connections); i++)
    {
      vec_reset_length (timed_out_indices);

      /* *INDENT-OFF* */
      pool_foreach (tc, ttm->connections[i],
      ({
        if (now - tc->last_heard > 60.0)
          vec_add1 (timed_out_indices, tc - ttm->connections[i]);
      }));
      /* *INDENT-ON* */

      for (j = 0; j < vec_len (timed_out_indices); j++)
        {
          clib_bihash_kv_8_8_t kv;

          tc = pool_elt_at_index (ttm->connections[i], timed_out_indices[j]);

          kv.key = ((u64)tc->src_address << 32) | tc->dst_port;

          if (clib_bihash_add_del_8_8 (&ttm->ct4, &kv, 0 /* is_add */))
            clib_warning ("delete fail?");

          pool_put_index (ttm->connections[i], timed_out_indices[j]);
        }
    }

  vlib_worker_thread_barrier_release (ttm->vlib_main);
}

static uword
tcptrack_periodic_process (vlib_main_t * vm,
	                  vlib_node_runtime_t * rt, vlib_frame_t * f)
{
  tcptrack_main_t *pm = &tcptrack_main;
  f64 now;
  f64 timeout = 17.0;
  uword *event_data = 0;
  uword event_type;
  int i;

  while (1)
    {
      if (pm->periodic_timer_enabled)
        vlib_process_wait_for_event_or_clock (vm, timeout);
      else
        vlib_process_wait_for_event (vm);

      now = vlib_time_now (vm);

      event_type = vlib_process_get_events (vm, (uword **) & event_data);

      switch (event_type)
	{
	case TCPTRACK_EVENT_PERIODIC_ENABLE_DISABLE:
	  for (i = 0; i < vec_len (event_data); i++)
	    handle_periodic_enable_disable (pm, now, event_data[i]);
	  break;

          /* Handle periodic timeouts */
	case ~0:
	  handle_timeout (pm, now);
	  break;
	}
      vec_reset_length (event_data);
    }
  return 0;			/* or not */
}

void tcptrack_create_periodic_process (tcptrack_main_t *tmp)
{
  /* Already created the process node? */
  if (tmp->periodic_node_index > 0)
    return;

  /* No, create it now and make a note of the node index */
  tmp->periodic_node_index = vlib_process_create (tmp->vlib_main,
    "tcptrack-periodic-process",
    tcptrack_periodic_process, 16 /* log2_n_stack_bytes */);
}

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
