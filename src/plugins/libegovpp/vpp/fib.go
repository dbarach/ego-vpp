package vpp

// #cgo CFLAGS: -Wno-address-of-packed-member
// #include "vpp.h"
import "C"

import (
	"net/ip4"
	"net/ip6"
	"std"
	"unsafe"
)

type (
	FibIndex    struct{ c C.u32 }
	FibId       uint32
	FibProto    uint8
	FibPathFlag uint32
)

// Encode both protocol and id in single 32 bit number.
const fib_proto_bits = 4

func (f FibId) Get() (p FibProto, i int) {
	p, i = FibProto(f&(1<<fib_proto_bits-1)), int(f>>fib_proto_bits)
	return
}
func (f FibId) Proto() (p FibProto) { p, _ = f.Get(); return }
func MakeFibId(p FibProto, i int) FibId {
	return FibId(p) + FibId(i)<<fib_proto_bits
}

type (
	FibPath struct {
		Proto  FibProto
		Weight uint8
		Table  FibIndex
		SwIntf SwIntf
		Flags  FibPathFlag
	}
	FibPrefix    struct{ c C.fib_prefix_t }
	FibSource    uint8
	FibEntryFlag uint32
)

func (r *FibPrefix) SetIp4(p ip4.Prefix) {
	r.c.fp_proto = C.fib_protocol_t(C.FIB_PROTOCOL_IP4)
	*(*ip4.Address)(unsafe.Pointer(C.ego_fib_prefix_ip4_address(&r.c))) = p.Address
	r.c.fp_len = C.u16(p.Len)
}
func (r *FibPrefix) SetIp6(p ip6.Prefix) {
	r.c.fp_proto = C.fib_protocol_t(C.FIB_PROTOCOL_IP6)
	*(*ip6.Address)(unsafe.Pointer(C.ego_fib_prefix_ip6_address(&r.c))) = p.Address
	r.c.fp_len = C.u16(p.Len)
}
func Ip4FibPrefix(p ip4.Prefix) (x FibPrefix) { x.SetIp4(p); return }
func Ip6FibPrefix(p ip6.Prefix) (x FibPrefix) { x.SetIp6(p); return }

func (p FibProto) CreateTable(id int) {
	C.ip_table_create(C.fib_protocol_t(p), C.u32(id), 0 /* is_api */, 1 /* create_mfib */, 0)
}
func (p FibProto) DeleteTable(id int) {
	C.ip_table_delete(C.fib_protocol_t(p), C.u32(id), 0 /* is_api */)
}
func (id FibId) Bind(s SwIntf) {
	p, i := id.Get()
	C.ip_table_bind(C.fib_protocol_t(p), C.u32(s.c), C.u32(i))
}
func (s SwIntf) BindFib(id FibId) { id.Bind(s) }

func (p FibProto) IndexForId(id int) (i FibIndex, err error) {
	p.CreateTable(id) // make sure it exists
	r := C.fib_table_find(C.fib_protocol_t(p), C.u32(id))
	if ok := r != ^C.u32(0); ok {
		i.c = r
	} else {
		err = std.Errorf("unknown fib table id %#x", id)
	}
	return
}
func (t FibId) Index() (FibIndex, error) {
	p, i := t.Get()
	return p.IndexForId(i)
}
func (t FibId) Del() {
	p, i := t.Get()
	p.DeleteTable(i)
}

func (p *FibPath) c() (c C.fib_route_path_t) {
	c.frp_proto = C.dpo_proto_t(p.Proto)
	c.frp_weight = C.u8(p.Weight)
	C.ego_fib_route_path_set_fib_index(&c, p.Table.c)
	C.ego_fib_route_path_set_sw_if_index(&c, C.u32(p.SwIntf.c))
	c.frp_flags = C.fib_route_path_flags_t(p.Flags)
	return
}

func (p FibIndex) add_del(x *FibPrefix, src FibSource, f FibEntryFlag, path []FibPath, is_del bool) {
	cps := unsafe.Slice(C.ego_fib_route_path_vec_alloc(C.uword(len(path))), len(path))
	for i := range path {
		cps[i] = path[i].c()
	}
	if is_del {
		C.fib_table_entry_path_remove2(p.c, &x.c, C.fib_source_t(src), &cps[0])
	} else {
		C.fib_table_entry_path_add2(p.c, &x.c, C.fib_source_t(src), C.fib_entry_flag_t(f), &cps[0])
	}
	C.ego_fib_route_path_vec_free(cps)
}

type FibAddConfig struct {
	Source     FibSource
	EntryFlags FibEntryFlag
	Prefix     FibPrefix
	Path       []FibPath
}

func (p FibIndex) AddDel(cf *FibAddConfig, is_del bool) {
	f := cf.EntryFlags
	if is_del {
		f = 0
	}
	p.add_del(&cf.Prefix, cf.Source, f, cf.Path, is_del)
}

const (
	FibProtoIp4        = FibProto(C.FIB_PROTOCOL_IP4)
	FibProtoIp6        = FibProto(C.FIB_PROTOCOL_IP6)
	FibProtoMPLS       = FibProto(C.FIB_PROTOCOL_MPLS)
	FibTableIp4Default = FibId(FibProtoIp4)
	FibTableIp6Default = FibId(FibProtoIp6)
)

