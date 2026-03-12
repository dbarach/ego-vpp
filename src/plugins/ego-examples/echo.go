package echo

import (
	"cpu"
	"net/ip"
	"net/ip4"
	"net/ip6"
	"std"
	"sync/atomic"
	"sys"
	"unsafe"
	. "vpp"
)

type (
	echo_main struct {
		vm Main
		App
		std.TestConfig
		nano_times                    [2]uint64
		cpu_times                     [2]uint64
		node_index                    uint32
		generate_incrementing_pattern bool
		validate_incrementing_pattern bool
		disconnect_on_done            bool
		tx_on_connect                 bool
		max_seq                       uint64
		serial                        uint64
		n_connected, n_accepted       uint32
		wm                            AppWorkerMap
		sides                         []echo_side
		side_index_by_ip_addr         map[ip4.Address]uint
		sessions                      []echo_session
		svr_session_handle            SessionHandle
	}
	echo_session struct {
		tx_seq, echo_seq, rx_seq uint64
		nano_times               [2]uint64
		cpu_times                [2]uint64
		handles                  [2]SessionHandle
		s                        [2]*Session
	}
	echo_side struct {
		sock   Socket
		loop   LoopbackIntf
		fib    FibIndex
		fib_id FibId
	}
)

func (es *echo_session) set_incrementing_pattern(b []byte) {
	i := 0
	n_left := len(b)

	{
		const vl = 64
		type V vec[[vl]byte] // +ego { aligned 1 }
		p := V{iota} + byte(es.tx_seq+uint64(i))
		for n_left >= 4*vl {
			*(*V)(unsafe.Pointer(&b[i+0*vl])) = p + 0*vl
			*(*V)(unsafe.Pointer(&b[i+1*vl])) = p + 1*vl
			*(*V)(unsafe.Pointer(&b[i+2*vl])) = p + 2*vl
			*(*V)(unsafe.Pointer(&b[i+3*vl])) = p + 3*vl
			n_left -= 4 * vl
			i += 4 * vl
			p += (4 * vl) % 256
		}
		for n_left >= 1*vl {
			*(*V)(unsafe.Pointer(&b[i+0*vl])) = p + 0*vl
			n_left -= 1 * vl
			i += 1 * vl
			p += (1 * vl) % 256
		}
	}

	{
		const vl = 16
		type V vec[[vl]byte] // +ego { aligned 1 }
		p := V{iota} + byte(es.tx_seq+uint64(i))
		for n_left >= 1*vl {
			*(*V)(unsafe.Pointer(&b[i+0*vl])) = p + 0*vl
			n_left -= 1 * vl
			i += 1 * vl
			p += (1 * vl) % 256
		}
	}

	for n_left > 0 {
		b[i] = byte(es.tx_seq + uint64(i))
		n_left -= 1
		i += 1
	}
}

func (es *echo_session) validate_incrementing_pattern(b []byte) bool {
	i := 0
	n_left := len(b)

	{
		const vl = 64
		type V vec[[vl]byte] // +ego { aligned 1 }
		p := V{iota} + byte(es.rx_seq+uint64(i))
		for n_left >= 4*vl {
			ok0 := *(*V)(unsafe.Pointer(&b[i+0*vl])) == p+0*vl
			ok1 := *(*V)(unsafe.Pointer(&b[i+1*vl])) == p+1*vl
			ok2 := *(*V)(unsafe.Pointer(&b[i+2*vl])) == p+2*vl
			ok3 := *(*V)(unsafe.Pointer(&b[i+3*vl])) == p+3*vl
			if reduce[clz](ok0 || ok1 || ok2 || ok3) != 0 {
				return false
			}
			n_left -= 4 * vl
			i += 4 * vl
			p += (4 * vl) % 256
		}
		for n_left >= 1*vl {
			ok0 := *(*V)(unsafe.Pointer(&b[i+0*vl])) == p+0*vl
			if reduce[clz](ok0) != 0 {
				return false
			}
			n_left -= 1 * vl
			i += 1 * vl
			p += (1 * vl) % 256
		}
	}

	{
		const vl = 16
		type V vec[[vl]byte] // +ego { aligned 1 }
		p := V{iota} + byte(es.rx_seq+uint64(i))
		for n_left >= 1*vl {
			ok0 := *(*V)(unsafe.Pointer(&b[i+0*vl])) == p+0*vl
			if reduce[clz](ok0) != 0 {
				return false
			}
			n_left -= 1 * vl
			i += 1 * vl
			p += (1 * vl) % 256
		}
	}

	for n_left > 0 {
		if b[i] != byte(es.rx_seq+uint64(i)) {
			return false
		}
		n_left -= 1
		i += 1
	}
	return true
}

