package spray

import (
	"container/hash"
	"cpu"
	"net/ip"
	"net/ip4"
	"std"
	"unsafe"
	. "vpp"
)

type (
	main struct {
		std.TestConfig
		input_node                     *Node
		ip4_spray_node                 *Node
		ip4_spray_frame_cache          FrameCache[struct{}, Ip4BufferIndex, vec[[ip4_hash_vec_len]any]]
		iter, n_iter, thread, n_thread int
		hash_map                       ip4_hash_map[vec[[ip4_hash_vec_len]any]]
	}
	s struct{ a, b uint32 }
	v struct{ c, d uint64 }
)

func (s *s) Format(b *std.Buffer) { b.Format("{s %d %d}", s.a, s.b) }
func (s *v) Format(b *std.Buffer) { b.Format("{v %d %d}", s.c, s.d) }

func (m *main) input(vm Main, r NodeRuntime, _ *InputFrame) int {
	x := NextIndex[s, v](m.thread)
	f := x.GetFrame(vm, r)
	nv := min(cap(f.V), m.Iterations-m.iter)
	f.V = f.V[:nv]
	*f.S = s{a: 170, b: 180}
	f.S.a += uint32(m.iter)
	for i := range nv {
		f.V[i] = v{c: uint64(1700 + i), d: uint64(1800 + m.iter + i)}
	}
	x.PutFrame(vm, r, nv)
	m.thread++
	if m.thread >= m.n_thread {
		m.thread = 0
	}
	m.iter += nv
	if m.iter >= m.Iterations {
		m.input_node.Enable(vm, false)
	}
	if m.Verbose > 1 {
		std.Log("spray input", vm.Thread(), f.S, &f.V[0])
	}
	return nv
}
func (m *main) drop(vm Main, r NodeRuntime, rxf *Frame[s, v]) int {
	if m.Verbose > 1 {
		ti := vm.Thread()
		std.Log("spray", ti, rxf.S, len(rxf.V))
		for i, v := range &rxf.V {
			std.Logf("  %d %v\n", ti, v)
		}
	}
	nv := len(rxf.V)
	ErrorCode(0).Count(vm, r, nv)
	return nv
}

type ip4_hash_map[V ~vec[any]] struct {
	h   hash.Map[uint128, uint64]
	aes hash.Aes
}

// +ego { always_inline }
func (t *ip4_hash_map[V]) hashv(k [2]vec[[V]uint64]) (h [2]vec[[V]uint64]) {
	n_left := len(V)
	kp0, kp1 := unsafe.Pointer(&k[0]), unsafe.Pointer(&k[1])
	hp0, hp1 := unsafe.Pointer(&h[0]), unsafe.Pointer(&h[1])
	o := uintptr(0)
	for n_left > 0 {
		switch {
		case cpu.HasAesEncryptx4() && n_left >= 8:
			k0 := *(*vec[[8]uint64])(unsafe.Add(kp0, o))
			k1 := *(*vec[[8]uint64])(unsafe.Add(kp1, o))
			x0 := vec[[4 * 4]uint32](shuffle(k0, k1, 8*(iota%2)+(iota/2)+0*4)) // 0x0 0x8 0x1 0x9 ... 0x3 0xb
			x1 := vec[[4 * 4]uint32](shuffle(k0, k1, 8*(iota%2)+(iota/2)+1*4)) // 0x4 0xc 0x6 0xd ... 0x7 0xf
			x0, x1 = t.aes.Hash2x4(x0, x1)
			y0 := shuffle(vec[[8]uint64](x0), vec[[8]uint64](x1), 2*iota+0)
			y1 := shuffle(vec[[8]uint64](x0), vec[[8]uint64](x1), 2*iota+1)
			*(*vec[[8]uint64])(unsafe.Add(hp0, o)) = y0
			*(*vec[[8]uint64])(unsafe.Add(hp1, o)) = y1
			n_left -= 8
			o += 8 * 8
		case cpu.HasAesEncryptx4() && n_left >= 4:
			k0 := *(*vec[[4]uint64])(unsafe.Add(kp0, o))
			k1 := *(*vec[[4]uint64])(unsafe.Add(kp1, o))
			x0 := vec[[4 * 4]uint32](shuffle(k0, k1, 4*(iota%2)+iota/2, 8))
			x0 = t.aes.Hash1x4(x0)
			y0 := shuffle(vec[[8]uint64](x0), vec[[8]uint64](x0), 2*iota+0, 4)
			y1 := shuffle(vec[[8]uint64](x0), vec[[8]uint64](x0), 2*iota+1, 4)
			*(*vec[[4]uint64])(unsafe.Add(hp0, o)) = y0
			*(*vec[[4]uint64])(unsafe.Add(hp1, o)) = y1
			n_left -= 4
			o += 4 * 8
		case cpu.HasAesEncryptx1() && n_left >= 1:
			k0 := *(*vec[[2]uint64])(unsafe.Add(kp0, o))
			k1 := *(*vec[[2]uint64])(unsafe.Add(kp1, o))
			x0 := vec[[4]uint32](shuffle(k0, k1, 2*(iota%2)+(iota/2)+0*1))
			x1 := vec[[4]uint32](shuffle(k0, k1, 2*(iota%2)+(iota/2)+1*1))
			x0, x1 = t.aes.Hash2x1(x0, x1)
			y0 := shuffle(vec[[2]uint64](x0), vec[[2]uint64](x1), 2*iota+0)
			y1 := shuffle(vec[[2]uint64](x0), vec[[2]uint64](x1), 2*iota+1)
			*(*vec[[2]uint64])(unsafe.Add(hp0, o)) = y0
			*(*vec[[2]uint64])(unsafe.Add(hp1, o)) = y1
			n_left -= 1
			o += 2 * 8
		default:
			panic("not yet")
		}
	}
	return
}

