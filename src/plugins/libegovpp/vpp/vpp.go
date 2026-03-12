package vpp

// #cgo CFLAGS: -Wno-address-of-packed-member
// #include "vpp.h"
import "C"

import (
	"cpu"
	"ego"
	"math"
	"net/ip4"
	"net/ip6"
	"runtime"
	"std"
	"sync"
	"unsafe"
)

type clib_vec struct{ c unsafe.Pointer }

func (v clib_vec) len() int             { return int(*(*uint32)(unsafe.Add(v.c, -int(unsafe.Sizeof(uintptr(0)))))) }
func clib_vec_len(p unsafe.Pointer) int { return clib_vec{c: p}.len() }
func vec_string(c *C.u8) string {
	v := clib_vec{c: unsafe.Pointer(c)}
	b := unsafe.String((*byte)(v.c), v.len())
	if l := len(b); l > 0 && b[l-1] == 0 {
		b = b[:l-1]
	}
	return string(b)
}

func cs(s string) (p *C.char) {
	if s != "" {
		p = (*C.char)(save(s + "\x00"))
	}
	return
}
func cu(s string) (p *C.u8) {
	if s != "" {
		p = (*C.u8)(save[ret](s + "\x00"))
	}
	return
}

// Convert c string slice to null terminated C.
func css(s []string) (p **C.char) {
	if len(s) > 0 {
		ss := make([]*C.char, 1+len(s))
		for i := range ss {
			ss[i] = cs(s[i])
		}
		p = (**C.char)(unsafe.Pointer(&s[0]))
	}
	return
}

// Convert c string slice to C non-null terminated C.
// Return length.
func cssl(s []string) (p **C.char, l C.int) {
	if l = C.int(len(s)); l > 0 {
		ss := make([]*C.char, len(s))
		for i := range ss {
			ss[i] = cs(s[i])
		}
		p = (**C.char)(unsafe.Pointer(&ss[0]))
	}
	return
}

type clib_error struct{ c *C.clib_error_t }

func cerror_ret(s string) (c clib_error) {
	c.c = C._clib_error_return(nil, 0, 0, "", "%s", s+"\x00")
	return
}
func cerror(e error) (c clib_error) {
	if e != nil {
		c = cerror_ret(e.Error())
	}
	return
}
func (c clib_error) Error() string {
	ptr := unsafe.Pointer(C.format(0, "%U%c", C.format_clib_error, c.c, 0))
	return unsafe.String((*byte)(ptr), clib_vec{c: ptr}.len())
}
func cerrorf(fmt string, args ...interface{}) clib_error {
	return cerror_ret(std.Formats(fmt, args...) + "\x00")
}
func clib_go_error(c unsafe.Pointer) error {
	if c == nil {
		return nil
	} else {
		var e clib_error
		e.c = (*C.clib_error_t)(c)
		return e
	}
}

type unformat_input C.unformat_input_t

func (i *unformat_input) parse_input() (in std.ParseInput) {
	const unformat_end_of_input = ^C.uword(0)
	if ii := i.index; ii != unformat_end_of_input {
		i.index = unformat_end_of_input // consume all the input
		b := unsafe.Pointer(i.buffer)
		p := unsafe.Add(b, ii)
		l := C.uword(clib_vec{c: b}.len()) - ii
		in = std.NewParseInput(string(unsafe.Slice((*byte)(p), l)))
	}
	return
}

type main struct {
	app_main  app_main
	node_main node_main
}

var global_main main

func (m *main) init() {
	m.app_main.init()
	m.node_main.init()
}

type (
	Main      struct{ c *C.vlib_main_t }
	node_main struct {
		init_once    sync.Once
		mutex        *std.Spinlock
		Nodes        std.PagedPool[Node]
		node_by_name map[string]*Node
		node_by_c    map[node_c]*Node
	}
	node_c      struct{ c *C.vlib_node_t }
	NodeRuntime struct{ c *C.vlib_node_runtime_t }
	Node        struct {
		node_c
		f func(m Main, r NodeRuntime, f *InputFrame) int
	}
	ErrorCode               uint16
	ErrorCodes[V ~vec[any]] vec[[V]ErrorCode]
)

func (m *node_main) init() {
	m.init_once.Do(func() {
		m.mutex = new(std.Spinlock)
	})
}