func (es *echo_session) tx(em *echo_main) (ok bool) {
	s := es.s[Tx]
	if ok = !s.RingFull(); !ok {
		return
	}
	for es.tx_seq < em.max_seq {
		std.BreakpointIf(es.tx_seq == 794624)
		b := s.GetTx()
		cb := uint64(cap(b))
		if cb == 0 {
			break
		}
		b = b[:min(cb, em.max_seq-es.tx_seq)]
		if em.generate_incrementing_pattern {
			es.set_incrementing_pattern(b)
		}
		s.PutTx(b)
		es.tx_seq += uint64(len(b))
		if em.Verbose > 1 {
			std.Logf("tx %d %d/%d %v\n", len(b), es.tx_seq, em.max_seq, s)
		}
	}
	return
}
func (es *echo_session) mark_time(i int) {
	es.nano_times[i] = sys.TimeNano()
	es.cpu_times[i] = cpu.TimeNow()
}
func (es *echo_session) rx_advance(em *echo_main, b []byte) (done bool) {
	if em.validate_incrementing_pattern && !es.validate_incrementing_pattern(b) {
		panic("seq")
	}
	es.rx_seq += uint64(len(b))
	done = es.seq_done(em)
	if done {
		es.mark_time(1)
		if em.disconnect_on_done {
			es.disconnect(em)
		}
	} else {
		es.tx(em)
	}
	return
}

func (es *echo_session) rx(rt RxTx, em *echo_main) {
	s := es.s[rt]
	done := false
	for !done {
		b := s.GetRx()
		if len(b) == 0 {
			break
		}
		if rt == Rx {
			if l := s.PutTx(b); l < len(b) {
				b = b[:l]
			}
			es.echo_seq += uint64(len(b))
		} else {
			done = es.rx_advance(em, b)
		}
		s.PutRx(len(b))

		if em.Verbose > 1 {
			std.Logf("rx %s %d %d/%d %v\n",
				ifelse(rt == Rx, "echo", "advance"),
				len(b), es.rx_seq, em.max_seq, s)
		}

		// Tx full so no more forward progress possible.
		if rt == Rx && len(b) == 0 {
			break
		}
	}
}

func (m *echo_main) s(s *Session) *echo_session { return &m.sessions[s.Opaque] }

const n_session_connects_or_accepts_per_event = 1024

func (s *echo_session) disconnect(m *echo_main) (e SessionError) {
	if m.Verbose > 0 {
		std.Logf("disconnect %v %v\n", s.s[0], s.s[1])
	}
	for _, h := range s.handles {
		if e = m.Disconnect(h); !e.Ok() {
			return
		}
	}
	return
}

