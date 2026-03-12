package vpp

// #cgo CFLAGS: -Wno-address-of-packed-member
// #include "vpp.h"
import "C"

import (
	"net/ip"
	"net/ip4"
	"net/ip6"
	"std"
	"sync/atomic"
	"unsafe"
)

type (
	SessionEndpointConfig struct {
		c C.session_endpoint_cfg_t
	}
	TransportEndpoint struct {
		c C.transport_endpoint_t
	}
)

func (c *SessionEndpointConfig) Endpoint() *TransportEndpoint {
	return (*TransportEndpoint)(unsafe.Pointer(c))
}
func (c *SessionEndpointConfig) Peer() *TransportEndpoint {
	return (*TransportEndpoint)(unsafe.Pointer(&c.c.peer))
}
func (c *SessionEndpointConfig) SetIp4(x ip4.Socket)       { c.Endpoint().SetIp4(x) }
func (c *SessionEndpointConfig) SetIp6(x ip6.Socket)       { c.Endpoint().SetIp6(x) }
func (c *SessionEndpointConfig) SetFib(i FibIndex)         { c.Endpoint().SetFib(i) }
func (e *SessionEndpointConfig) SetProto(p TransportProto) { e.c.transport_proto = C.u8(p) }

func (e *TransportEndpoint) SetIp4(x ip4.Socket) {
	e.c.is_ip4 = 1
	*(*ip4.Address)(unsafe.Pointer(&e.c.ip[12])) = x.Address
	e.c.port = C.u16(x.Port)
}
func (e *TransportEndpoint) SetIp6(x ip6.Socket) {
	e.c.is_ip4 = 0
	*(*ip6.Address)(unsafe.Pointer(&e.c.ip[0])) = x.Address
	e.c.port = C.u16(x.Port)
}
func (e *TransportEndpoint) SetFib(i FibIndex) { e.c.fib_index = C.u32(i.c) }

func (e *TransportEndpoint) Parse(in *std.ParseInput) {
	var (
		a4 ip4.Socket
		a6 ip6.Socket
	)
	switch {
	case in.Parse("%v:%v", &a4.Address, &a4.Port):
		e.SetIp4(a4)
	case in.Parse("%v:%v", &a6.Address, &a6.Port):
		e.SetIp6(a6)
	default:
		in.ParseError()
	}
}
func (cfg *SessionEndpointConfig) Parse(in *std.ParseInput) {
	var (
		proto TransportProto
		flags SessionEndpointConfigFlags
		te    [2]TransportEndpoint
	)
	switch {
	case in.Parse("%v:", &proto):
	case in.Parse("%vs:", &proto):
		flags |= SESSION_ENDPOINT_SECURE
	default:
		in.ParseError()
	}

	if !in.Parse("//%v", &te[0]) {
		in.ParseError()
	}
	if !in.End() {
		if !in.Parse("%v", &te[1]) {
			in.ParseError()
		}
	}
	*cfg.Endpoint() = te[0]
	*cfg.Peer() = te[1]
	cfg.SetProto(proto)
	cfg.c.flags = C.session_endpoint_cfg_flags_t(flags)
}

type Socket struct {
	a      ip6.Address
	is_ip4 bool
	p      ip.Port
}