const (
	FibPathFlagResolveViaHost     = FibPathFlag(C.FIB_ROUTE_PATH_RESOLVE_VIA_HOST)
	FibPathFlagResolveViaAttached = FibPathFlag(C.FIB_ROUTE_PATH_RESOLVE_VIA_ATTACHED)
	FibPathFlagLocal              = FibPathFlag(C.FIB_ROUTE_PATH_LOCAL)
	FibPathFlagAttached           = FibPathFlag(C.FIB_ROUTE_PATH_ATTACHED)
	FibPathFlagDrop               = FibPathFlag(C.FIB_ROUTE_PATH_DROP)
	FibPathFlagExclusive          = FibPathFlag(C.FIB_ROUTE_PATH_EXCLUSIVE)
	FibPathFlagIntfRx             = FibPathFlag(C.FIB_ROUTE_PATH_INTF_RX)
	FibPathFlagRPFId              = FibPathFlag(C.FIB_ROUTE_PATH_RPF_ID)
	FibPathFlagSourceLookup       = FibPathFlag(C.FIB_ROUTE_PATH_SOURCE_LOOKUP)
	FibPathFlagUDPEncap           = FibPathFlag(C.FIB_ROUTE_PATH_UDP_ENCAP)
	FibPathFlagBIERFmask          = FibPathFlag(C.FIB_ROUTE_PATH_BIER_FMASK)
	FibPathFlagBIERTable          = FibPathFlag(C.FIB_ROUTE_PATH_BIER_TABLE)
	FibPathFlagBIERImp            = FibPathFlag(C.FIB_ROUTE_PATH_BIER_IMP)
	FibPathFlagDeaggregate        = FibPathFlag(C.FIB_ROUTE_PATH_DEAG)
	FibPathFlagDVR                = FibPathFlag(C.FIB_ROUTE_PATH_DVR)
	FibPathFlagICMPUnreachable    = FibPathFlag(C.FIB_ROUTE_PATH_ICMP_UNREACH)
	FibPathFlagICMPProhibit       = FibPathFlag(C.FIB_ROUTE_PATH_ICMP_PROHIBIT)
	FibPathFlagClassify           = FibPathFlag(C.FIB_ROUTE_PATH_CLASSIFY)
	FibPathFlagPopPWCW            = FibPathFlag(C.FIB_ROUTE_PATH_POP_PW_CW)
	FibPathFlagGlean              = FibPathFlag(C.FIB_ROUTE_PATH_GLEAN)
)

const (
	FibSourceClassify                  = FibSource(C.FIB_SOURCE_CLASSIFY)
	FibSourceProxy                     = FibSource(C.FIB_SOURCE_PROXY)
	FibSourceInterface                 = FibSource(C.FIB_SOURCE_INTERFACE)
	FibSourceSR                        = FibSource(C.FIB_SOURCE_SR)
	FibSourceBIER                      = FibSource(C.FIB_SOURCE_BIER)
	FibSource6RD                       = FibSource(C.FIB_SOURCE_6RD)
	FibSourceAPI                       = FibSource(C.FIB_SOURCE_API)
	FibSourceCLI                       = FibSource(C.FIB_SOURCE_CLI)
	FibSourceLisp                      = FibSource(C.FIB_SOURCE_LISP)
	FibSourceMap                       = FibSource(C.FIB_SOURCE_MAP)
	FibSourceDhcp                      = FibSource(C.FIB_SOURCE_DHCP)
	FibSourceIp6NeighborDiscoveryProxy = FibSource(C.FIB_SOURCE_IP6_ND_PROXY)
	FibSourceIp6NeighborDiscovery      = FibSource(C.FIB_SOURCE_IP6_ND)
	FibSourceAdjacency                 = FibSource(C.FIB_SOURCE_ADJ)
	FibSourceMPLS                      = FibSource(C.FIB_SOURCE_MPLS)
	FibSourceAE                        = FibSource(C.FIB_SOURCE_AE)
	FibSourceRR                        = FibSource(C.FIB_SOURCE_RR)
	FibSourceURPFExempt                = FibSource(C.FIB_SOURCE_URPF_EXEMPT)
	FibSourceDefaultFib                = FibSource(C.FIB_SOURCE_DEFAULT_ROUTE)
	FibSourceInterpose                 = FibSource(C.FIB_SOURCE_INTERPOSE)
)

const (
	FibEntryFlagNone             = FibEntryFlag(0)
	FibEntryFlagConnected        = FibEntryFlag(C.FIB_ENTRY_FLAG_CONNECTED)
	FibEntryFlagAttached         = FibEntryFlag(C.FIB_ENTRY_FLAG_ATTACHED)
	FibEntryFlagDrop             = FibEntryFlag(C.FIB_ENTRY_FLAG_DROP)
	FibEntryFlagExclusive        = FibEntryFlag(C.FIB_ENTRY_FLAG_EXCLUSIVE)
	FibEntryFlagLocal            = FibEntryFlag(C.FIB_ENTRY_FLAG_LOCAL)
	FibEntryFlagImport           = FibEntryFlag(C.FIB_ENTRY_FLAG_IMPORT)
	FibEntryFlagNoAttachedExport = FibEntryFlag(C.FIB_ENTRY_FLAG_NO_ATTACHED_EXPORT)
	FibEntryFlagLooseURPFExempt  = FibEntryFlag(C.FIB_ENTRY_FLAG_LOOSE_URPF_EXEMPT)
	FibEntryFlagMulticast        = FibEntryFlag(C.FIB_ENTRY_FLAG_MULTICAST)
	FibEntryFlagCoveredInherit   = FibEntryFlag(C.FIB_ENTRY_FLAG_COVERED_INHERIT)
	FibEntryFlagInterpose        = FibEntryFlag(C.FIB_ENTRY_FLAG_INTERPOSE)
)