func (m *echo_main) SessionConnected(s *ConnectedSession) (err SessionError) {
	m.serial++
	nc := 1 + atomic.Add(&m.n_connected, 1)
	if m.Verbose > 0 {
		std.Logf("connected index %#x, %v %v\n", s.ApiContext, s.Error, s)
	}
	s.Opaque = s.ApiContext
	es := m.s(s.Session)
	es.mark_time(0)
	es.handles[Tx] = s.Handle
	es.s[Tx] = s.Session
	if s.Error == SESSION_ERROR_OK && m.tx_on_connect {
		es.tx(m)
	}
	const n = n_session_connects_or_accepts_per_event
	if nc%n == n/2 || int(nc) >= len(m.sessions) {
		m.vm.SignalEvent(m.node_index, 0, 0)
	}
	return
}
func (m *echo_main) SessionAccepted(s *Session) (err SessionError) {
	m.serial++
	na := 1 + atomic.Add(&m.n_accepted, 1)
	si := m.session_index_for_connected_session(s)
	s.Opaque = uint32(si)
	es := m.s(s)
	es.handles[Rx] = s.Handle
	es.s[Rx] = s
	if m.Verbose > 0 {
		std.Logf("accepted index %#x, %v\n", si, s)
	}
	const n = n_session_connects_or_accepts_per_event
	if na%n == n/2 || int(na) >= len(m.sessions) {
		m.vm.SignalEvent(m.node_index, 0, 1)
	}
	return
}
func (m *echo_main) SessionDisconnected(s *Session) {
	m.serial++
	if m.Verbose > 0 {
		std.Log("disconnected", s)
	}
	// Session is "done" when both handles are nil (e.g. disconnected).
	m.s(s).handles[s.Side()] = SessionHandleNil
}
func (m *echo_main) SessionRx(ss []*Session) (err SessionError) {
	m.serial++
	for _, s := range ss {
		m.s(s).rx(s.Side(), m)
	}
	return
}
func (m *echo_main) SessionTx(ss []*Session) (err SessionError) { return m.SessionRx(ss) }

const port_per_ip = 1<<16 - uint(ip.PORT_MIN_USER)

func (m *echo_main) socket_for_session(i int) (s4 ip4.Socket, s6 ip6.Socket, side int) {
	i0, i1 := uint(i)/port_per_ip, uint(i)%port_per_ip
	p := (ip.PORT_MIN_USER + ip.HostPort(i1)).Net()
	if is_ip4 := m.sides[1].sock.Get(&s4, &s6); is_ip4 {
		s4.Address = s4.Address.Add(uint32(i0))
		s4.Port = p
	} else {
		s6.Address = s6.Address.Add(uint128(i0))
		s6.Port = p
	}
	side = 1 + int(i0)
	return
}
func (m *echo_main) session_index_for_connected_session(s *Session) uint {
	var (
		s4 [2]ip4.Socket
		s6 [2]ip6.Socket
	)
	is_ip4 := m.sides[1].sock.Get(&s4[0], &s6[0])
	s.Socket(Tx).Get(&s4[1], &s6[1])
	if is_ip4 {
		return port_per_ip*uint(s4[1].Address.Host()-s4[0].Address.Host()) + uint(s4[1].Port.Host()-s4[0].Port.Host())
	} else {
		return port_per_ip*uint(s6[1].Address.Host()-s6[0].Address.Host()) + uint(s6[1].Port.Host()-s6[0].Port.Host())
	}
}

func (s *echo_session) seq_done(m *echo_main) bool { return s.rx_seq >= m.max_seq }
func (m *echo_main) seq_done() bool {
	for _, s := range &m.sessions {
		if !s.seq_done(m) {
			return false
		}
	}
	return true
}
func (m *echo_main) disconnect_done() bool {
	for _, s := range &m.sessions {
		for _, h := range s.handles {
			if h != SessionHandleNil {
				return false
			}
		}
	}
	return true
}