func (s *Socket) a4() *ip4.Address {
	return (*ip4.Address)(unsafe.Pointer(&s.a[12]))
}
func (s *Socket) from_transport(a *C.ip46_address_t, p C.u16, is_ip4 bool) {
	s.is_ip4 = is_ip4
	s.p = ip.Port(p)
	s.a = *(*ip6.Address)(unsafe.Pointer(a))
}
func (s *Socket) Format(b *std.Buffer) {
	if s.is_ip4 {
		t := &ip4.Socket{Address: *s.a4(), Port: s.p}
		t.Format(b)
	} else {
		t := &ip6.Socket{Address: s.a, Port: s.p}
		t.Format(b)
	}
}
func (s *Session) Format(b *std.Buffer) {
	if s == nil {
		b.AddString("nil")
	} else {
		arrow := "->"
		if s.Accepted {
			arrow = "<-"
		}
		b.Format("%v %s %v", s.Socket(Rx), arrow, s.Socket(Tx))
	}
}
func (s *Session) Socket(rt RxTx) *Socket { return &s.sides[rt].sock }
func (s *Socket) Get(s4 *ip4.Socket, s6 *ip6.Socket) (is_ip4 bool) {
	if is_ip4 = s.is_ip4; is_ip4 {
		s4.Address = *s.a4()
		s4.Port = s.p
	} else {
		s6.Address = s.a
		s6.Port = s.p
	}
	return
}
func (s *Socket) Set(s4 *ip4.Socket, s6 *ip6.Socket) {
	if s4 != nil {
		s.is_ip4 = true
		*s.a4() = s4.Address
		s.p = s4.Port
	} else {
		s.is_ip4 = false
		s.a = s6.Address
		s.p = s6.Port
	}
}

type RxTx uint8

func (r RxTx) OtherSide() RxTx { return r ^ 1 }

const (
	Rx RxTx = iota
	Tx
	NRxTx
)

type (
	session_main struct {
		session_workers std.Vec[session_worker]
	}
	Session struct {
		Handle    session_handle
		Opaque    uint32
		Accepted  bool
		Connected bool
		ai        AppInterface
		aw        *app_worker
		sw        *session_worker
		sides     [NRxTx]session_side
	}
	session_side struct {
		fifo fifo
		sock Socket
	}
	ConnectedSession struct {
		*Session
		ApiContext uint32
		Error      SessionError
	}
	session_c      struct{ c *C.session_t }
	session_worker struct {
		c        *C.session_worker_t
		mq       session_message_queue
		sessions std.PagedVec[Session]
	}
	session_handle uint64
)

func (m *session_main) init(n_threads int) {
	m.session_workers.ValidateLeni(n_threads)
}

type SessionHandle = session_handle

const SessionHandleNil = ^session_handle(0)

// really session worker index
func (h session_handle) Worker() *session_worker {
	return &global_main.app_main.session_workers[h.WorkerIndex()]
}

// +ego always_inline
func (h session_handle) Session() *Session {
	return h.Worker().sessions.At(h.SessionIndex())
}

// +ego always_inline
func (h session_handle) WorkerIndex() uint { return uint(h >> 32) }

// +ego always_inline
func (h session_handle) SessionIndex() uint { return uint(h & (1<<32 - 1)) }
func (h session_handle) get_session_c() (c session_c) {
	c.c = C.ego_session_get_from_handle(C.u64(h))
	return
}

func (s session_c) handle() session_handle { return session_handle(C.ego_get_session_handle(s.c)) }
func (s session_c) Session() *Session      { return s.handle().Session() }
func (s session_c) session_worker() (w *session_worker) {
	return session_worker_with_index(s.handle().WorkerIndex())
}
func (s session_c) set_state(t SessionState) { s.c.session_state = C.session_state_t(t) }
func session_worker_with_index(i uint) *session_worker {
	m := &get_app_main().session_main
	w := &m.session_workers[i]
	if w.c == nil {
		w.c = C.ego_session_main_get_worker(C.clib_thread_index_t(i))
		w.mq.message_queue.init(w.c.vpp_event_queue)
	}
	return w
}
func (s session_c) app_worker() *app_worker { return app_worker_with_index(uint32(s.c.app_wrk_index)) }