func (r NodeRuntime) Index() uint32   { return uint32(r.c.node_index) }
func (r NodeRuntime) Opaque() uintptr { return uintptr(r.c.opaque) }
func (n *Node) init() {
	m := &global_main.node_main
	m.init()
	m.mutex.Lock()
	m.node_by_c[n.node_c] = n
	m.node_by_name[dup(vec_string(n.c.name))] = n
	m.mutex.Unlock()
}
func (r NodeRuntime) Node(m Main) (n *Node) {
	n = global_main.node_main.Nodes.At(uint(r.Opaque()))
	if n.c == nil {
		n.c = C.ego_vlib_get_node(m.c, C.u32(r.c.node_index))
		n.init()
	}
	return
}
func (m Main) NodeWithName(name string) *Node {
	nm := &global_main.node_main
	n, ok := nm.node_by_name[name]
	if !ok {
		n, _ = nm.Nodes.Get()
		n.c = C.vlib_get_node_by_name(m.c, cu(name))
		if n.c == nil {
			panic("no node with given name")
		}
		n.init()
	}
	return n
}
func NumThreads() int {
	var n C.u32
	C.ego_vlib_threads(&n)
	return int(n)
}
func (m Main) Thread() int { return int(C.ego_vlib_threads(nil)) }
func (r NodeRuntime) Error(i int) ErrorCode {
	ce := unsafe.Pointer(r.c.errors)
	es := unsafe.Slice((*ErrorCode)(ce), clib_vec{c: ce}.len())
	return es[i]
}
func (c ErrorCode) Count(m Main, r NodeRuntime, n int) {
	C.ego_vlib_error_count(m.c, C.uword(r.c.node_index), C.uword(c), C.uword(n))
}
func (m Main) Suspend(secs float64) { C.ego_vlib_process_suspend(m.c, C.f64(secs)) }
func (m Main) SignalEvent(node_index uint32, kind, data uintptr) {
	C.ego_vlib_process_signal_event(m.c, C.uword(node_index), C.uword(kind), C.uword(data))
}
func (m Main) WaitForEvent(timeout_secs float64) float64 {
	return float64(C.ego_vlib_process_wait_for_event_or_clock(m.c, C.f64(timeout_secs)))
}
func (m Main) ReceiveEvent() (kind uintptr, data []uintptr) {
	var v unsafe.Pointer
	kind = uintptr(C.ego_vlib_process_get_events(m.c, (**C.uword)(unsafe.Pointer(&v))))
	if kind != ^uintptr(0) {
		l := clib_vec_len(v)
		data = make[ret]([]uintptr, l)
		copy(data, unsafe.Slice((*uintptr)(v), l))
		C.ego_vlib_process_put_event_data(m.c, v)
	}
	return
}
func (m Main) CurrentNodeIndex() uint32 {
	return uint32(C.ego_vlib_get_current_process_node_index(m.c))
}

const (
	NODE_STATE_POLLING node_state = iota
	NODE_STATE_INTERRUPT
	NODE_STATE_DISABLED
)

func (n node_c) Enable(m Main, new bool) (old bool) {
	old_state := node_state(C.ego_vlib_node_get_state(m.c, n.c.index))
	old = old_state != NODE_STATE_DISABLED
	if old != new {
		new_state := NODE_STATE_POLLING
		if !new {
			new_state = NODE_STATE_DISABLED
		}
		C.ego_vlib_node_set_state(m.c, n.c.index, C.vlib_node_state_t(new_state))
	}
	return
}

type (
	frame_c         struct{ c *C.vlib_frame_t }
	Frame[S, V any] struct {
		c frame_c
		S *S
		V []V
	}
)

const FrameSize = int(C.VLIB_FRAME_SIZE)

func (f *Frame[S, V]) IsNil() bool { return f.c.c == nil }
func (f *Frame[S, V]) SetNil()     { f.c.c = nil }
func (f *Frame[S, V]) init(c frame_c) {
	var (
		p        [2]unsafe.Pointer
		len, cap uintptr
	)
	if c.c != nil {
		C.ego_vlib_frame_args(c.c, &p[0], &p[1])
		len = uintptr(c.c.n_vectors)
		cap = uintptr(FrameSize)
	}
	f.c = c
	f.S = (*S)(p[0])
	f.V = unsafe.Slice((*V)(p[1]), cap)[:len]
}

type (
	init_function_type int
	InitFunc           struct {
		Type       init_function_type
		Name       string
		Func       func(Main) error
		RunsBefore []string
		RunsAfter  []string
		InitOrder  []string
		wrap       func(Main) *clib_error
	}
	ConfigFunc struct {
		Name    string
		Func    func(m Main, in *std.ParseInput) error
		IsEarly bool
		wrap    func(m Main, i *unformat_input) clib_error
	}
	CliCommand struct {
		// Command path (e.g. "show something").
		// Spaces delimit elements of path.
		Path string

		// Function.
		Func func(m Main, in *std.ParseInput) error

		// Wrapped function called by C code.
		wrap func(m Main, i *unformat_input, _ unsafe.Pointer) clib_error

		// Short/long help strings.
		ShortHelp, LongHelp string

		IsMPSafe bool
	}
)

