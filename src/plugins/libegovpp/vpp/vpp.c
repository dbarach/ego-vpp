#include "vpp.h"

__clib_export u8 *ego_vec_string(void *ptr, uword len)
{
  u8 *r = 0;
  vec_add(r, ptr, len);
  return r;
}
__clib_export void ego_vec_free(void *ptr) { vec_free(ptr); }

__clib_export vlib_main_t * ego_vlib_get_main () { return vlib_get_main(); }
__clib_export vlib_node_t * ego_vlib_get_node (vlib_main_t * vm, u32 i) { return vlib_get_node(vm, i); }
__clib_export u32 ego_vlib_get_current_process_node_index (vlib_main_t * vm) { return vlib_get_current_process_node_index (vm); }
__clib_export vlib_node_state_t ego_vlib_node_get_state (vlib_main_t * vm, u32 node_index) { return vlib_node_get_state(vm, node_index); }
__clib_export void ego_vlib_node_set_state (vlib_main_t * vm, u32 node_index, vlib_node_state_t new_state) {
  vlib_node_set_state (vm, node_index, new_state);
}
__clib_export void ego_vlib_frame_args (vlib_frame_t * f, void **scalar, void **vector) {
  *scalar = f->scalar_offset != 0 ? vlib_frame_scalar_args (f) : 0;
  *vector = vlib_frame_vector_args (f);
}
__clib_export vlib_frame_t * ego_vlib_node_runtime_get_next_frame (vlib_main_t * vm, vlib_node_runtime_t * n, u32 next_index, u32 *ds) {
  vlib_next_frame_t * nf = vlib_node_runtime_get_next_frame (vm, n, next_index);
  vlib_frame_t * f = nf->frame;
  if (ds != NULL && f != NULL)
    *ds = f->deferment_state;
  return f;
}

__clib_export u32 ego_vlib_threads (u32 *n_threads) {
  if (n_threads != 0)
    *n_threads = os_get_nthreads();
  return vlib_get_thread_index();
}

__clib_export vnet_sw_interface_t *ego_vnet_get_sw_interface (vnet_main_t * vnm, u32 sw_if_index) { return vnet_get_sw_interface (vnm, sw_if_index); }
__clib_export vnet_sw_interface_flags_t ego_vnet_sw_interface_get_flags (vnet_main_t * vnm, u32 sw_if_index) { return vnet_sw_interface_get_flags (vnm, sw_if_index); }

// clang -shared -g -O0 -o /tmp/plugins/foo.so x.c -I/home/eliot/vpp/src -I/home/eliot/vpp/x/CMakeFiles
// run unix interactive plugins { add-path /tmp/plugins }

__clib_export void ego_add_init_func(int which, vlib_init_function_t * f, char *name, char **runs_before, char **runs_after, char **init_order) {
  vlib_global_main_t *vm = &vlib_global_main;
  _vlib_init_function_list_elt_t *l, **p = 0;
  switch (which) {
  case 0:
    p = &vm->init_function_registrations;
    break;
  case 1:
    p = &vm->main_loop_enter_function_registrations;
    break;
  case 2:
    p = &vm->main_loop_exit_function_registrations;
    break;
  case 3:
    p = &vm->worker_init_function_registrations;
    break;
  case 4:
    p = &vm->num_workers_change_function_registrations;
    break;
  case 5:
    p = &vm->api_init_function_registrations;
    break;
  }
  l = clib_mem_alloc (sizeof(l[0]));
  clib_memset (l, 0, sizeof (l[0]));
  l->f = f;
  l->name = name;
  l->runs_before = runs_before;
  l->runs_after = runs_after;
  l->init_order = init_order;
  l->next_init_function = *p;
  *p = l;
}

__clib_export void ego_add_config_func(vlib_config_function_t * f, char *name, int is_early) {
  vlib_global_main_t *vm = &vlib_global_main;
  vlib_config_function_runtime_t *r;
  r = clib_mem_alloc (sizeof (r[0]));
  clib_memset (r, 0, sizeof (r[0]));
  r->function = f;
  r->is_early = is_early ? 1 : 0;
  strncpy(r->name, name, sizeof(r->name));
  r->next_registration = vm->config_function_registrations;
  vm->config_function_registrations = r;
}

__clib_export void ego_add_cli_command (vlib_cli_command_function_t *f, char *path, char *short_help, char *long_help, int is_mp_safe) {
  vlib_global_main_t * vgm = &vlib_global_main;
  vlib_cli_main_t *cm = &vgm->cli_main;
  vlib_cli_command_t *c;
  vlib_cli_command_t **p = &cm->cli_command_registrations;

  c = clib_mem_alloc (sizeof(c[0]));
  memset (c, 0, sizeof (c[0]));
  c->path = path;
  c->short_help = short_help;
  c->long_help = long_help;
  c->function = f;
  c->next_cli_command = *p;
  c->is_mp_safe = is_mp_safe;
  *p = c;
 }