func (f *fifo) init(c *C.svm_fifo_t) {
	f.c = c
	f.r = (*fifo_shared_memory)(unsafe.Pointer(f.c.shr))
	f.g = global_main.app_main.find_or_create_segment(uint32(c.segment_manager), uint32(c.segment_index))
}
func (c session_c) validate(accepted bool) (s *Session) {
	h := c.handle()
	w := c.session_worker()
	s = w.sessions.Validate(h.SessionIndex())
	*s = Session{
		Handle:    h,
		Accepted:  accepted,
		Connected: !accepted,
		aw:        c.app_worker(),
		sw:        c.session_worker(),
	}
	s.sides[Rx].fifo.init(c.c.rx_fifo)
	s.sides[Tx].fifo.init(c.c.tx_fifo)
	s.ai = s.aw.ai
	{
		type T struct {
			rmt_ip, lcl_ip     C.ip46_address_t
			fib_index          C.u32
			rmt_port, lcl_port C.u16
			dscp               C.u8
			_                  [3]C.u8
			is_ip4             C.u8
			proto              C.u8
			_                  [2]C.u8
		}
		t := (*T)(unsafe.Pointer(C.session_get_transport(c.c)))
		is_ip4 := t.is_ip4 != 0
		s.sides[Rx].sock.from_transport(&t.lcl_ip, t.lcl_port, is_ip4)
		s.sides[Tx].sock.from_transport(&t.rmt_ip, t.rmt_port, is_ip4)
	}
	return
}

// +ego always_inline
func (f *fifo) get_tail() (tail uint32, t *fifo_chunk, d []byte) {
	tail = f.r.tail.offset
	t = f.r.tail.chunk.get(f)
	ti := tail - t.lo
	d = t.data()[ti:ti]
	return
}

// +ego always_inline
func (f *fifo) new_tail(old *fifo_chunk, data_len uint32) (new *fifo_chunk, new_data []byte) {
	if new = f.new_after(old); new != nil {
		o := new.offset(f.g)
		f.r.tail.chunk = o
		f.r.end_chunk = o
		new_data = new.data()[:data_len]
	}
	return
}

// +ego always_inline
func (s *Session) Side() (d RxTx) {
	d = Rx
	if s.Connected {
		d = Tx
	}
	return
}

// +ego always_inline
func (s *Session) GetTx() []byte {
	f := &s.sides[Tx].fifo
	tail, t, data := f.get_tail()
	n_data := cap(data)
	n_left := int(f.r.cap - (tail - f.r.head.offset))
	if n_data == 0 && n_left > 0 {
		// Allocate new chunk and add to tail.
		t, data = f.new_tail(t, 0)
		n_data = cap(data)
	}
	n_data = min(n_data, n_left)
	data = data[::n_data]
	if n_data == 0 {
		f.r.signals.set_want_dequeue_notification(fifo_notify_on_dequeue)
	}
	return data
}

// +ego always_inline
func (s *Session) PutTx(data []byte) (n_data int) {
	if len(data) == 0 {
		return 0
	}
	f := &s.sides[Tx].fifo
	tail, t, t_data := f.get_tail()

	// Need to copy data into fifo?  Or is data already there.
	n_data = min(len(data), int(f.r.cap-(tail-f.r.head.offset)))
	if &data[0] != &t_data[0] {
		d := data[:n_data]
		n_alloc := 0
		t_data = t_data[:cap(t_data)]
		n_data = 0
		for {
			n := copy(t_data, d)
			n_data += n
			if n >= len(d) {
				break
			}
			d = d[n:]
			var c *fifo_chunk
			c, t_data = f.new_tail(t, t.len)
			if c == nil {
				break
			}
			t = c
			n_alloc++
		}
		if n_alloc > 0 {
			f.r.end_chunk = t.offset(f.g)
		}
	}

	// Advance tail and possibly signal vpp.
	atomic.StoreWithOptions(&f.r.tail.offset, tail+uint32(n_data), atomic.Options{Ordering: atomic.Release})
	if !f.r.signals.set_has_event() {
		s.sw.mq.tx32(session_io_event_tx, s.Handle.SessionIndex())
	}
	if n_data < len(data) {
		f.r.signals.set_want_dequeue_notification(fifo_notify_on_dequeue)
	}
	return n_data
}
func (s *Session) RingFull() bool { return s.sw.mq.ring_full(session_message_queue_io) }