func (f *InitFunc) Free()   { free(f.wrap); free(f.Func); free(f) }
func (f *ConfigFunc) Free() { free(f.wrap); free(f.Func); free(f) }
func (f *CliCommand) Free() { free(f.wrap); free(f.Func); free(f) }

const (
	INIT_FUNCTION init_function_type = iota
	MAIN_LOOP_ENTER_FUNCTION
	MAIN_LOOP_EXIT_FUNCTION
	MAIN_WORKER_INIT_FUNCTION
	MAIN_NUM_WORKERS_CHANGE_FUNCTION
	MAIN_API_INIT_FUNCTION
)

func AddInitFunc(f *InitFunc) *InitFunc {
	f = save(f)
	f.Func = save(f.Func)
	C.ego_add_init_func(C.int(f.Type), (*C.vlib_init_function_t)(f.Func), cs(f.Name), css(f.RunsBefore), css(f.RunsAfter), css(f.InitOrder))
	return f
}
func (f *ConfigFunc) wrapper(m Main, i *unformat_input) clib_error {
	in := i.parse_input()
	return cerror(f.Func(m, &in))
}
func AddConfigFunc(f *ConfigFunc) *ConfigFunc {
	f = save(f)
	f.Func = save(f.Func)
	f.wrap = save(f.wrapper)
	C.ego_add_config_func((*C.vlib_config_function_t)(f.wrap), cs(f.Name), C.int(ifelse(f.IsEarly)))
	return f
}

func (m Main) Logf(format string, args ...any) {
	var b std.Buffer
	b.Format(format, args...)
	b.Add(0)
	C.vlib_cli_output(m.c, string(b.Slice()))
	b.Free()
}
func (m Main) Log(args ...any) { m.Logf("", args...) }

func (f *CliCommand) wrapper(m Main, i *unformat_input, _ unsafe.Pointer) clib_error {
	in := i.parse_input()
	return cerror(f.Func(m, &in))
}

func AddCliCommand(f *CliCommand) *CliCommand {
	f = save(f)
	f.Func = save(f.Func)
	f.wrap = save(f.wrapper)
	C.ego_add_cli_command((*C.vlib_cli_command_function_t)(f.wrap), cs(f.Path), cs(f.ShortHelp), cs(f.LongHelp), C.int(ifelse(f.IsMPSafe)))
	return f
}

// +ego { aligned cache }
type Buffer struct {
	// First 64 byte line.
	current_data_offset              int16
	current_length                   uint16
	flags                            uint32
	flow_id                          uint32
	ref_count                        uint8
	buffer_pool_index                uint8
	Error                            ErrorCode
	next_buffer                      uint32
	current_config_index_punt_reason uint32
	Opaque1                          [10]uint32

	// 2nd 64 byte line.
	trace_handle                            uint32
	total_length_not_including_first_buffer uint32
	Opaque2                                 [14]uint32

	// Lines 3-4 buffer data prepend area.
	pre_data [128]byte

	buffer_data [0]byte
}

func (m Main) buffer_ptr(o uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(m.c.buffer_main.buffer_mem_start) + o)
}
func (o uintptrs[T]) buffer_ptrs(m Main) vec[[T]unsafe.Pointer] {
	return vec[[T]unsafe.Pointer](uintptr(m.c.buffer_main.buffer_mem_start) + o)
}

type (
	// Offset from buffer memory base in units of cache lines.
	BufferIndex uint32
	// SIMD vector of buffer indices.
	BufferIndices[T ~vec[[...]any]] vec[[T]BufferIndex] // +ego { aligned 1 }
	BufferPtrs[T ~vec[[...]any]]    vec[[T]*Buffer]
	uintptrs[T ~vec[[...]any]]      vec[[T]uintptr]
)

func (b BufferIndex) ptr(m Main, o uintptr) unsafe.Pointer {
	return m.buffer_ptr(o + uintptr(b)<<cpu.Log2CacheLineBytes)
}
func (b BufferIndices[T]) ptr(m Main, o uintptrs[T]) vec[[T]unsafe.Pointer] {
	o += uintptrs[T](b) << cpu.Log2CacheLineBytes
	return o.buffer_ptrs(m)
}
func (b BufferIndex) Buffer(m Main) *Buffer             { return (*Buffer)(b.ptr(m, 0)) }
func (b BufferIndices[T]) Buffers(m Main) BufferPtrs[T] { return BufferPtrs[T](b.ptr(m, 0)) }

