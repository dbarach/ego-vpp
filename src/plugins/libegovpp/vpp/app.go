package vpp

// #cgo CFLAGS: -Wno-address-of-packed-member
// #include "vpp.h"
import "C"

import (
	"std"
	"sync"
	"sync/atomic"
	"sys"
	"unsafe"
)

type (
	app_main struct {
		init_once    sync.Once
		mutex        *std.Spinlock
		app_by_index std.Vec[AppInterface]
		app_workers  std.PagedVec[*app_worker]
		segment_main
		session_main
	}
	AppInterface interface {
		SessionConnected(s *ConnectedSession) SessionError
		SessionDisconnected(s *Session)
		SessionAccepted(s *Session) SessionError
		SessionRx(s []*Session) SessionError
		SessionTx(s []*Session) SessionError
	}
	App struct {
		AppConfig
		app_index uint32
	}
	AppConfig struct {
		AcceptRedirect              bool
		AddSegment                  bool
		IsBuiltin                   bool // rx/tx callbacks active.
		IsTransport                 bool
		IsProxy                     bool
		GlobalScope                 bool
		LocalScope                  bool
		EventMessageQueueUseEventFd bool
		MemfdForBuiltin             bool
		UseHugePages                bool // for segments
		GetOriginalDst              bool
		EventCollector              bool
		NoDumpSegments              bool

		Name                       string
		EventQueueLen              uint64 // number of items in event queue between app <-> vpp
		FirstSegmentSize           uint64 // in bytes for first segment
		AdditionalSegmentSize      uint64 // in bytes for additional segments
		MaxFifoMemorySize          uint64 // in bytes: limits number of segments
		RxFifoSize, TxFifoSize     uint64 // in bytes: rx/tx fifo capacity are powers of 2 >= given size
		Log2ChunkSize              uint64 // log2 fifo chunk size in bytes
		MaxFifoSize                uint64 // in bytes.  not sure what this is used for.
		NumPreallocFifoPairs       uint64 // number of fifo rx/tx pairs to pre-allocate
		NumPreallocFifoHeaders     uint64
		Namespace                  uint64 // name space index
		NamespaceSecret            uint64 // secret if use_sock_api is false
		ProxyTransport             uint64 // mask of transports to proxy (e.g. 1 << TRANSPORT_PROTO_TCP)
		TLSEngine                  uint64
		MemUsagePercentHiWatermark uint64 // hi pressure when memory usage > this percentage
		MemUsagePercentLoWatermark uint64 // lo pressure when memory usage > this percentage but less than hi threshold
	}
)

func (a *app_main) init() {
	nt := NumThreads()
	if nt > 1 {
		a.mutex = new(std.Spinlock)
	}
	a.session_main.init(nt)
}
func get_app_main() (m *app_main) {
	m = &global_main.app_main
	m.init_once.Do(m.init)
	return
}