// +ego always_inline
func (s *Session) GetRx() (data []byte) {
	f := &s.sides[Rx].fifo
	head, tail := f.r.head.offset, f.r.tail.offset
	h := f.r.head.chunk.get(f)
	if head-h.lo >= h.len && h.next != 0 {
		next := h.next
		n := next.get(f)
		f.r.head.chunk = next
		f.r.start_chunk = next
		f.free_chunk(h)
		h = n
	}
	hi := head - h.lo
	hl := min(tail-head, h.len-hi)
	data = h.data()[hi : hi+hl]
	return
}

// +ego always_inline
func (s *Session) PutRx(len int) {
	if len > 0 {
		f := &s.sides[Rx].fifo
		f.r.signals.unset_has_event()
		atomic.AddWithOptions(&f.r.head.offset, uint32(len), atomic.Options{Ordering: atomic.Release})
	}
}

type (
	fifo_chunk struct {
		lo, len    uint32 // lo offset plus length on this chunk in bytes.
		next       fifo_segment_offset_for_chunk
		enq, deq   uint32  // rb tree indices
		chunk_data [0]byte // len bytes of data follows.
	}
	// +ego { volatile }
	fifo_signals struct {
		has_event                 uint32
		want_dequeue_notification uint32
		has_dequeue_notification  uint32
		n_subscribers             uint8
		subscribers               [C.SVM_FIFO_MAX_EVT_SUBSCRIBERS]uint8
		dequeue_threshold         uint32
	}
	// +ego { aligned cache }
	fifo_head_tail struct {
		chunk  fifo_segment_offset_for_chunk // offset (pointer) to head/tail chunk
		offset uint32                        // byte offset for head or tail
	}
	// +ego { aligned cache }
	fifo_shared_memory struct {
		start_chunk, end_chunk fifo_segment_offset_for_chunk
		min_alloc              uint32
		cap                    uint32 // capacity of fifo (full when tail-head == cap)
		slice_index            uint8
		next                   fifo_segment_offset_for_chunk
		signals                fifo_signals
		head, tail             fifo_head_tail
	}
	fifo struct {
		c *C.svm_fifo_t // never moves so we can retain pointer
		g *fifo_segment
		r *fifo_shared_memory
	}
	// +ego { aligned cache }
	fifo_segment_shared_memory_basic struct {
		n_cached_bytes     uintptr /**< Cached bytes */
		n_active_fifos     uint32  /**< Number of active fifos */
		n_reserved_bytes   uint32  /**< Bytes not to be allocated */
		max_log2_fifo_size uint32  /**< Max log2(chunk size) for fs */
		n_slices           uint8   /**< Number of slices */
		pct_first_alloc    uint8   /**< Pct of fifo size to alloc */
		n_mqs              uint8   /**< Num mqs for mqs segment */
	}
	// +ego { aligned cache }
	fifo_segment_shared_memory_alloc struct {
		byte_index       uintptr
		max_byte_index   uintptr
		start_byte_index uintptr
	}
	// +ego { aligned cache }
	fifo_segment_shared_memory_slice struct {
		free_chunk_heads [n_fifo_chunk_sizes]fifo_segment_free_chunk_head
		free_fifo_head   fifo_segment_offset
		n_fl_chunk_bytes uintptr                    /**< Chunk bytes on freelist */
		virtual_mem      uintptr                    /**< Slice sum of all fifo sizes */
		num_chunks       [n_fifo_chunk_sizes]uint32 /**< Allocated chunks by chunk size */
	}
	fifo_segment_free_chunk_head fifo_segment_offset_for_chunk
	// +ego { aligned cache }
	fifo_segment_shared_memory struct {
		fifo_segment_shared_memory_basic
		fifo_segment_shared_memory_alloc
		slices [0]fifo_segment_shared_memory_slice
	}
	fifo_segment struct {
		r      *fifo_segment_shared_memory
		slices []fifo_segment_shared_memory_slice
		n_push uint64
		n_pop  uint64
	}
	fifo_segment_offset           uintptr
	fifo_segment_offset_for_chunk fifo_segment_offset
)