func (b *Buffer) data() unsafe.Pointer     { return unsafe.Pointer(&b.buffer_data) }
func (b *Buffer) cur_data() unsafe.Pointer { return unsafe.Add(b.data(), b.current_data_offset) }
func (b *Buffer) Data() []byte {
	return unsafe.Slice((*byte)(b.cur_data()), b.current_length)
}
func (b BufferPtrs[T]) Len() vec[[T]uint16] { return b.current_length }
func (b BufferPtrs[T]) data() vec[[T]unsafe.Pointer] {
	return vec[[T]unsafe.Pointer](&b.buffer_data)
}
func (b BufferPtrs[T]) cur_data() vec[[T]unsafe.Pointer] {
	return unsafe.Add(b.data(), b.current_data_offset)
}

func prefetch_ptr(p unsafe.Pointer) {
	cpu.PrefetchPtrWithOptions(p, cpu.PrefetchOptions{Locality: cpu.PrefetchL2})
}
func (b *Buffer) Prefetch() {
	p := unsafe.Pointer(b)
	// Prefetch first 2 lines.
	prefetch_ptr(unsafe.Add(p, 0))
	prefetch_ptr(unsafe.Add(p, 64))
}
func (b BufferPtrs[T]) Prefetch() {
	for i := range b {
		b[i].Prefetch()
		prefetch_ptr(b[i].cur_data())
	}
}

func (b *Buffer) Ip4() *ip4.Header { return (*ip4.Header)(b.cur_data()) }
func (b *Buffer) Ip6() *ip6.Header { return (*ip6.Header)(b.cur_data()) }

func (b BufferPtrs[T]) Ip4() ip4.Headers[T] { return (ip4.Headers[T])(b.cur_data()) }

type (
	Ip4BufferIndex                BufferIndex
	Ip6BufferIndex                BufferIndex
	Ip4BufferIndices[T ~vec[any]] vec[[T]Ip4BufferIndex] // +ego { align 1 }
	Ip6BufferIndices[T ~vec[any]] vec[[T]Ip6BufferIndex] // +ego { align 1 }
	Ip4BufferFrame                Frame[struct{}, Ip4BufferIndex]
	Ip6BufferFrame                Frame[struct{}, Ip6BufferIndex]
)

func (x Ip4BufferIndex) Header(m Main) *ip4.Header { return BufferIndex(x).Buffer(m).Ip4() }
func (x Ip6BufferIndex) Header(m Main) *ip6.Header { return BufferIndex(x).Buffer(m).Ip6() }
func (x Ip4BufferIndices[T]) Header(m Main) ip4.Headers[T] {
	return BufferIndices[T](x).Buffers(m).Ip4()
}
func (f *Ip4BufferFrame) BufferIndices() (b []BufferIndex) {
	if l := len(f.V); l > 0 {
		b = unsafe.Slice((*BufferIndex)(unsafe.Pointer(&f.V[0])), l)
	}
	return
}

type (
	node_type            uint8
	node_state           uint8
	node_flags           uint16
	NodeConfig[S, V any] struct {
		Func         func(m Main, r NodeRuntime, f *Frame[S, V]) int
		Name         string
		SiblingOf    string
		Type         node_type
		State        node_state
		ProtocolHint uint8
		Errors       []string
		NextNodes    []string
		NextThreads  []int
	}
	InputNodeConfig     = NodeConfig[struct{}, struct{}]
	InputFrame          = Frame[struct{}, struct{}]
	Next                uint32
	NextIndex[S, V any] Next

	BufferFrame                    = Frame[struct{}, BufferIndex]
	BufferNextIndex                = NextIndex[struct{}, BufferIndex]
	BufferNextIndices[L ~vec[any]] vec[[L]BufferNextIndex]
)

const (
	NODE_TYPE_INTERNAL node_type = iota
	NODE_TYPE_INPUT
	NODE_TYPE_PRE_INPUT
	NODE_TYPE_PROCESS
)