// Convert to array of 64 bit option parameters in vnet_app_attach_args_t
func (c *AppConfig) u64(o *[C.APP_OPTIONS_N_OPTIONS]C.u64) {
	var f C.int
	if c.AcceptRedirect {
		f |= C.APP_OPTIONS_FLAGS_ACCEPT_REDIRECT
	}
	if c.AddSegment {
		f |= C.APP_OPTIONS_FLAGS_ADD_SEGMENT
	}
	if c.IsBuiltin {
		f |= C.APP_OPTIONS_FLAGS_IS_BUILTIN
	}
	if c.IsTransport {
		f |= C.APP_OPTIONS_FLAGS_IS_TRANSPORT_APP
	}
	if c.IsProxy {
		f |= C.APP_OPTIONS_FLAGS_IS_PROXY
	}
	if c.GlobalScope {
		f |= C.APP_OPTIONS_FLAGS_USE_GLOBAL_SCOPE
	}
	if c.LocalScope {
		f |= C.APP_OPTIONS_FLAGS_USE_LOCAL_SCOPE
	}
	if c.EventMessageQueueUseEventFd {
		f |= C.APP_OPTIONS_FLAGS_EVT_MQ_USE_EVENTFD
	}
	if c.MemfdForBuiltin {
		f |= C.APP_OPTIONS_FLAGS_MEMFD_FOR_BUILTIN
	}
	if c.UseHugePages {
		f |= C.APP_OPTIONS_FLAGS_USE_HUGE_PAGE
	}
	if c.GetOriginalDst {
		f |= C.APP_OPTIONS_FLAGS_GET_ORIGINAL_DST
	}
	if c.EventCollector {
		f |= C.APP_OPTIONS_FLAGS_EVT_COLLECTOR
	}
	if c.NoDumpSegments {
		f |= C.APP_OPTIONS_FLAGS_NO_DUMP_SEGMENTS
	}
	o[C.APP_OPTIONS_FLAGS] = C.u64(f)
	o[C.APP_OPTIONS_EVT_QUEUE_SIZE] = C.u64(c.EventQueueLen)
	o[C.APP_OPTIONS_SEGMENT_SIZE] = C.u64(c.FirstSegmentSize)
	o[C.APP_OPTIONS_ADD_SEGMENT_SIZE] = C.u64(c.AdditionalSegmentSize)
	o[C.APP_OPTIONS_MAX_FIFO_MEMORY] = C.u64(c.MaxFifoMemorySize)
	o[C.APP_OPTIONS_RX_FIFO_SIZE] = C.u64(c.RxFifoSize)
	o[C.APP_OPTIONS_TX_FIFO_SIZE] = C.u64(c.TxFifoSize)
	o[C.APP_OPTIONS_PREALLOC_FIFO_PAIRS] = C.u64(c.NumPreallocFifoPairs)
	o[C.APP_OPTIONS_PREALLOC_FIFO_HDRS] = C.u64(c.NumPreallocFifoHeaders)
	o[C.APP_OPTIONS_NAMESPACE] = C.u64(c.Namespace)
	o[C.APP_OPTIONS_NAMESPACE_SECRET] = C.u64(c.NamespaceSecret)
	o[C.APP_OPTIONS_PROXY_TRANSPORT] = C.u64(c.ProxyTransport)
	o[C.APP_OPTIONS_TLS_ENGINE] = C.u64(c.TLSEngine)
	o[C.APP_OPTIONS_MAX_FIFO_SIZE] = C.u64(c.MaxFifoSize)
	o[C.APP_OPTIONS_HIGH_WATERMARK] = C.u64(c.MemUsagePercentHiWatermark)
	o[C.APP_OPTIONS_LOW_WATERMARK] = C.u64(c.MemUsagePercentLoWatermark)
	o[C.APP_OPTIONS_LOG2_CHUNK_SIZE] = C.u64(c.Log2ChunkSize)
}

func (a *App) Attach(ai AppInterface, cf *AppConfig) (e SessionError) {
	args := &C.vnet_app_attach_args_t{
		api_client_index: C.APP_INVALID_INDEX,
		// namespace_id:     "default",
		// use_sock_api:     1,
	}

	if cf.Name == "" {
		cf.Name = "no name"
	}
	{
		b := []byte(cf.Name)
		args.name = C.ego_vec_string(unsafe.Pointer(&b[0]), C.uword(len(b)))
	}

	var options [C.APP_OPTIONS_N_OPTIONS]C.u64
	cf.u64(&options)
	args.options = &options[0]

	args.session_cb_vft = &C.session_cb_vft_t{
		add_segment_callback: func(app_wrk_index C.u32, h C.u64) C.int {
			return C.int(segment_handle(h).add_del(false))
		},
		del_segment_callback: func(app_wrk_index C.u32, h C.u64) C.int {
			return C.int(segment_handle(h).add_del(true))
		},
		session_accept_callback: func(s_ *C.session_t) C.int {
			sc := session_c{c: s_}
			s := sc.validate(true)
			e := s.ai.SessionAccepted(s)
			if e.Ok() {
				sc.set_state(SessionStateReady)
			}
			return C.int(e)
		},
		session_connected_callback: func(app_wrk_index_, api_context_ C.u32, s_ *C.session_t, e_ C.session_error_t) C.int {
			s := session_c{c: s_}.validate(false)
			cs := &ConnectedSession{Session: s, ApiContext: uint32(api_context_), Error: SessionError(e_)}
			return C.int(s.ai.SessionConnected(cs))
		},
		session_disconnect_callback: func(s_ *C.session_t) {
			s := session_c{c: s_}.Session()
			s.ai.SessionDisconnected(s)
		},
		session_transport_closed_callback: func(c *C.session_t) {
			s := session_c{c: c}.Session()
			s.ai.SessionDisconnected(s)
		},
		// session_cleanup_callback:          func(s *C.session_t, ntf C.session_cleanup_ntf_t) { panic("cleanup") },
		// half_open_cleanup_callback:        func(s *C.session_t) { panic("not yet") },
		session_reset_callback:   func(s *C.session_t) { panic("not yet") },
		session_migrate_callback: func(s *C.session_t, h C.session_handle_t) { panic("not yet") },
		session_listened_callback: func(app_wrk_index, api_context C.u32, h C.session_handle_t, rv C.int) C.int {
			panic("not yet")
			return 0
		},
		session_unlistened_callback: func(app_wrk_index C.u32, h C.session_handle_t, context C.u32, rv C.int) { panic("not yet") },
		builtin_app_rx_callback: func(c *C.session_t) C.int {
			s := session_c{c: c}.Session()
			return C.int(s.ai.SessionRx([]*Session{s}))
		},
		builtin_app_rx_tx_batch_callback: func(thread_index C.u32, session_indices *C.u32, is_tx C.u32) C.int {
			sis := unsafe.Slice(session_indices, clib_vec_len(unsafe.Pointer(session_indices)))
			w := session_worker_with_index(uint(thread_index))
			s := makes([]*Session, len(sis))
			for i := range s {
				s[i] = w.sessions.At(uint(sis[i]))
			}
			if is_tx != 0 {
				return C.int(s[0].ai.SessionTx(s))
			} else {
				return C.int(s[0].ai.SessionRx(s))
			}
		},
		builtin_app_tx_callback: func(c *C.session_t) C.int {
			s := session_c{c: c}.Session()
			return C.int(s.ai.SessionTx([]*Session{s}))
		},
		// fifo_tuning_callback: func(s *C.session_t, f *C.svm_fifo_t, action C.session_ft_action_t, bytes C.u32) C.int {
		// 	return 0
		// },
	}
	if e = SessionError(C.vnet_application_attach(args)); e.Ok() {
		m := get_app_main()
		a.init(m, ai, uint32(args.app_index), true)
		a.AppConfig = *cf
	}
	C.ego_vec_free(unsafe.Pointer(args.name))
	return
}

