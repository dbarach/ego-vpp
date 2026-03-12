#include <vlib/vlib.h>
#include <vlib/unix/plugin.h>	/* VLIB_PLUGIN_REGISTER */
#include <vnet/ethernet/ethernet.h>
#include <vnet/session/application.h>
#include <vnet/session/session.h>
#include <vnet/ip/ip4.h>
#include <vnet/ip/ip6.h>
#include <vnet/fib/fib.h>
#include <vnet/pg/pg.h>

u8 *ego_vec_string(void *ptr, uword len);
void ego_vec_free(void *ptr);

void ego_add_init_func(int which, vlib_init_function_t * f, char *name, char **runs_before, char **runs_after, char **init_order);
void ego_add_config_func(vlib_config_function_t * f, char *name, int is_early);
void ego_add_cli_command(vlib_cli_command_function_t *f, char *path, char *short_help, char *long_help, int is_mp_safe);

typedef struct {
  char * name;
  int typ, state, flags, scalar_size, vector_size;
  vlib_node_function_t * f;
  int n_next_nodes;
  char **next_nodes;
  u32 *next_threads;
  int n_errors;
  char **errors;
  uword opaque;
} ego_vlib_node_config_t;

void ego_add_node(ego_vlib_node_config_t *cf);

vlib_main_t * ego_vlib_get_main ();
vlib_node_t * ego_vlib_get_node (vlib_main_t * vm, u32 i);
u32 ego_vlib_threads (u32 *n_threads);
u32 ego_vlib_get_current_process_node_index (vlib_main_t * vm);
vlib_node_state_t ego_vlib_node_get_state (vlib_main_t * vm, u32 node_index);
void ego_vlib_node_set_state (vlib_main_t * vm, u32 node_index, vlib_node_state_t new_state);
void ego_vlib_frame_args (vlib_frame_t * f, void **scalar, void **vector);
vlib_frame_t * ego_vlib_node_runtime_get_next_frame (vlib_main_t * vm, vlib_node_runtime_t * n, u32 next_index, u32 *ds);

u32 ego_vlib_buffer_alloc (vlib_main_t * vm, u32 * buffers, u32 n_buffers);
void ego_vlib_buffer_free (vlib_main_t * vm, u32 * buffers, u32 n_buffers);
void ego_vlib_buffer_enqueue_to_next (vlib_main_t * vm, vlib_node_runtime_t * node, u32 * buffers, u16 * nexts, uword count);
void ego_vlib_process_suspend (vlib_main_t * vm, f64 dt);
void ego_vlib_process_signal_event (vlib_main_t * vm, uword node_index, uword type_opaque, uword data);
f64 ego_vlib_process_wait_for_event_or_clock (vlib_main_t * vm, f64 dt);
void ego_vlib_process_put_event_data (vlib_main_t * vm, void *event_data);
uword ego_vlib_process_get_events (vlib_main_t * vm, uword ** data_vector);
void ego_vlib_error_count (vlib_main_t * vm, uword node_index, uword counter, uword increment);

vnet_sw_interface_t *ego_vnet_get_sw_interface (vnet_main_t * vnm, u32 sw_if_index);
vnet_sw_interface_flags_t ego_vnet_sw_interface_get_flags (vnet_main_t * vnm, u32 sw_if_index);

void ego_fib_route_path_set_fib_index (fib_route_path_t *p, u32 i);
void ego_fib_route_path_set_sw_if_index (fib_route_path_t *p, u32 i);
fib_route_path_t * ego_fib_route_path_vec_alloc (uword n);
void ego_fib_route_path_vec_free (fib_route_path_t *v);
ip4_address_t *ego_fib_prefix_ip4_address (fib_prefix_t *p);
ip6_address_t *ego_fib_prefix_ip6_address (fib_prefix_t *p);

u32 ego_get_session_index (session_t *s);
u64 ego_get_session_handle (session_t *s);
session_worker_t *ego_session_main_get_worker (clib_thread_index_t thread_index);
session_t *ego_session_get_from_handle (u64 handle);
void ego_unlisten_args_set_handle(vnet_unlisten_args_t *args, u64 handle);

pg_node_t *ego_pg_get_node (uword node_index);