func (c *NodeConfig[S, V]) Add() {
	next_names, n_next_names := cssl(c.NextNodes)
	error_names, n_errors := cssl(c.Errors)

	var next_threads_ptr *C.u32
	if len(c.NextThreads) > 0 {
		nts := makes([]C.u32, len(c.NextThreads))
		next_threads_ptr = &nts[0]
		for i := range nts {
			nts[i] = C.u32(c.NextThreads[i])
		}
	}
	var (
		tmp_s S
		tmp_v V
	)
	nm := &global_main.node_main
	n, ni := nm.Nodes.Get()
	clo := save(c.Func)
	*(*ego.ClosureStruct)(unsafe.Pointer(&n.f)) = *(*ego.ClosureStruct)(unsafe.Pointer(&clo))
	cf := &C.ego_vlib_node_config_t{
		state:        C.int(c.State),
		scalar_size:  C.int(unsafe.Sizeof(tmp_s)),
		vector_size:  C.int(unsafe.Sizeof(tmp_v)),
		f:            (*C.vlib_node_function_t)(c_node_dispatch),
		name:         cs(c.Name),
		n_next_nodes: C.int(n_next_names),
		next_nodes:   next_names,
		next_threads: next_threads_ptr,
		n_errors:     C.int(n_errors),
		errors:       error_names,
		opaque:       C.uword(ni),
	}
	// Positive vector size implies internal type (if unspecified).
	if cf.vector_size > 0 {
		c.Type = NODE_TYPE_INTERNAL
	} else if c.Type != NODE_TYPE_INPUT {
		c.Type = NODE_TYPE_INPUT
	}
	cf.typ = C.int(c.Type)
	C.ego_add_node(cf)
}
func AddNode[S, V any](cf *NodeConfig[S, V]) { cf.Add() }

const ThreadIndexUnspecified = ^ThreadIndex(0)

type ThreadIndex uint32

func (n *Node) AddNextWithThread(m Main, next string, thread ThreadIndex, slot int) {
	if C.uword(slot) != C.vlib_node_add_named_next_with_thread_and_slot(m.c, C.uword(n.c.index), cs(next), C.uword(thread), C.uword(slot)) {
		panic("add fails")
	}
}
func (n *Node) AddNext(m Main, next string, slot int) {
	n.AddNextWithThread(m, next, ThreadIndexUnspecified, slot)
}

const (
	PacketGeneratorEditKindIp4 = iota
	PacketGeneratorEditKindIp6
)

func (n *Node) SetPacketGeneratorEditKind(vm Main, kind int) {
	pn := C.ego_pg_get_node(C.uword(n.c.index))
	var fn C.unformat_function_t
	switch kind {
	case PacketGeneratorEditKindIp4:
		fn = C.unformat_pg_ip4_header
	case PacketGeneratorEditKindIp6:
		fn = C.unformat_pg_ip6_header
	default:
		panic("not yet")
	}
	pn.unformat_edit = (*C.unformat_function_t)(fn)
}

func c_node_dispatch(m_ *C.vlib_main_t, r_ *C.vlib_node_runtime_t, f_ *C.vlib_frame_t) C.uword {
	m, r, fc := Main{c: m_}, NodeRuntime{c: r_}, frame_c{c: f_}
	n := r.Node(m)
	var f Frame[struct{}, struct{}]
	f.init(fc)
	return C.uword(n.f(m, r, &f))
}

func (x NextIndex[S, V]) GetFrame(m Main, r NodeRuntime) (f Frame[S, V]) {
	const allocate_new = 1
	nf := C.ego_vlib_node_runtime_get_next_frame(m.c, r.c, C.u32(x), nil)
	if nf != nil {
		nf.deferment_state = C.u16(C.VLIB_FRAME_DEFERRED_UNMARKED)
		f.c.c = nf
	} else {
		f.c.c = C.vlib_get_next_frame_internal(m.c, r.c, C.u32(x), allocate_new)
	}
	f.init(f.c)
	return
}
func (f *Frame[S, V]) validate(m Main, r NodeRuntime, x NextIndex[S, V]) (ok bool) {
	if f.c.c == nil {
		return
	}
	var ds C.u32
	nf := C.ego_vlib_node_runtime_get_next_frame(m.c, r.c, C.u32(x), &ds)
	ok = nf == nil
	if !ok {
		ok = ds == C.u32(C.VLIB_FRAME_DEFERRED_ALREADY_PUT)
	}
	if ok {
		f.c.c = nil
	}
	return
}

type PutFrameConfig struct {
	DeferEnable bool
	DeferNext   bool // defer put until next main loop
	DeferAlways bool // defer put until unspecified time in the future
}