func (t *ip4_hash_map[V]) set(h *ip4.Header, u uint32, v ip4_hash_map_value) {
	k := h.Key(u)
	hash := vec[[2]uint64](t.aes.Hash(vec[[4]uint32](vec[[2]uint64]{k[0], k[1]})))
	t.h.Set(hash[0], uint128(k[0])+uint128(k[1])<<64, uint64(v))
}
func (t *ip4_hash_map[V]) get(h *ip4.Header, u uint32) (v uint64, ok bool) {
	k := h.Key(u)
	hash := vec[[2]uint64](t.aes.Hash(t.aes.Hash(vec[[4]uint32](vec[[2]uint64]{k[0], k[1]}))))
	v, ok = t.h.Get(hash[0], uint128(k[0])+uint128(k[1])<<64)
	return
}

// +ego { always_inline }
func (t *ip4_hash_map[V]) getv(iph ip4.Headers[V], u vec[[V]uint32]) (ip4_hash_map_values[V], vec[[V]bool]) {
	k := iph.Key(u)
	h := t.hashv([2]vec[[V]uint64](k))
	var p hash.MapPointers[uint128, uint64, V]
	p.Init(&t.h, h[0])
	k128 := vec[[V]uint128](k[0]) + vec[[V]uint128](k[1])<<64
	v, ok := p.Get(h[0], k128)
	return ip4_hash_map_values[V](v), ok
}

const ip4_hash_vec_len = 8

type (
	ip4_hash_map_value               uint64
	ip4_hash_map_values[V ~vec[any]] vec[[V]uint64]
)

type ip4_hash_map_value_config struct {
	Next   int
	Opaque int
}

func (cf ip4_hash_map_value_config) v() ip4_hash_map_value {
	return ip4_hash_map_value(cf.Next) + ip4_hash_map_value(cf.Opaque)<<32
}
func (x ip4_hash_map_value) opaque() uint32             { return uint32(x >> 32) }
func (x ip4_hash_map_value) next() uint32               { return uint32(x) }
func (x ip4_hash_map_values[V]) opaque() vec[[V]uint32] { return vec[[V]uint32](x >> 32) }
func (x ip4_hash_map_values[V]) next() vec[[V]uint32]   { return vec[[V]uint32](x) }

func (m *main) spray_ip4(vm Main, r NodeRuntime, rxf *Frame[struct{}, Ip4BufferIndex]) int {
	const vl = ip4_hash_vec_len

	var sv std.Svec[vec[[vl]any], Ip4BufferIndex]
	sv.Init(unsafe.Pointer(&rxf.V[0]), uint(len(rxf.V)), vm.VecAlignIp4Buffers(vl-1))

	const (
		error_next   = 0
		enable_defer = true
	)
	error_code := r.Error(0)
	fc := &m.ip4_spray_frame_cache
	fc.Begin(vm, r, enable_defer)
	for i, sbi := range sv.Vec() {
		this_ok := sv.Ok(i)
		bi := BufferIndices[vec[[vl]any]](sbi)
		bi4 := Ip4BufferIndices[vec[[vl]any]](sbi)
		h := bi4.Header(vm)
		v, v_ok := m.hash_map.getv(h, 0)
		nexts := vec[[vl]NextIndex[struct{}, Ip4BufferIndex]](ifelse(v_ok, v.next(), error_next))
		fc.Add(vm, r, bi4, nexts, this_ok, enable_defer)

		b := bi.Buffers(vm)

		// Send lookup opaque to next node via buffer opaque.
		b.Opaque1[0] = v.opaque()

		// Opaque gives error codes for next error-drop.
		ifelse(nexts == error_next, b.Error) = ErrorCodes[vec[[vl]any]](ifelse(v_ok, v.opaque(), error_code))
	}
	fc.Put(vm, r, enable_defer)
	return len(rxf.V)
}

