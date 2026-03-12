package vpp

// #cgo CFLAGS: -Wno-address-of-packed-member
// #include "vpp.h"
import "C"

import (
	"net"
	"net/ethernet"
	"net/ip4"
	"net/ip6"
	"std"
	"unsafe"
)

type (
	intf   struct{ c C.u32 }
	SwIntf struct{ intf }
	HwIntf struct{ intf }
)

var SwIntfNil = SwIntf{intf: intf{c: ^C.u32(0)}}

func (s SwIntf) set_get_flags(new SwIntfFlags, is_set bool) (old SwIntfFlags) {
	vm := C.vnet_get_main()
	old = SwIntfFlags(C.ego_vnet_sw_interface_get_flags(vm, s.c))
	if is_set {
		C.vnet_sw_interface_set_flags(vm, C.u32(s.c), C.vnet_sw_interface_flags_t(new))
	}
	return
}
func (s SwIntf) SetFlags(new SwIntfFlags) SwIntfFlags { return s.set_get_flags(new, true) }
func (s SwIntf) GetFlags() SwIntfFlags                { return s.set_get_flags(0, false) }

func (s SwIntf) toggle(new bool, x SwIntfFlags) (old bool) {
	f := s.GetFlags()
	if old = f&x != 0; old != new {
		if new {
			f |= x
		} else {
			f &^= x
		}
		s.SetFlags(f)
	}
	return
}
func (s SwIntf) UpDown(v bool) bool { return s.toggle(v, SwIntfAdminUp) }
func (s SwIntf) Up() bool           { return s.UpDown(true) }
func (s SwIntf) Down() bool         { return s.UpDown(false) }
func (s SwIntf) SetMtu(new uint) (old uint) {
	vm := C.vnet_get_main()
	si := C.ego_vnet_get_sw_interface(vm, s.c)
	old = uint(si.mtu[C.VNET_MTU_L3])
	C.vnet_sw_interface_set_mtu(vm, s.c, C.u32(new))
	return
}

type (
	LoopbackIntf   struct{ SwIntf }
	LoopbackConfig struct {
		Ethernet      ethernet.Address
		Instance      uint
		InstanceValid bool
	}
)

func NewLoopbackIntf(cf *LoopbackConfig) (l LoopbackIntf, err error) {
	if e := VnetError(C.vnet_create_loopback_interface(&l.c,
		(*C.u8)(&cf.Ethernet[0]),
		ifelse(cf.InstanceValid), C.u32(cf.Instance))); e != VNET_ERROR_OK {
		err = std.Errorf("%v", e)
	}
	return
}
func (l *LoopbackIntf) Del() {
	C.vnet_delete_loopback_interface(C.u32(l.c))
	l.SwIntf = SwIntfNil
}

type (
	ip4_address struct{ c C.ip4_address_t }
	ip6_address struct{ c C.ip6_address_t }
)

func (a *ip4_address) set(b ip4.Address)            { *(*net.Uint32)(unsafe.Pointer(a)) = b.Net() }
func (a *ip6_address) set(b ip6.Address)            { *(*net.Uint128)(unsafe.Pointer(a)) = b.Net() }
func new_ip4_address(b ip4.Address) (a ip4_address) { a.set(b); return }
func new_ip6_address(b ip6.Address) (a ip6_address) { a.set(b); return }

func (i SwIntf) AddDelIp4(p ip4.Prefix, is_del bool) error {
	vm := C.ego_vlib_get_main()
	b := new_ip4_address(p.Address)
	return clib_go_error(unsafe.Pointer(C.ip4_add_del_interface_address(vm, C.u32(i.c), &b.c, C.u32(p.Len), ifelse(is_del))))
}
func (i SwIntf) AddDelIp6(p ip6.Prefix, is_del bool) error {
	vm := C.ego_vlib_get_main()
	b := new_ip6_address(p.Address)
	return clib_go_error(unsafe.Pointer(C.ip6_add_del_interface_address(vm, C.u32(i.c), &b.c, C.u32(p.Len), ifelse(is_del))))
}
func (i SwIntf) AddIp4(p ip4.Prefix) error { return i.AddDelIp4(p, false) }
func (i SwIntf) DelIp4(p ip4.Prefix) error { return i.AddDelIp4(p, true) }
func (i SwIntf) AddIp6(p ip6.Prefix) error { return i.AddDelIp6(p, false) }
func (i SwIntf) DelIp6(p ip6.Prefix) error { return i.AddDelIp6(p, true) }