// +ego always_inline
func (f *fifo_signals) get_has_event() bool { return f.has_event != 0 }

// +ego always_inline
func (f *fifo_signals) set_has_event() (old bool) {
	old = 1 == atomic.SwapWithOptions(&f.has_event, 1, atomic.Options{Ordering: atomic.Release})
	return
}

// +ego always_inline
func (f *fifo_signals) unset_has_event() {
	atomic.SwapWithOptions(&f.has_event, 0, atomic.Options{Ordering: atomic.Acquire})
}

type fifo_notify_flags uint32

const (
	fifo_notify_on_dequeue fifo_notify_flags = 1 << iota
	fifo_notify_on_transition_from_full
	fifo_notify_on_transition_to_empty
)

// +ego always_inline
func (f *fifo_signals) set_want_dequeue_notification(new fifo_notify_flags) (old fifo_notify_flags) {
	old = fifo_notify_flags(atomic.OrWithOptions(&f.want_dequeue_notification, uint32(new), atomic.Options{Ordering: atomic.Acquire}))
	return
}

// +ego always_inline
func (f *fifo_signals) unset_want_dequeue_notification(new fifo_notify_flags) (old fifo_notify_flags) {
	old = fifo_notify_flags(atomic.AndWithOptions(&f.want_dequeue_notification, ^uint32(new), atomic.Options{Ordering: atomic.Acquire}))
	return
}

// +ego always_inline
func (o fifo_segment_offset) ptr(g *fifo_segment) (p unsafe.Pointer) {
	if o != 0 {
		p = unsafe.Add(unsafe.Pointer(g.r), o)
	}
	return
}

// +ego always_inline
func (o fifo_segment_offset_for_chunk) get(f *fifo) *fifo_chunk { return o.getg(f.g) }

// +ego always_inline
func (o fifo_segment_offset_for_chunk) getg(g *fifo_segment) *fifo_chunk {
	return (*fifo_chunk)(fifo_segment_offset(o).ptr(g))
}

// +ego always_inline
func (c *fifo_chunk) offset(g *fifo_segment) fifo_segment_offset_for_chunk {
	var o uintptr
	if c != nil {
		o = uintptr(unsafe.Pointer(c)) - uintptr(unsafe.Pointer(g.r))
	}
	return fifo_segment_offset_for_chunk(o)
}

// +ego always_inline
func (c *fifo_chunk) data() []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&c.chunk_data)), c.len)
}

type (
	segment_handle  uint64
	segment_manager struct {
		fifo_segments std.Vec[*fifo_segment]
	}
	segment_main struct {
		segment_managers std.PagedVec[*segment_manager]
	}
)

func (h segment_handle) manager() uint32 { return uint32((h >> 32) & (1<<24 - 1)) }
func (h segment_handle) index() uint32   { return uint32(h) }
func make_segment_handle(sm_index, fs_index uint32) segment_handle {
	return segment_handle(fs_index) + segment_handle(sm_index)<<32
}
func (m *app_main) find_or_create_segment(mi, fi uint32) (fs *fifo_segment) {
	m.mutex.LockIf()
	smp := m.segment_managers.Validate(uint(mi))
	m.mutex.UnlockIf()
	sm := *smp
	if sm == nil {
		sm = new(segment_manager)
		*smp = sm
	}
	fsp := sm.fifo_segments.Validate(uint(fi))
	fs = *fsp
	if fs == nil {
		fs = new(fifo_segment)
		*fsp = fs
		cfs := C.segment_manager_get_segment_w_handle(C.u64(make_segment_handle(mi, fi)))
		shr := (*fifo_segment_shared_memory)(unsafe.Pointer(cfs.h))
		fs.r = shr
		fs.slices = unsafe.Slice((*fifo_segment_shared_memory_slice)(unsafe.Pointer(&shr.slices)), shr.n_slices)
	}
	return
}
func (h segment_handle) add_del(is_del bool) SessionError {
	m := &global_main.app_main
	m.mutex.LockIf()
	defer m.mutex.UnlockIf()
	mi, fi := h.manager(), h.index()
	if is_del {
		sm := *m.segment_managers.At(uint(mi))
		free(sm.fifo_segments[fi])
	} else {
		m.find_or_create_segment(mi, fi)
	}
	return 0
}