func (m *main) drop_ip4(vm Main, r NodeRuntime, rxf *Frame[struct{}, Ip4BufferIndex]) int {
	nv := len(rxf.V)
	ErrorCode(0).Count(vm, r, nv)

	// Prefetch makes it slower...
	if false {
		const vl = 8
		var sv std.Svec[vec[[vl]any], Ip4BufferIndex]
		sv.Init(unsafe.Pointer(&rxf.V[0]), uint(len(rxf.V)), vm.VecAlignIp4Buffers(vl-1))
		for i, sbi := range sv.Vec() {
			bi := BufferIndices[vec[[vl]any]](sbi)
			bi.Buffers(vm).Prefetch()
		}
	}

	vm.BufferFree((*Ip4BufferFrame)(rxf).BufferIndices())
	return nv
}

func (m *main) run(vm Main, in *std.ParseInput) error {
	m.TestConfig = std.DefaultTestConfig()
	m.iter = 0
	for !in.End() {
		switch {
		case in.Parse("%v", &m.TestConfig):
		default:
			return in.ParseErr()
		}
	}
	m.input_node.Enable(vm, true)
	return nil
}

func (m *main) init(vm Main) error {
	m.n_thread = NumThreads()
	m.input_node = vm.NodeWithName("spray-input")
	for i := range m.n_thread {
		m.input_node.AddNextWithThread(vm, "spray-drop", ThreadIndex(i), i)
	}
	m.ip4_spray_node = vm.NodeWithName("spray-ip4")
	m.ip4_spray_node.AddNext(vm, "error-drop", 0)
	for i := range m.n_thread {
		m.ip4_spray_node.AddNextWithThread(vm, "spray-ip4-drop", ThreadIndex(i), i+1)
	}
	m.ip4_spray_node.SetPacketGeneratorEditKind(vm, PacketGeneratorEditKindIp4)
	m.hash_map.h.Alloc(0)
	m.ip4_spray_frame_cache.Init(&FrameCacheConfig{
		DeferPutThreshhold: FrameSize,
	})

	{
		var h struct {
			iph  ip4.Header
			tcph ip.TcpHeader
		}
		h.iph = ip4.Header{
			Version_and_header_length: 0x45,
			Protocol:                  ip.PROTOCOL_TCP,
			Src:                       ip4.Address{1, 2, 3, 4},
			Dst:                       ip4.Address{5, 6, 7, 8},
		}
		h.tcph = ip.TcpHeader{
			PortPair: ip.PortPair{
				Src: ip.HostPort(0xa000).Net(),
				Dst: ip.HostPort(0xdead).Net(),
			},
		}

		t := &m.hash_map
		cf := ip4_hash_map_value_config{Opaque: 17}
		for i := range m.n_thread {
			cf.Next = 1 + i
			t.set(&h.iph, 0, cf.v())
			h.tcph.PortPair.Src = h.tcph.PortPair.Src.Add(1)
		}
	}

	return nil
}

var global_main main

func init() {
	m := &global_main

	m.hash_map.aes.SetRandomSeeds()

	AddInitFunc(&InitFunc{
		Name: "spray",
		Func: m.init,
	})
	AddCliCommand(&CliCommand{
		Path: "spray",
		Func: m.run,
	})
	AddNode(&InputNodeConfig{
		Name:  "spray-input",
		Func:  m.input,
		State: NODE_STATE_DISABLED,
	})
	AddNode(&NodeConfig[s, v]{
		Name:   "spray-drop",
		Func:   m.drop,
		Errors: []string{"drops"},
	})
	AddNode(&NodeConfig[struct{}, Ip4BufferIndex]{
		Name:   "spray-ip4",
		Func:   m.spray_ip4,
		Errors: []string{"lookup miss"},
	})
	AddNode(&NodeConfig[struct{}, Ip4BufferIndex]{
		Name:   "spray-ip4-drop",
		Func:   m.drop_ip4,
		Errors: []string{"lookup hit drops"},
	})
}

var plugin = PluginRegistration{
	Version:           "1.2.3.4",
	Description:       "ego example spray",
	EarlyInitFunction: "vpp.early_init",
	LoadAfter:         "libegovpp",
}