__clib_export void ego_add_node(ego_vlib_node_config_t *cf)
{
  vlib_global_main_t * vgm = &vlib_global_main;
  vlib_node_registration_t *r;
  struct _vlib_node_registration **p = &vgm->node_registrations;

  r = clib_mem_alloc (sizeof (r[0]) + cf->n_next_nodes*sizeof(char *));
  clib_memset (r, 0, sizeof (r[0]));

  /* Registrations already performed, too late now. */
  ASSERT ((*p)->index == 0);
  r->next_registration = *p;
  *p = r;

  r->type = cf->typ;
  r->state = cf->state;
  r->function = cf->f;
  r->name = cf->name;
  r->n_next_nodes = cf->n_next_nodes;
  r->error_strings = cf->errors;
  r->n_errors = cf->n_errors;
  r->flags = cf->flags;
  for (int i = 0; i < cf->n_next_nodes; i++)
    r->next_nodes[i] = cf->next_nodes[i];
  if (cf->next_threads != 0)
    {
      vec_resize (r->next_node_thread_indices, cf->n_next_nodes);
      for (int i = 0; i < cf->n_next_nodes; i++)
	r->next_node_thread_indices[i] = cf->next_threads[i];
    }
  r->scalar_size = cf->scalar_size;
  r->vector_size = cf->vector_size;
  r->opaque = cf->opaque;
}

__clib_export u32 ego_vlib_buffer_alloc (vlib_main_t * vm, u32 * buffers, u32 n_buffers)
{
  u32 n;
  n = vlib_buffer_alloc (vm, buffers, n_buffers);
  if (CLIB_DEBUG > 0 && 0)
    vlib_buffer_validate_alloc_free (vm, buffers, n, VLIB_BUFFER_KNOWN_FREE);
  return n;
}
__clib_export void ego_vlib_buffer_free (vlib_main_t * vm, u32 * buffers, u32 n_buffers)
{
  if (CLIB_DEBUG > 0 && 0)
    vlib_buffer_validate_alloc_free (vm, buffers, n_buffers, VLIB_BUFFER_KNOWN_ALLOCATED);
  vlib_buffer_free (vm, buffers, n_buffers);
}
__clib_export void ego_vlib_buffer_enqueue_to_next (vlib_main_t * vm, vlib_node_runtime_t * node, u32 * buffers, u16 * nexts, uword count) {
  vlib_buffer_enqueue_to_next (vm, node, buffers, nexts, count);
}
__clib_export void ego_vlib_process_suspend (vlib_main_t * vm, f64 dt) {
  vlib_worker_thread_barrier_release (vm);
  vlib_process_suspend (vm, dt);
  vlib_worker_thread_barrier_sync(vm);
}
__clib_export void ego_vlib_process_signal_event (vlib_main_t * vm, uword node_index, uword type_opaque, uword data) {
  vlib_process_signal_event_mt (vm, node_index, type_opaque, data);
}
__clib_export f64 ego_vlib_process_wait_for_event_or_clock (vlib_main_t * vm, f64 dt) { return vlib_process_wait_for_event_or_clock (vm, dt); }
__clib_export void ego_vlib_process_put_event_data (vlib_main_t * vm, void *event_data) {  vlib_process_put_event_data (vm, event_data); }
__clib_export uword ego_vlib_process_get_events (vlib_main_t * vm, uword ** data_vector) { return vlib_process_get_events (vm, data_vector); }

__clib_export void ego_vlib_error_count (vlib_main_t * vm, uword node_index, uword counter, uword increment) { vlib_error_count (vm, node_index, counter, increment); }

__clib_export void ego_fib_route_path_set_fib_index (fib_route_path_t *p, u32 i) { p->frp_fib_index = i; }
__clib_export void ego_fib_route_path_set_sw_if_index (fib_route_path_t *p, u32 i) { p->frp_sw_if_index = i; }
__clib_export fib_route_path_t * ego_fib_route_path_vec_alloc (uword n) { fib_route_path_t *v = 0; vec_resize (v, n);  return v; }
__clib_export void ego_fib_route_path_vec_free (fib_route_path_t *v) { vec_free (v); }
__clib_export ip4_address_t *ego_fib_prefix_ip4_address (fib_prefix_t *p) { return &p->fp_addr.ip4; }
__clib_export ip6_address_t *ego_fib_prefix_ip6_address (fib_prefix_t *p) { return &p->fp_addr.ip6; }

__clib_export u32 ego_get_session_index (session_t *s) { return s->session_index; }
__clib_export u64 ego_get_session_handle (session_t *s) { return (u64)(session_handle(s)); }
__clib_export session_worker_t *ego_session_main_get_worker (clib_thread_index_t thread_index) { return session_main_get_worker(thread_index); }
__clib_export session_t *ego_session_get_from_handle (u64 handle) { return session_get_from_handle (handle); }
__clib_export void ego_unlisten_args_set_handle(vnet_unlisten_args_t *args, u64 handle) {  args->handle = handle; }

__clib_export pg_node_t *ego_pg_get_node (uword node_index) { return pg_get_node (node_index); }