func (x NextIndex[S, V]) PutFrame(m Main, r NodeRuntime, n int) {
	x.PutFrameWithConfig(m, r, n, &PutFrameConfig{})
}
func (x NextIndex[S, V]) PutFrameWithConfig(m Main, r NodeRuntime, n int, cf *PutFrameConfig) {
	if cf.DeferEnable {
		flags := C.u8(0)
		if cf.DeferNext {
			flags |= C.u8(C.VLIB_DEFER_PUT_NEXT_FRAME_DEFER_NEXT)
			if cf.DeferAlways {
				flags |= C.u8(C.VLIB_DEFER_PUT_NEXT_FRAME_DEFER_ALWAYS)
			}
		}
		C.vlib_defer_put_next_frame(m.c, r.c, C.u32(x), C.u32(n), flags)
	} else {
		C.vlib_put_next_frame(m.c, r.c, C.u32(x), C.u32(FrameSize-n))
	}
}

// +ego always_inline
func compress[L ~vec[any], T any](ok vec[[L]bool], x, y vec[[L]T]) (z vec[[L]T]) {
	return asm[call](
		func(value vec[[L]T], mask vec[[L]bool], passthru vec[[L]T]) vec[[L]T],
		"llvm.experimental.vector.compress.%0",
		x, ok, y,
	)
}

const (
	frame_cache_log2_n_bins = 4
	frame_cache_n_bins      = 1 << frame_cache_log2_n_bins
	// Re-compute stats every 2^8 puts (e.g. up to 2^8 * 256 = 2^16 vectors).
	frame_log2_n_put_per_stats = 8
	frame_n_put_per_stats      = 1 << frame_log2_n_put_per_stats
)

type (
	SparseFrameCache[S, V any, L ~vec[any]] struct {
		// Maps sparse next index to dense next index into frame vector, etc.
		m std.IntMap[vec[[32]any], NextIndex[S, V], uint8]
		// Frames one for each dense next index.
		f std.Vec[Frame[S, V]]
		frame_cache_common
	}
	FrameCache[S, V any, L ~vec[any]] struct {
		// Frames one for each dense next index.
		f std.Vec[Frame[S, V]]
		frame_cache_common
	}
	frame_cache_common struct {
		// Vector count for corresponding frame.
		n_vectors  std.Vec[uint32]
		next_stats std.Vec[frame_cache_next_stats]
		// Average input vector length for each bin.
		rx_ave_vec_len      [frame_cache_n_bins]float32
		n_put               uint64
		defer_put_threshold int
	}
	frame_cache_next_stats struct {
		// Loop count for this next.  Frame will be put when this count reaches tx_n_loops_per_put.
		n_loops uint8
		// Number of main loop interations needed on average to fill
		// this next to average vector length.
		tx_n_loops_per_put uint8

		// Accumulated vector count for this next.  Cleared on each statistics update.
		n_vectors_this_next uint32

		// Average output vector length for this next.  Circular
		// buffer of last 256 bins.
		tx_ave_vec_len [frame_cache_n_bins]float32
	}
	FrameCacheConfig struct {
		DeferPutThreshhold int
	}
)

func (c *frame_cache_common) Init(cf *FrameCacheConfig) {
	c.defer_put_threshold = max(1, min(cf.DeferPutThreshhold, FrameSize))
}

func (c *SparseFrameCache[S, V, L]) Begin(vm Main, r NodeRuntime, enable_defer bool) {
	if enable_defer {
		for i := range c.m.Len() {
			if c.f[i].validate(vm, r, c.m.Key(i)) {
				c.n_vectors[i] = 0
			}
		}
	}
}

// +ego { no_inline }
func (c *SparseFrameCache[S, V, L]) add_slow(x NextIndex[S, V]) (fi uint) {
	fi = c.m.Len()
	c.m.Set(x, uint8(fi))
	c.f.Validate(fi)
	c.n_vectors.Validate(fi)
	c.next_stats.Validate(fi)
	c.f[fi].SetNil()
	c.n_vectors[fi] = 0
	return
}

// +ego { align 1 }
type unaligned[L ~vec[any], V any] vec[[L]V]