func (m *echo_main) fib_add_del(is_del bool) {
	add := &FibAddConfig{
		Source: FibSourceCLI,
		Path: []FibPath{
			FibPath{
				Proto:  FibProtoIp4,
				Weight: 1,
				SwIntf: SwIntfNil,
				Flags:  FibPathFlagDeaggregate,
			},
		},
	}
	for i := range len(m.sides) - 1 {
		for j := range 2 {
			s0, s1 := &m.sides[0], &m.sides[1+i]
			if j == 1 {
				s0, s1 = s1, s0
			}
			add.Path[0].Table = s1.fib
			var (
				s4 ip4.Socket
				s6 ip6.Socket
			)
			if s1.sock.Get(&s4, &s6) {
				add.Prefix = Ip4FibPrefix(ip4.Prefix{Address: s4.Address, Len: 32})
			} else {
				add.Prefix = Ip6FibPrefix(ip6.Prefix{Address: s6.Address, Len: 128})
			}
			s0.fib.AddDel(add, is_del)
		}
	}
}

func (m *echo_main) free() {
	free(m.sides)
	free(m.sessions)
	free(m)
}

func (m *echo_main) wait(dt float64) {
	if m.vm.WaitForEvent(dt) > 0 {
		kind, data := m.vm.ReceiveEvent()
		if m.Verbose > 0 {
			std.Log("event", kind, data)
		}
		free(data)
	}
}