func (h *fifo_segment_free_chunk_head) l() fifo_segment_free_chunk_head {
	return fifo_segment_free_chunk_head(
		atomic.LoadWithOptions((*uintptr)(h),
			atomic.Options{Ordering: atomic.Acquire},
		))
}
func (h *fifo_segment_free_chunk_head) cas(old, new fifo_segment_free_chunk_head) (fifo_segment_free_chunk_head, bool) {
	v, ok := atomic.CompareAndSwapWithOptions((*uintptr)(h), uintptr(old), uintptr(new),
		atomic.CompareAndSwapOptions{
			SuccessOrdering: atomic.Release,
			FailureOrdering: atomic.Acquire,
		},
	)
	return fifo_segment_free_chunk_head(v), ok
}

// ABA problem guard counter for stack.
const (
	fifo_segment_free_chunk_head_aba_inc  = 1 << 48
	fifo_segment_free_chunk_head_aba_mask = fifo_segment_free_chunk_head_aba_inc - 1
)

func (h *fifo_segment_free_chunk_head) get(f *fifo) *fifo_chunk {
	return fifo_segment_offset_for_chunk(*h & fifo_segment_free_chunk_head_aba_mask).get(f)
}
func (h *fifo_segment_free_chunk_head) pop(g *fifo_segment) *fifo_chunk {
	old := h.l()
	for {
		const (
			a = fifo_segment_free_chunk_head_aba_inc
			m = a - 1
		)
		o := fifo_segment_offset_for_chunk(old & m)
		if o == 0 {
			return nil
		}
		c := o.getg(g)
		new := fifo_segment_free_chunk_head(c.next) + ((old + a) &^ m)
		if v, ok := h.cas(old, new); ok {
			c.next = 0
			g.n_pop++
			return c
		} else {
			old = v
		}
	}
}
func (h *fifo_segment_free_chunk_head) push(g *fifo_segment, c *fifo_chunk) { h.push_batch(g, c, c, 1) }
func (h *fifo_segment_free_chunk_head) push_batch(g *fifo_segment, head, tail *fifo_chunk, count int) {
	head_offset := fifo_segment_free_chunk_head(head.offset(g))
	old := h.l()
	for {
		const (
			a = fifo_segment_free_chunk_head_aba_inc
			m = a - 1
		)
		tail.next = fifo_segment_offset_for_chunk(old & m)
		new := head_offset + ((old + a) &^ m)
		if v, ok := h.cas(old, new); ok {
			g.n_push += uint64(count)
			return
		} else {
			old = v
		}
	}
}

const (
	min_log2_fifo_chunk_bytes = 10
	max_log2_fifo_chunk_bytes = 22
	n_fifo_chunk_sizes        = 1 + max_log2_fifo_chunk_bytes - min_log2_fifo_chunk_bytes
)

func fifo_chunk_size_index_for_bytes(n_bytes int) (i int) {
	if n_bytes > 1<<min_log2_fifo_chunk_bytes {
		i = int(std.MaxLog2(uint(n_bytes))) - min_log2_fifo_chunk_bytes
	}
	return
}