// +ego { always_inline }
func (c *SparseFrameCache[S, V, L]) Add(vm Main, r NodeRuntime, v vec[[L]V], x vec[[L]NextIndex[S, V]], ok vec[[L]bool], enable_defer bool) {
	ok_left := ok
	for {
		xi := len(ok) - 1 - reduce[clz](ok_left)
		if xi < 0 {
			break
		}
		this_x := x[xi]
		this_ok := x == this_x && ok_left
		ok_left = ok_left && !this_ok

		fi, fok := c.m.Get(this_x)
		if !fok {
			fi = uint8(c.add_slow(this_x))
		}
		if c.f[fi].IsNil() {
			c.f[fi] = this_x.GetFrame(vm, r)
		}
		frame := &c.f[fi]
		nv := c.n_vectors[fi]
		xs := &c.next_stats[fi]

		// This is slow on zen4 cores...
		// ifelse(this_ok, frame.V[nv]) = v
		var zero vec[[L]V]
		u := compress[vec[[L]any]](this_ok, v, zero)
		*(*unaligned[L, V])(unsafe.Pointer(&frame.V[nv])) = unaligned[L, V](u)
		nv_this := uint32(reduce[add](this_ok))
		nv += nv_this
		xs.n_vectors_this_next += uint32(nv_this)
		if dt := c.defer_put_threshold; enable_defer && dt >= 0 && dt <= FrameSize {
			if nv >= uint32(dt) {
				this_x.PutFrameWithConfig(vm, r, dt, &PutFrameConfig{DeferEnable: enable_defer})
				nv -= uint32(dt)
				if nv == 0 {
					c.f[fi].SetNil()
				} else {
					c.f[fi] = this_x.GetFrame(vm, r)
					for i := uint32(0); i < nv; i++ {
						frame.V[i] = u[nv_this-nv+i]
					}
				}
			}
		}
		c.n_vectors[fi] = nv
	}
}

// +ego always_inline
func ctpop[L ~vec[any], T any](x vec[[L]T]) vec[[L]T] {
	return asm[call](
		func(x vec[[L]T]) vec[[L]T],
		"llvm.ctpop.%0",
		x,
	)
}

func (c *SparseFrameCache[S, V, L]) Put(m Main, r NodeRuntime, defer_frames bool) {
	cf := &PutFrameConfig{DeferEnable: defer_frames, DeferNext: defer_frames}
	for i := range c.m.Len() {
		if defer_frames {
			xs := &c.next_stats[i]
			xs.n_loops++
			if xs.n_loops >= xs.tx_n_loops_per_put {
				xs.n_loops = 0
				cf.DeferNext = false
			}
		}
		ki := c.m.Key(i)
		nv := int(c.n_vectors[i])
		ki.PutFrameWithConfig(m, r, nv, cf)
	}
	if !defer_frames {
		c.m.Reset()
	} else {
		c.update_stats()
	}
}

func (c *frame_cache_common) update_stats() {
	c.n_put++
	n0, n1 := c.n_put/frame_n_put_per_stats, c.n_put%frame_n_put_per_stats
	if n1 == 0 {
		bin := n0 % frame_cache_n_bins
		total_count := uint64(0)
		for _, xs := range &c.next_stats {
			n := xs.n_vectors_this_next
			xs.n_vectors_this_next = 0
			total_count += uint64(n)
			xs.tx_ave_vec_len[bin] = float32(n) / frame_n_put_per_stats
		}
		c.rx_ave_vec_len[bin] = float32(total_count) / frame_n_put_per_stats
		n_bin := min(n0, frame_cache_n_bins)
		for _, xs := range &c.next_stats {
			// Compute average and standard deviation of input to output ratio.
			var sum, sum2 float64
			for j := range n_bin {
				bin_j := (bin - j) % frame_cache_n_bins
				// Ratio of average input vector length to average output vector length for this next.
				x := float64(c.rx_ave_vec_len[bin_j]) / float64(xs.tx_ave_vec_len[bin_j])
				sum += x
				sum2 += x * x
			}
			ave := sum / float64(n_bin)
			rms := math.Sqrt(sum2/float64(n_bin) - ave*ave)
			// Compute number of trips around main needed loop to accumulate average input vector length.
			tx_n_loop := ave - 2*rms
			tx_n_loop = max(tx_n_loop, 1)
			tx_n_loop = min(tx_n_loop, 16)
			xs.tx_n_loops_per_put = uint8(tx_n_loop + .5)
		}
	}
}