func (a *App) init(m *app_main, ai AppInterface, i uint32, reg bool) {
	if reg {
		a.app_index = i
		*m.app_by_index.Validate(uint(i)) = ai
	} else {
		a.app_index = ^uint32(0)
		m.app_by_index[i] = nil
	}
}

func (a *App) Detach() SessionError {
	args := &C.vnet_app_detach_args_t{
		app_index:        C.u32(a.app_index),
		api_client_index: C.APP_INVALID_INDEX,
	}
	e := SessionError(C.vnet_application_detach(args))
	if e.Ok() {
		a.init(&global_main.app_main, nil, a.app_index, false)
	}
	return e
}

type (
	AppWorkerMap  struct{ c C.u32 }
	ConnectConfig struct {
		WorkerMap  AppWorkerMap
		ApiContext uint32
	}
)

func (a *App) Connect(cfg *SessionEndpointConfig, cf *ConnectConfig) (s SessionHandle, e SessionError) {
	args := &C.vnet_connect_args_t{
		app_index:     C.u32(a.app_index),
		wrk_map_index: cf.WorkerMap.c,
		api_context:   C.u32(cf.ApiContext), // opaque passed to connected callback
	}
	*(*SessionEndpointConfig)(unsafe.Pointer(args)) = *cfg
	if e = SessionError(C.vnet_connect(args)); e.Ok() {
		s = SessionHandle(args.sh) // session handle result
	}
	return
}

// Analagous to close(2) on Unix socket.
func (a *App) Disconnect(s SessionHandle) SessionError {
	args := &C.vnet_disconnect_args_t{
		app_index: C.u32(a.app_index),
		handle:    C.session_handle_t(s),
	}
	return SessionError(C.vnet_disconnect_session(args))
}

// Analagous to shutdown(2) on Unix socket.
func (a *App) Shutdown(s SessionHandle) SessionError {
	args := &C.vnet_shutdown_args_t{
		app_index: C.u32(a.app_index),
		handle:    C.session_handle_t(s),
	}
	return SessionError(C.vnet_shutdown_session(args))
}

func (a *App) Listen(cfg *SessionEndpointConfig, m AppWorkerMap) (s SessionHandle, e SessionError) {
	args := &C.vnet_listen_args_t{
		app_index:     C.u32(a.app_index),
		wrk_map_index: m.c,
	}
	*(*SessionEndpointConfig)(unsafe.Pointer(args)) = *cfg
	if e = SessionError(C.vnet_listen(args)); e.Ok() {
		s = SessionHandle(args.handle) // session handle result
	}
	return
}

func (a *App) Unlisten(s SessionHandle, m AppWorkerMap) SessionError {
	args := &C.vnet_unlisten_args_t{
		app_index:     C.u32(a.app_index),
		wrk_map_index: m.c,
	}
	// Handle is union in C code: therefore need to call C function to get the job done.
	C.ego_unlisten_args_set_handle(args, C.u64(s))
	return SessionError(C.vnet_unlisten(args))
}

