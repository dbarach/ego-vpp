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
  __os_thread_index = 0;

  /* Bootstrap the ego runtime here... */
  OsInit();

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

void ego_start_thread (void)
{
  ego_g2_fp = (void *) Vpp_start_g2;
  pthread_cond_signal(&g2_thread_cv);
}

#include <sys/types.h>

int ego_gtk_start_check (void)
{
  extern int getresuid (uid_t *__ruid, uid_t *euid, uid_t *__suid);
  extern int getresgid (uid_t *__ruid, uid_t *euid, uid_t *__suid);
  uid_t ruid, euid, suid; /* Real, effective and saved user ID's */
  gid_t rgid, egid, sgid; /* Real, effective and saved group ID's */

  if (getresuid (&ruid, &euid, &suid) != 0 ||
      getresgid (&rgid, &egid, &sgid) != 0)
    {
      suid = ruid = getuid ();
      sgid = rgid = getgid ();
      euid = geteuid ();
      egid = getegid ();
    }

  /* gtk certainly won't start... */
  if (ruid != euid || ruid != suid ||
      rgid != egid || rgid != sgid)
    {
      return (1);
    }

  /* Some chance this might work... */
  (void) setenv ("DISPLAY", ":1", 0 /* don't overwrite */);

  /* gtk may start, e.g. if $DISPLAY is OK, etc. */
  return (0);
}

static clib_error_t *
event_log_viewer_command_fn (vlib_main_t * vm,
			  unformat_input_t * input, vlib_cli_command_t * cmd)
{
  vlib_cli_output (vm, "Starting the event log viewer...");
  ego_start_thread();
  return 0;
}

/* *INDENT-OFF* */
VLIB_CLI_COMMAND (event_log_viewer_command, static) =
{
  .path = "event-logger viewer",
  .short_help = "Start the graphical event viewer",
  .function = event_log_viewer_command_fn,
  .is_mp_safe = 1,
};
/* *INDENT-ON* */

/*
 * fd.io coding-style-patch-verification: ON
 *
 * Local Variables:
 * eval: (c-set-style "gnu")
 * End:
 */