func (c *FrameCache[S, V, L]) Add(vm Main, r NodeRuntime, v vec[[L]V], nexts vec[[L]NextIndex[S, V]], ok vec[[L]bool], enable_defer bool) {
	var nv vec[[L]uint32]

	nv = ifelse(ok && nexts < NextIndex[S, V](len(c.n_vectors)), c.n_vectors[nexts])

	// Validate frames for all nexts with 0 length.
	empty_nexts := nv == 0 && ok
	for {
		i := len(empty_nexts) - 1 - reduce[clz](empty_nexts)
		if i < 0 {
			break
		}
		xi := nexts[i]
		uxi := uint(xi)
		c.f.Validate(uxi)
		c.n_vectors.Validate(uxi)
		c.next_stats.Validate(uxi)
		c.f[xi] = xi.GetFrame(vm, r)
		empty_nexts = empty_nexts && nexts != xi
	}

	x := vec[[L]uint32](nexts)
	y := shuffle(x, x, iota%len(L), 16)
	z := ctpop[vec[[16]any], uint32](cpu.X86_conflict_detect_32(y))
	i := vec[[L]uint32](shuffle(z, z, iota, len(L)))

	nv += i

	this_ok := ok && nv < uint32(FrameSize)
	f := &c.f[nexts]
	ifelse(this_ok, f.V[nv]) = v
	nv += 1
	ifelse(this_ok, c.n_vectors[nexts]) = nv

	if dt := c.defer_put_threshold; enable_defer && dt >= 0 && dt <= FrameSize {
		overflow := nv >= uint32(dt) && ok
		for {
			i := len(overflow) - 1 - reduce[clz](overflow)
			if i < 0 {
				break
			}
			xi, nvi := nexts[i], nv[i]
			xi.PutFrameWithConfig(vm, r, dt, &PutFrameConfig{DeferEnable: enable_defer})
			c.next_stats[xi].n_vectors_this_next += uint32(dt)
			nvi -= uint32(dt)
			if nvi == 0 {
				c.n_vectors[xi] = 0
			} else {
				c.f[xi] = xi.GetFrame(vm, r)
				c.n_vectors[xi] = nvi
				var zero vec[[L]V]
				u := compress[vec[[L]any]](nexts == xi, v, zero)
				*(*unaligned[L, V])(unsafe.Pointer(&c.f[xi].V[0])) = unaligned[L, V](u)
			}
			overflow = overflow && xi != nexts
		}
	}
}

func (c *FrameCache[S, V, L]) Begin(vm Main, r NodeRuntime, enable_defer bool) {
	if enable_defer {
		for i := range c.f {
			if c.f[i].validate(vm, r, NextIndex[S, V](i)) {
				c.n_vectors[i] = 0
			}
		}
	}
}

func (c *FrameCache[S, V, L]) Put(m Main, r NodeRuntime, defer_frames bool) {
	cf := &PutFrameConfig{DeferEnable: defer_frames, DeferNext: defer_frames}
	for i := range c.n_vectors {
		if defer_frames {
			xs := &c.next_stats[i]
			xs.n_loops++
			if xs.n_loops >= xs.tx_n_loops_per_put {
				xs.n_loops = 0
				cf.DeferNext = false
			}
		}
		nv := c.n_vectors[i]
		xi := NextIndex[S, V](i)
		xi.PutFrameWithConfig(m, r, int(nv), cf)
		if !cf.DeferNext {
			c.next_stats[xi].n_vectors_this_next += nv
			c.n_vectors[xi] = 0
		}
	}
	if defer_frames {
		c.update_stats()
	}
}

func (m Main) BufferAlloc(b []BufferIndex) {
	C.ego_vlib_buffer_alloc(m.c, (*C.u32)(&b[0]), C.u32(len(b)))
}
func (m Main) BufferFree(b []BufferIndex) {
	C.ego_vlib_buffer_free(m.c, (*C.u32)(&b[0]), C.u32(len(b)))
}

// Dummy buffers for aligning frames to vector length boundaries.
// +ego { thread-local }
var vec_align_buffers std.Vec[BufferIndex]

func (m Main) VecAlignBuffers(n int) []BufferIndex {
	if l := len(vec_align_buffers); l < n {
		vec_align_buffers.ValidateLeni(n)
		m.BufferAlloc(vec_align_buffers[l:])
	}
	return vec_align_buffers[:n]
}
func (m Main) VecAlignIp4Buffers(n int) []Ip4BufferIndex {
	b := m.VecAlignBuffers(n)
	return unsafe.Slice((*Ip4BufferIndex)(unsafe.Pointer(&b[0])), n)
}

// Special section where vpp can find registration:
// +ego { section(.vlib_plugin_r2) }
type PluginRegistration struct {
	DefaultDisabled   int
	Version           string
	VersionRequired   string
	Overrides         string
	EarlyInitFunction string
	Description       string
	LoadAfter         string
}

var plugin = PluginRegistration{
	Version:           "0.0",
	Description:       "ego vpp plugin",
	EarlyInitFunction: "vpp.early_init",
}

func early_init(vm *C.vlib_main_t, pi *C.plugin_info_t) (ce clib_error) {
	cf := &runtime.OsConfig{
		HeapLock: true,
	}
	if runtime.OsInit(cf) {
		runtime.PackageInitFunctions()
	}
	p := runtime.Plugin{Path: vec_string(pi.filename)}
	if err := p.LoadWithHandle(pi.handle); err != nil {
		ce = cerror(err)
	}
	return
}