type (
	app_worker_map_c struct{ c C.u32 } // index into application.worker_maps; maps app local worker index to global app_wrk_index
	app_worker       struct {
		c  *C.app_worker_t
		ai AppInterface
		mq session_message_queue
	}
)

func app_worker_with_index(i uint32) (w *app_worker) {
	m := &global_main.app_main
	m.mutex.LockIf()
	defer m.mutex.UnlockIf()
	p := m.app_workers.Validate(uint(i))
	w = *p
	if w == nil {
		w = new(app_worker)
		w.c = C.app_worker_get(C.u32(i))
		w.ai = m.app_by_index[w.c.app_index]
		w.mq.message_queue.init(w.c.event_queue)
		*p = w
	}
	return
}

const (
	session_message_queue_io = iota
	session_message_queue_control
)

const (
	session_io_event_rx session_event_type = iota
	session_io_event_tx
	session_io_event_tx_flush
	session_io_event_builtin_rx
	session_io_event_tx_main
	session_control_event_rpc
	session_control_event_half_close
	session_control_event_close
	session_control_event_reset
	session_control_event_listened
	session_control_event_unlisten_reply
	session_control_event_accepted
	session_control_event_accepted_reply
	session_control_event_connected
	session_control_event_disconnected
	session_control_event_disconnected_reply
	session_control_event_reset_reply
	session_control_event_request_worker_update
	session_control_event_worker_update
	session_control_event_worker_update_reply
	session_control_event_shutdown
	session_control_event_disconnect
	session_control_event_connect
	session_control_event_connect_uri
	session_control_event_listen
	session_control_event_listen_uri
	session_control_event_unlisten
	session_control_event_app_detach
	session_control_event_app_add_segment
	session_control_event_app_del_segment
	session_control_event_migrated
	session_control_event_cleanup
	session_control_event_app_wrk_rpc
	session_control_event_transport_attribute
	session_control_event_transport_attribute_reply
	session_control_event_transport_closed
	session_control_event_half_cleanup
	session_control_event_connect_stream
)

type session_event struct {
	typ       session_event_type
	postponed uint8
	d         [0]uint8
}

func (e *session_event) data() unsafe.Pointer { return unsafe.Pointer(&e.d) }
func (e *session_event) data32() *uint32      { return (*uint32)(e.data()) }
func (e *session_event) data64() *uint64      { return (*uint64)(e.data()) }

type (
	session_message_queue struct{ message_queue }
	session_event_type    uint8
)

func (q *session_message_queue) tx32(typ session_event_type, i uint) (ok bool) {
	ptr, need_signal, ring_full := q.tx(session_message_queue_io, true)
	if ring_full {
		return
	}
	e := (*session_event)(ptr)
	e.typ = typ
	e.postponed = 0
	*e.data32() = uint32(i)
	if need_signal {
		q.send_signal(false)
	}
	ok = true
	return
}

func (p *vuint32) l() uint32 {
	return atomic.LoadWithOptions((*uint32)(p), atomic.Options{Ordering: atomic.Monotonic})
}
func (p *vuint32) s(v uint32) {
	atomic.StoreWithOptions((*uint32)(p), v, atomic.Options{Ordering: atomic.Release})
}
func (p *vuint32) add(v uint32) uint32 {
	return atomic.AddWithOptions((*uint32)(p), v, atomic.Options{Ordering: atomic.Release})
}

// Index full?  Ring full?
func (q *message_queue_shared) full() bool      { return q.len.l() >= q.cap }
func (q *message_queue_ring_shared) full() bool { return q.len.l() >= q.cap }
func (q *message_queue) ring_full(r int) bool {
	return q.message_queue_shared.full() || q.rings[r].full()
}
func (q *message_queue) lock(wait bool) (locked bool) {
	efd := q.event_fd()
	if efd == -1 {
		locked = q.mutex_lock(wait)
	} else {
		C.clib_spinlock_lock(&q.c.q.lock)
	}
	return
}
func (q *message_queue) unlock() {
	efd := q.event_fd()
	if efd == -1 {
		q.mutex_unlock()
	} else {
		C.clib_spinlock_unlock(&q.c.q.lock)
	}
}
func (q *message_queue) mutex_lock(wait bool) (ok bool) {
	var rv C.int
	if wait {
		rv = C.pthread_mutex_lock(&q.mutex)
	} else {
		rv = C.pthread_mutex_trylock(&q.mutex)
	}
	ok = true
	if rv == C.int(C.EOWNERDEAD) {
		ok = C.pthread_mutex_consistent(&q.mutex) == 0
	}
	return
}
func (q *message_queue) mutex_unlock() { C.pthread_mutex_unlock(&q.mutex) }