func main(vm Main, in *std.ParseInput) error {
	m := save(&echo_main{vm: vm})

	m.node_index = vm.CurrentNodeIndex()

	src := ip4.Socket{
		Address: ip4.Address{1, 0, 0, 0},
		Port:    ip.PORT_MIN_USER.Net(),
	}
	dst := ip4.Socket{
		Address: ip4.Address{2, 0, 0, 0},
		Port:    (ip.PORT_MIN_USER - 1).Net(),
	}
	m.TestConfig = std.DefaultTestConfig()
	app_config := &AppConfig{
		Name:            "ego echo",
		GlobalScope:     true,
		LocalScope:      true,
		IsBuiltin:       true,
		UseHugePages:    true,
		MemfdForBuiltin: false, // causes constipation?
		// EventMessageQueueUseEventFd: true, needs socket api
		FirstSegmentSize:      1 << 29,
		AdditionalSegmentSize: 1 << 29,
		RxFifoSize:            1 << 12,
		TxFifoSize:            1 << 12,
		MaxFifoSize:           256 << 10,
	}
	m.max_seq = 10000
	log2_chunk_size := 12
	for !in.End() {
		var (
			f float64
			i int
		)
		switch {
		case in.Parse("%v", &m.TestConfig):
		case in.Parse("max %f", &f):
			m.max_seq = uint64(f)
		case in.Parse("log2-segment-size %d", &i):
			n := uint64(1) << i
			app_config.FirstSegmentSize = n
			app_config.AdditionalSegmentSize = n
		case in.Parse("log2-fifo-size %d", &i):
			n := uint64(1) << i
			app_config.RxFifoSize = n
			app_config.TxFifoSize = n
		case in.Parse("log2-chunk-size %d", &i):
			log2_chunk_size = i
		default:
			return in.ParseErr()
		}
	}
	m.generate_incrementing_pattern = m.Validate > 0
	m.validate_incrementing_pattern = m.Validate > 1
	app_config.Log2ChunkSize = uint64(log2_chunk_size)

	{
		s0, s1 := uint(m.Size)/port_per_ip, uint(m.Size)%port_per_ip
		n_side := 1 + s0 + ifelse(s1 != 0)
		m.sides = make([]echo_side, n_side)
	}
	m.sessions = make([]echo_session, m.Size)

	svr, cli := &m.sides[0], &m.sides[1]
	for i, si := range &m.sides {
		if i == 0 {
			si.sock.Set(&dst, nil)
		} else {
			t := src
			t.Address = t.Address.Add(uint32(i - 1))
			si.sock.Set(&t, nil)
		}
	}
	std.Logf("echo %v %v -> %v\n", &m.TestConfig, &cli.sock, &svr.sock)

	for i, si := range &m.sides {
		si.fib_id = MakeFibId(FibProtoIp4, i+1)
		if x, err := si.fib_id.Index(); err != nil {
			return err
		} else {
			si.fib = x
		}
		if l, e := NewLoopbackIntf(&LoopbackConfig{}); e != nil {
			return e
		} else {
			si.loop = l
		}
		si.loop.BindFib(si.fib_id)

		var (
			s4 ip4.Socket
			s6 ip6.Socket
		)
		if is_ip4 := si.sock.Get(&s4, &s6); is_ip4 {
			if e := si.loop.AddIp4(ip4.Prefix{Address: s4.Address, Len: 32}); e != nil {
				return e
			}
		} else {
			if e := si.loop.AddIp6(ip6.Prefix{Address: s6.Address, Len: 128}); e != nil {
				return e
			}
		}
		si.loop.Up()
	}

	// Add routes for 2.2.2.2/32 to lookup in fib 1; 1.1.1.1/32 to lookup in fib 0
	// Without this RPF check turfs packets.
	m.fib_add_del(false)

	const (
		fifo_size  = 4 << 10
		chunk_size = 4 << 10
	)
	if e := m.Attach(m, app_config); !e.Ok() {
		return e
	}

	// Listen on port 0.0.0.0:22222.
	{
		var (
			c  SessionEndpointConfig
			s4 ip4.Socket
			s6 ip6.Socket
		)
		if svr.sock.Get(&s4, &s6) {
			c.SetIp4(ip4.Socket{Port: s4.Port})
		} else {
			c.SetIp6(ip6.Socket{Port: s6.Port})
		}
		c.SetFib(svr.fib)
		if h, e := m.Listen(&c, m.wm); e.Ok() {
			m.svr_session_handle = h
		} else {
			return e
		}
	}

	// Connect to now listening server.
	for i, s := range &m.sessions {
		var (
			c          SessionEndpointConfig
			s4         [2]ip4.Socket
			s6         [2]ip6.Socket
			side_index int
		)
		s4[1], s6[1], side_index = m.socket_for_session(i)
		if svr.sock.Get(&s4[0], &s6[0]) {
			c.SetIp4(s4[0])
			c.Peer().SetIp4(s4[1])
		} else {
			c.SetIp6(s6[0])
			c.Peer().SetIp6(s6[1])
		}
		c.SetFib(m.sides[side_index].fib)
		if h, e := m.Connect(&c, &ConnectConfig{ApiContext: uint32(i), WorkerMap: m.wm}); e.Ok() {
			// Save handle for half-open connection; replaced by open connection handle in connect callback.
			m.sessions[i].handles[Tx] = h
		} else {
			return e
		}
		if i > 0 && i%n_session_connects_or_accepts_per_event == 0 {
			m.wait(1)
		}
	}
	if !m.tx_on_connect {
		for {
			m.wait(1)
			if int(m.n_connected) >= len(m.sessions) && int(m.n_accepted) >= len(m.sessions) {
				break
			}
		}

		m.cpu_times[0] = cpu.TimeNow()
		m.nano_times[0] = sys.TimeNano()

		for i, s := range &m.sessions {
			for !s.tx(m) {
				m.vm.Suspend(100e-3)
			}
		}
	} else {
		m.cpu_times[0] = cpu.TimeNow()
		m.nano_times[0] = sys.TimeNano()
	}

	timed_out := false
	const (
		suspend_secs     = 250e-3
		timeout_secs     = 5
		print_every_secs = 2
	)
	last_serial := m.serial
	n_iter_since_last_print := 0
	n_last_serial := 0
	bytes_done_last_print := uint64(0)
	for !m.seq_done() {
		m.vm.Suspend(suspend_secs)
		if m.serial == last_serial {
			n_last_serial++
		}
		last_serial = m.serial
		ns := len(m.sessions)
		timed_out = float64(n_last_serial)*suspend_secs >= timeout_secs && m.n_connected == uint32(ns) && m.n_accepted == uint32(ns)
		if timed_out {
			std.Log("timed out connections:")
			n_to := 0
			max_n_print := 10
			for i, s := range &m.sessions {
				if !s.seq_done(m) {
					n_to++
					if max_n_print > 0 {
						std.Logf("  0x%06x %v %d -> %d -> %d max %d\n", i, s.s[Tx], s.tx_seq, s.echo_seq, s.rx_seq, m.max_seq)
						max_n_print--
					}
				}
			}
			std.Logf("  %d timed out connections (fraction %.4e)\n", n_to, float64(n_to)/float64(ns))
			break
		}
		n_iter_since_last_print++
		if float64(n_iter_since_last_print)*suspend_secs >= print_every_secs {
			n_iter_since_last_print = 0
			rx_total := uint64(0)
			for _, s := range &m.sessions {
				rx_total += s.rx_seq
			}
			rt := float64(rx_total)
			tt := float64(uint64(len(m.sessions)) * m.max_seq)
			std.Logf("%.6e bytes sent; %.1f%% done", rt, 100*rt/tt)
			if l := len(m.sessions); m.n_connected < uint32(l) || m.n_accepted < uint32(l) {
				std.Logf(" %.2f%% connected %.2f%% accepted",
					100*float64(m.n_connected)/float64(l),
					100*float64(m.n_accepted)/float64(l))
			}
			if bytes_done_last_print != 0 {
				db := 2 * (rx_total - bytes_done_last_print) // 2 since we're both sender and receiver
				bps := 8 * float64(db) / float64(print_every_secs)
				std.Logf(" %.6e bits/sec", bps)
			}
			std.Logf("\n")
			bytes_done_last_print = rx_total
		}
	}
	if !timed_out {
		m.nano_times[1] = sys.TimeNano()
		m.cpu_times[1] = cpu.TimeNow()
		bytes_total := float64(2 * uint64(m.Size) * m.max_seq)

		dt_secs := 1e-9 * float64(m.nano_times[1]-m.nano_times[0])
		dt_cycles := float64(m.cpu_times[1] - m.cpu_times[0])
		std.Logf("done %.0e bytes sent in %.4e secs, %.6e bits/sec %.6e bytes/cycle\n",
			bytes_total, dt_secs,
			float64(8*bytes_total/dt_secs),
			float64(bytes_total/dt_cycles),
		)

		dt_secs = 0
		dt_cycles = 0
		for _, s := range &m.sessions {
			dt_secs += 1e-9 * float64(s.nano_times[1]-s.nano_times[0])
			dt_cycles += float64(s.cpu_times[1] - s.cpu_times[0])
		}
		ns := float64(len(m.sessions))
		dt_secs /= ns
		dt_cycles /= ns
		std.Logf("  %.4e secs, %.6e bits/sec %.6e bytes/cycle\n",
			dt_secs,
			float64(8*bytes_total/dt_secs),
			float64(bytes_total/dt_cycles),
		)
	}

	if e := m.Unlisten(m.svr_session_handle, m.wm); !e.Ok() {
		return e
	}

	if !m.disconnect_on_done {
		for _, s := range &m.sessions {
			s.disconnect(m)
		}
	}

	for {
		m.vm.Suspend(suspend_secs)
		if m.disconnect_done() {
			break
		}
	}

	if e := m.Detach(); !e.Ok() {
		return e
	}
	m.fib_add_del(true)
	for i := range m.sides {
		m.sides[i].loop.Del()
		m.sides[i].fib_id.Del()
	}
	m.free()
	return nil
}

func init() {
	AddCliCommand(&CliCommand{
		Path: "ego-echo",
		Func: main,
	})
}

var plugin = PluginRegistration{
	Version:           "1.2.3.4",
	Description:       "ego echo application",
	EarlyInitFunction: "vpp.early_init",
	LoadAfter:         "libegovpp",
}