func (s *fifo_segment) ptr(o uintptr) unsafe.Pointer {
	return unsafe.Add(unsafe.Pointer(s.r), o)
}
func (g *fifo_segment) alloc_aligned(n_bytes, align int) (p unsafe.Pointer) {
	m := uintptr(align - 1)
	cur_pos := g.r.byte_index
	cur_pos_align := (cur_pos + m) &^ m
	new_pos_align := cur_pos_align + ((uintptr(n_bytes) + m) &^ m)
	for {
		if new_pos_align >= g.r.max_byte_index {
			return
		}
		v, ok := atomic.CompareAndSwapWithOptions(&g.r.byte_index, cur_pos, new_pos_align,
			atomic.CompareAndSwapOptions{
				SuccessOrdering: atomic.Acquire,
				FailureOrdering: atomic.Monotonic,
			})
		if ok {
			p = g.ptr(cur_pos_align)
			return
		}
		cur_pos = v
		cur_pos_align = (cur_pos + m) &^ m
		new_pos_align = cur_pos_align + ((uintptr(n_bytes) + m) &^ m)
	}
}

func (c *fifo_chunk) next_ptr(l uintptr) (n *fifo_chunk) {
	n = (*fifo_chunk)(unsafe.Add(unsafe.Pointer(c), l))
	return
}
func (f *fifo) new_chunk(len int) (c *fifo_chunk) {
	return f.g.alloc_slice(len, int(f.r.slice_index))
}
func (f *fifo) new_after(d *fifo_chunk) (c *fifo_chunk) {
	c = f.new_chunk(int(d.len))
	if c != nil {
		d.next = c.offset(f.g)
		c.lo = d.lo + d.len
		c.len = d.len
	}
	return
}

// +ego always_inline
func (s *fifo_segment_shared_memory_slice) add_sub(g *fifo_segment, i, n int, is_sub bool) {
	l := uintptr(n) << (min_log2_fifo_chunk_bytes + i)
	if is_sub {
		atomic.Sub(&s.n_fl_chunk_bytes, l)
		atomic.Sub(&g.r.n_cached_bytes, l)
	} else {
		atomic.Add(&s.n_fl_chunk_bytes, l)
		atomic.Add(&g.r.n_cached_bytes, l)
	}
}

// +ego always_inline
func (s *fifo_segment_shared_memory_slice) add(g *fifo_segment, i, n int) { s.add_sub(g, i, n, false) }

// +ego always_inline
func (s *fifo_segment_shared_memory_slice) sub(g *fifo_segment, i, n int) { s.add_sub(g, i, n, true) }

// +ego always_inline
func (f *fifo) free_chunk(c *fifo_chunk) {
	i := fifo_chunk_size_index_for_bytes(int(c.len))
	s := &f.g.slices[f.c.shr.slice_index]
	s.free_chunk_heads[i].push(f.g, c)
	s.add(f.g, i, 1)
}
func (g *fifo_segment) alloc_slice(n_bytes, slice_index int) (c *fifo_chunk) {
	i := fifo_chunk_size_index_for_bytes(n_bytes)
	s := &g.slices[slice_index]
	free_head := &s.free_chunk_heads[i]
	for {
		if c = free_head.pop(g); c != nil {
			s.add(g, i, 1)
			return
		}
		const (
			batch_size  = 32
			chunk_align = 8
		)
		len_each_chunk_no_overhead := uintptr(1) << (min_log2_fifo_chunk_bytes + i)
		len_each_chunk_with_overhead := unsafe.Sizeof(fifo_chunk{}) + len_each_chunk_no_overhead
		len_all_chunks_with_overhead := batch_size * len_each_chunk_with_overhead

		d := (*fifo_chunk)(g.alloc_aligned(int(len_all_chunks_with_overhead), chunk_align))
		if d == nil {
			return nil
		}
		head, tail := (*fifo_chunk)(nil), d
		for i := range batch_size {
			d.lo = 0
			d.len = uint32(len_each_chunk_no_overhead)
			d.next = head.offset(g)
			head = d
			d = d.next_ptr(len_each_chunk_with_overhead)
		}
		free_head.push_batch(g, head, tail, batch_size)
		s.num_chunks[i] += batch_size
		s.add(g, i, batch_size)
	}
}