func (q *message_queue) tx(ring_index int, wait bool) (p unsafe.Pointer, need_signal, ring_full bool) {
	if ok := q.lock(wait); !ok {
		return
	}
	if ring_full = q.ring_full(ring_index); !ring_full {
		r := q.rings[ring_index]
		qtail, rtail := q.tail, r.tail
		q.index[qtail] = message_index{ring_index: uint32(ring_index), elt_index: rtail}
		p = r.at(rtail)
		rtail++
		qtail++
		r.tail = ifelse(rtail < r.cap, rtail, 0)
		q.tail = ifelse(qtail < q.cap, qtail, 0)
		r.len.add(1)
		need_signal = q.len.add(1) == 0
	}
	q.unlock()
	return
}
func (q *message_queue) event_fd() int32 { return int32(q.c.q.evtfd) }
func (q *message_queue) send_signal(is_consumer bool) (ok bool) {
	efd := q.event_fd()
	if efd == -1 {
		if is_consumer {
			if rv := C.pthread_mutex_lock(&q.mutex); rv == C.int(C.EOWNERDEAD) {
				ok = C.pthread_mutex_consistent(&q.mutex) == 0
				return
			}
		}
		C.pthread_cond_broadcast(&q.condvar)
		if is_consumer {
			C.pthread_mutex_unlock(&q.mutex)
		}
	} else {
		data := uint64(1)
		n, e := sys.FileDesc(efd).Write(unsafe.Slice((*byte)(unsafe.Pointer(&data)), 8))
		if n != 8 || e != sys.OK {
			std.Warnf("signal write on %d returned %d %v", efd, n, e)
		}
	}
	ok = true
	return
}

type (
	message_queue_c      struct{ c *C.svm_msg_q_t }
	vuint32              uint32 // +ego volatile
	message_queue_shared struct {
		mutex         C.pthread_mutex_t
		condvar       C.pthread_cond_t
		head, tail    uint32
		len           vuint32
		cap           uint32
		bytes_per_elt uint32 // always 8 = svm_msg_q_msg_t size
		_             [4]uint8
		// cap messages follow
		index_data [0]message_index
		// messages are followed by rings.
	}
	// invalid ~0 for both
	message_index struct {
		ring_index uint32 // index of ring
		elt_index  uint32 // index of message in ring
	}
	message_queue_ring_shared struct {
		len           vuint32 // current number of elts in ring
		cap           uint32  // max size of the ring
		head, tail    uint32  // head for dequeue; tail for enqueue
		bytes_per_elt uint32  // size of an element in bytes
		// cap * bytes_per_elt bytes of data follow for this ring.
		data [0]uint8 // chunk of memory for msg data
	}
	message_queue_ring struct {
		cap, bytes_per_elt uint32
		*message_queue_ring_shared
	}
	message_queue struct {
		*message_queue_shared
		c     *C.svm_msg_q_t
		index []message_index // slice pointing to shared index
		rings []message_queue_ring
	}
)

func (r *message_queue_ring) at(i uint32) unsafe.Pointer {
	return unsafe.Add(unsafe.Pointer(&r.data), i*r.bytes_per_elt)
}
func (q *message_queue) at(i uint32) (p unsafe.Pointer) {
	m := &q.index[i]
	p = q.rings[m.ring_index].at(m.elt_index)
	return
}

func (q *message_queue) init(c *C.svm_msg_q_t) {
	q.c = c
	s := (*message_queue_shared)(unsafe.Pointer(c.q.shr))
	q.message_queue_shared = s
	q.index = unsafe.Slice((*message_index)(unsafe.Pointer(&s.index_data)), q.cap)
	q.rings = make([]message_queue_ring, clib_vec{c: unsafe.Pointer(c.rings)}.len())
	c_rings := unsafe.Slice((*C.svm_msg_q_ring_t)(unsafe.Pointer(c.rings)), len(q.rings))
	for i := range q.rings {
		r := message_queue_ring{
			cap:                       uint32(c_rings[i].nitems),
			bytes_per_elt:             uint32(c_rings[i].elsize),
			message_queue_ring_shared: (*message_queue_ring_shared)(unsafe.Pointer(c_rings[i].shr)),
		}
		q.rings[i] = r
	}
}
