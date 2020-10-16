/*
 * g2.c - skeleton vpp engine plug-in
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

#include <pthread.h>
#include <vnet/vnet.h>
#include <vnet/plugin/plugin.h>
#include <vpp/app/version.h>
#include <g2/g2.h>

g2_main_t g2_main;

static volatile void *ego_g2_fp;
static pthread_mutex_t g2_thread_lock;
static pthread_cond_t g2_thread_cv;

void
ego_g2_thread_fn (void *arg)
{
  vlib_worker_thread_t *w = (vlib_worker_thread_t *) arg;
  vlib_cli_command_function_t *fp;

  vlib_worker_thread_init (w);
  clib_mem_set_heap (w->thread_mheap);
  pthread_mutex_init(&g2_thread_lock, NULL);
  pthread_cond_init(&g2_thread_cv, NULL);


  while (1) {
    pthread_mutex_lock (&g2_thread_lock);
    pthread_cond_wait (&g2_thread_cv, &g2_thread_lock);
    pthread_mutex_unlock (&g2_thread_lock);
    /* Should never happen... */
    if (ego_g2_fp == 0)
      continue;
    fp = ego_g2_fp;
    (*fp)(0, 0, 0);
    sleep(1);
    ego_g2_fp = 0;
  }
}

/* *INDENT-OFF* */
VLIB_REGISTER_THREAD (ego_g2_thread_reg, static) = {
  .name = "g2-thread",
  .short_name = "g2",
  .function = ego_g2_thread_fn,
  .use_pthreads = 1,
  .no_data_structure_clone = 1,
  .fixed_count = 1,
  .count = 1,
};
/* *INDENT-ON* */

void ego_start_thread (vlib_cli_command_function_t *fp)
{
  ego_g2_fp = (void *) fp;
  pthread_cond_signal(&g2_thread_cv);
}

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
