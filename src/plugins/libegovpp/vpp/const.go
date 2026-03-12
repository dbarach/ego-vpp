package vpp

// #cgo CFLAGS: -Wno-address-of-packed-member
// #include "vpp.h"
import "C"

import "std"

const (
	CRYPTO_ENGINE_NONE    = CryptoEngine(C.CRYPTO_ENGINE_NONE)
	CRYPTO_ENGINE_OPENSSL = CryptoEngine(C.CRYPTO_ENGINE_OPENSSL)
	CRYPTO_ENGINE_PICOTLS = CryptoEngine(C.CRYPTO_ENGINE_PICOTLS)
	CRYPTO_ENGINE_VPP     = CryptoEngine(C.CRYPTO_ENGINE_VPP)
	CRYPTO_ENGINE_MBEDTLS = CryptoEngine(C.CRYPTO_ENGINE_MBEDTLS)
)

func (e SessionError) Ok() bool      { return e == SESSION_ERROR_OK }
func (e SessionError) Error() string { return session_error_strings[-e] }
func (e SessionError) Must() {
	if !e.Ok() {
		panic(error(e))
	}
}

const (
	SESSION_ERROR_OK                 = SessionError(0)
	SESSION_ERROR_UNKNOWN            = SessionError(C.SESSION_E_UNKNOWN)
	SESSION_ERROR_REFUSED            = SessionError(C.SESSION_E_REFUSED)
	SESSION_ERROR_TIME_OUT           = SessionError(C.SESSION_E_TIMEDOUT)
	SESSION_ERROR_ALLOC              = SessionError(C.SESSION_E_ALLOC)
	SESSION_ERROR_OWNER              = SessionError(C.SESSION_E_OWNER)
	SESSION_ERROR_NO_ROUTE           = SessionError(C.SESSION_E_NOROUTE)
	SESSION_ERROR_NO_INTERFACE       = SessionError(C.SESSION_E_NOINTF)
	SESSION_ERROR_NO_IP_ADDRESS      = SessionError(C.SESSION_E_NOIP)
	SESSION_ERROR_NO_PORT            = SessionError(C.SESSION_E_NOPORT)
	SESSION_ERROR_NO_SUPPORT         = SessionError(C.SESSION_E_NOSUPPORT)
	SESSION_ERROR_NO_LISTEN          = SessionError(C.SESSION_E_NOLISTEN)
	SESSION_ERROR_NO_SESSION         = SessionError(C.SESSION_E_NOSESSION)
	SESSION_ERROR_NO_APP             = SessionError(C.SESSION_E_NOAPP)
	SESSION_ERROR_APP_ATTACHED       = SessionError(C.SESSION_E_APP_ATTACHED)
	SESSION_ERROR_PORT_IN_USE        = SessionError(C.SESSION_E_PORTINUSE)
	SESSION_ERROR_IP_IN_USE          = SessionError(C.SESSION_E_IPINUSE)
	SESSION_ERROR_ALREADY_LISTENING  = SessionError(C.SESSION_E_ALREADY_LISTENING)
	SESSION_ERROR_ADDR_NOT_IN_USE    = SessionError(C.SESSION_E_ADDR_NOT_IN_USE)
	SESSION_ERROR_INVALID            = SessionError(C.SESSION_E_INVALID)
	SESSION_ERROR_INVALID_REMOTE_IP  = SessionError(C.SESSION_E_INVALID_RMT_IP)
	SESSION_ERROR_INVALID_APP_WORKER = SessionError(C.SESSION_E_INVALID_APPWRK)
	SESSION_ERROR_INVALID_NAMESPACE  = SessionError(C.SESSION_E_INVALID_NS)
	SESSION_ERROR_SEG_NO_SPACE       = SessionError(C.SESSION_E_SEG_NO_SPACE)
	SESSION_ERROR_SEG_NO_SPACE2      = SessionError(C.SESSION_E_SEG_NO_SPACE2)
	SESSION_ERROR_SEG_CREATE         = SessionError(C.SESSION_E_SEG_CREATE)
	SESSION_ERROR_FILTERED           = SessionError(C.SESSION_E_FILTERED)
	SESSION_ERROR_SCOPE              = SessionError(C.SESSION_E_SCOPE)
	SESSION_ERROR_BAPI_NO_FD         = SessionError(C.SESSION_E_BAPI_NO_FD)
	SESSION_ERROR_BAPI_SEND_FD       = SessionError(C.SESSION_E_BAPI_SEND_FD)
	SESSION_ERROR_BAPI_NO_REG        = SessionError(C.SESSION_E_BAPI_NO_REG)
	SESSION_ERROR_MQ_MSG_ALLOC       = SessionError(C.SESSION_E_MQ_MSG_ALLOC)
	SESSION_ERROR_TLS_HANDSHAKE      = SessionError(C.SESSION_E_TLS_HANDSHAKE)
	SESSION_ERROR_EVENTFD_ALLOC      = SessionError(C.SESSION_E_EVENTFD_ALLOC)
	SESSION_ERROR_NOEXTCFG           = SessionError(C.SESSION_E_NOEXTCFG)
	SESSION_ERROR_NOCRYPTOENG        = SessionError(C.SESSION_E_NOCRYPTOENG)
	SESSION_ERROR_NOCRYPTOCKP        = SessionError(C.SESSION_E_NOCRYPTOCKP)
	SESSION_ERROR_LOCAL_CONNECT      = SessionError(C.SESSION_E_LOCAL_CONNECT)
	SESSION_ERROR_WRONG_NS_SECRET    = SessionError(C.SESSION_E_WRONG_NS_SECRET)
	SESSION_ERROR_SYSCALL            = SessionError(C.SESSION_E_SYSCALL)
	SESSION_ERROR_TRANSPORT_NO_REG   = SessionError(C.SESSION_E_TRANSPORT_NO_REG)
	SESSION_ERROR_MAX_STREAMS_HIT    = SessionError(C.SESSION_E_MAX_STREAMS_HIT)
)

var session_error_strings = [...]string{
	-SESSION_ERROR_OK:                 "OK",
	-SESSION_ERROR_UNKNOWN:            "Generic/unknown error",
	-SESSION_ERROR_REFUSED:            "Refused",
	-SESSION_ERROR_TIME_OUT:           "Time out",
	-SESSION_ERROR_ALLOC:              "object/memory allocation error",
	-SESSION_ERROR_OWNER:              "object not owned by application",
	-SESSION_ERROR_NO_ROUTE:           "no route",
	-SESSION_ERROR_NO_INTERFACE:       "no resolving interface",
	-SESSION_ERROR_NO_IP_ADDRESS:      "no ip address for local interface",
	-SESSION_ERROR_NO_PORT:            "no local port",
	-SESSION_ERROR_NO_SUPPORT:         "not supported",
	-SESSION_ERROR_NO_LISTEN:          "not listening",
	-SESSION_ERROR_NO_SESSION:         "session does not exist",
	-SESSION_ERROR_NO_APP:             "application not attached",
	-SESSION_ERROR_APP_ATTACHED:       "application already attached",
	-SESSION_ERROR_PORT_IN_USE:        "local port in use",
	-SESSION_ERROR_IP_IN_USE:          "ip address in use",
	-SESSION_ERROR_ALREADY_LISTENING:  "ip port pair already listened on",
	-SESSION_ERROR_ADDR_NOT_IN_USE:    "address not in use",
	-SESSION_ERROR_INVALID:            "invalid value",
	-SESSION_ERROR_INVALID_REMOTE_IP:  "invalid remote ip address",
	-SESSION_ERROR_INVALID_APP_WORKER: "invalid app worker",
	-SESSION_ERROR_INVALID_NAMESPACE:  "invalid namespace",
	-SESSION_ERROR_SEG_NO_SPACE:       "failed to allocate a fifo pair",
	-SESSION_ERROR_SEG_NO_SPACE2:      "created segment but failed to allocate a fifo pair",
	-SESSION_ERROR_SEG_CREATE:         "failed to create a new segment",
	-SESSION_ERROR_FILTERED:           "session filtered",
	-SESSION_ERROR_SCOPE:              "scope not supported",
	-SESSION_ERROR_BAPI_NO_FD:         "binary api has no socket fd",
	-SESSION_ERROR_BAPI_SEND_FD:       "binary api send failed over socket fd",
	-SESSION_ERROR_BAPI_NO_REG:        "app binary api registration not found",
	-SESSION_ERROR_MQ_MSG_ALLOC:       "failed to alloc mq msg",
	-SESSION_ERROR_TLS_HANDSHAKE:      "failed TLS handshake",
	-SESSION_ERROR_EVENTFD_ALLOC:      "failed to alloc eventfd",
	-SESSION_ERROR_NOEXTCFG:           "no extended transport config",
	-SESSION_ERROR_NOCRYPTOENG:        "no crypto engine",
	-SESSION_ERROR_NOCRYPTOCKP:        "cert key pair not found ",
	-SESSION_ERROR_LOCAL_CONNECT:      "could not connect with local scope",
	-SESSION_ERROR_WRONG_NS_SECRET:    "wrong ns secret",
	-SESSION_ERROR_SYSCALL:            "system call error",
	-SESSION_ERROR_TRANSPORT_NO_REG:   "transport was not registered",
	-SESSION_ERROR_MAX_STREAMS_HIT:    "max streams hit",
}

func (e SessionError) Format(b *std.Buffer) {
	n := -int(e)
	if n < len(session_error_strings) {
		b.AddString(session_error_strings[n])
	} else {
		b.Format("unknown %d", n)
	}
}

const (
	SessionStateCreated          = SessionState(C.SESSION_STATE_CREATED)
	SessionStateListening        = SessionState(C.SESSION_STATE_LISTENING)
	SessionStateConnecting       = SessionState(C.SESSION_STATE_CONNECTING)
	SessionStateAccepting        = SessionState(C.SESSION_STATE_ACCEPTING)
	SessionStateReady            = SessionState(C.SESSION_STATE_READY)
	SessionStateOpened           = SessionState(C.SESSION_STATE_OPENED)
	SessionStateTransportClosing = SessionState(C.SESSION_STATE_TRANSPORT_CLOSING)
	SessionStateClosing          = SessionState(C.SESSION_STATE_CLOSING)
	SessionStateApp_closed       = SessionState(C.SESSION_STATE_APP_CLOSED)
	SessionStateTransportClosed  = SessionState(C.SESSION_STATE_TRANSPORT_CLOSED)
	SessionStateClosed           = SessionState(C.SESSION_STATE_CLOSED)
	SessionStateTransportDeleted = SessionState(C.SESSION_STATE_TRANSPORT_DELETED)
)

const (
	SESSION_ENDPOINT_PROXY_LISTEN = SessionEndpointConfigFlags(C.SESSION_ENDPT_CFG_F_PROXY_LISTEN)
	SESSION_ENDPOINT_SECURE       = SessionEndpointConfigFlags(C.SESSION_ENDPT_CFG_F_SECURE)
)

const (
	TransportProtoTCP        = TransportProto(C.TRANSPORT_PROTO_TCP)
	TransportProtoUDP        = TransportProto(C.TRANSPORT_PROTO_UDP)
	TransportProtoCutThrough = TransportProto(C.TRANSPORT_PROTO_CT)
	TransportProtoTLS        = TransportProto(C.TRANSPORT_PROTO_TLS)
	TransportProtoQUIC       = TransportProto(C.TRANSPORT_PROTO_QUIC)
	TransportProtoDTLS       = TransportProto(C.TRANSPORT_PROTO_DTLS)
	TransportProtoSRTP       = TransportProto(C.TRANSPORT_PROTO_SRTP)
	TransportProtoHTTP       = TransportProto(C.TRANSPORT_PROTO_HTTP)
)

var transport_proto_strings = [...]string{
	TransportProtoTCP:        "tcp",
	TransportProtoUDP:        "udp",
	TransportProtoCutThrough: "ct",
	TransportProtoTLS:        "tls",
	TransportProtoQUIC:       "quic",
	TransportProtoDTLS:       "dtls",
	TransportProtoSRTP:       "srtp",
	TransportProtoHTTP:       "http",
}

var transport_proto_string_map = func() (m std.StringMap) {
	for t, s := range transport_proto_strings {
		m[s] = uint(t)
	}
	return
}()

func (t TransportProto) Format(b *std.Buffer) { b.Stringer(transport_proto_strings[:], int(t)) }
func (t *TransportProto) Parse(in *std.ParseInput) {
	var v uint
	switch {
	case in.Parse("%v", &transport_proto_string_map, nil, &v):
	default:
		in.ParseError()
	}
	*t = TransportProto(v)
}

const (
	VNET_ERROR_OK                                 = VnetError(0)
	VNET_ERROR_UNSPECIFIED                        = VnetError(C.VNET_ERR_UNSPECIFIED)
	VNET_ERROR_INVALID_SW_IF_INDEX                = VnetError(C.VNET_ERR_INVALID_SW_IF_INDEX)
	VNET_ERROR_NO_SUCH_FIB                        = VnetError(C.VNET_ERR_NO_SUCH_FIB)
	VNET_ERROR_NO_SUCH_INNER_FIB                  = VnetError(C.VNET_ERR_NO_SUCH_INNER_FIB)
	VNET_ERROR_NO_SUCH_LABEL                      = VnetError(C.VNET_ERR_NO_SUCH_LABEL)
	VNET_ERROR_NO_SUCH_ENTRY                      = VnetError(C.VNET_ERR_NO_SUCH_ENTRY)
	VNET_ERROR_INVALID_VALUE                      = VnetError(C.VNET_ERR_INVALID_VALUE)
	VNET_ERROR_INVALID_VALUE_2                    = VnetError(C.VNET_ERR_INVALID_VALUE_2)
	VNET_ERROR_UNIMPLEMENTED                      = VnetError(C.VNET_ERR_UNIMPLEMENTED)
	VNET_ERROR_INVALID_SW_IF_INDEX_2              = VnetError(C.VNET_ERR_INVALID_SW_IF_INDEX_2)
	VNET_ERROR_SYSCALL_ERROR_1                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_1)
	VNET_ERROR_SYSCALL_ERROR_2                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_2)
	VNET_ERROR_SYSCALL_ERROR_3                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_3)
	VNET_ERROR_SYSCALL_ERROR_4                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_4)
	VNET_ERROR_SYSCALL_ERROR_5                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_5)
	VNET_ERROR_SYSCALL_ERROR_6                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_6)
	VNET_ERROR_SYSCALL_ERROR_7                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_7)
	VNET_ERROR_SYSCALL_ERROR_8                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_8)
	VNET_ERROR_SYSCALL_ERROR_9                    = VnetError(C.VNET_ERR_SYSCALL_ERROR_9)
	VNET_ERROR_SYSCALL_ERROR_10                   = VnetError(C.VNET_ERR_SYSCALL_ERROR_10)
	VNET_ERROR_FEATURE_DISABLED                   = VnetError(C.VNET_ERR_FEATURE_DISABLED)
	VNET_ERROR_INVALID_REGISTRATION               = VnetError(C.VNET_ERR_INVALID_REGISTRATION)
	VNET_ERROR_NEXT_HOP_NOT_IN_FIB                = VnetError(C.VNET_ERR_NEXT_HOP_NOT_IN_FIB)
	VNET_ERROR_UNKNOWN_DESTINATION                = VnetError(C.VNET_ERR_UNKNOWN_DESTINATION)
	VNET_ERROR_NO_PATHS_IN_ROUTE                  = VnetError(C.VNET_ERR_NO_PATHS_IN_ROUTE)
	VNET_ERROR_NEXT_HOP_NOT_FOUND_MP              = VnetError(C.VNET_ERR_NEXT_HOP_NOT_FOUND_MP)
	VNET_ERROR_NO_MATCHING_INTERFACE              = VnetError(C.VNET_ERR_NO_MATCHING_INTERFACE)
	VNET_ERROR_INVALID_VLAN                       = VnetError(C.VNET_ERR_INVALID_VLAN)
	VNET_ERROR_VLAN_ALREADY_EXISTS                = VnetError(C.VNET_ERR_VLAN_ALREADY_EXISTS)
	VNET_ERROR_INVALID_SRC_ADDRESS                = VnetError(C.VNET_ERR_INVALID_SRC_ADDRESS)
	VNET_ERROR_INVALID_DST_ADDRESS                = VnetError(C.VNET_ERR_INVALID_DST_ADDRESS)
	VNET_ERROR_ADDRESS_LENGTH_MISMATCH            = VnetError(C.VNET_ERR_ADDRESS_LENGTH_MISMATCH)
	VNET_ERROR_ADDRESS_NOT_FOUND_FOR_INTERFACE    = VnetError(C.VNET_ERR_ADDRESS_NOT_FOUND_FOR_INTERFACE)
	VNET_ERROR_ADDRESS_NOT_DELETABLE              = VnetError(C.VNET_ERR_ADDRESS_NOT_DELETABLE)
	VNET_ERROR_IP6_NOT_ENABLED                    = VnetError(C.VNET_ERR_IP6_NOT_ENABLED)
	VNET_ERROR_NO_SUCH_NODE                       = VnetError(C.VNET_ERR_NO_SUCH_NODE)
	VNET_ERROR_NO_SUCH_NODE2                      = VnetError(C.VNET_ERR_NO_SUCH_NODE2)
	VNET_ERROR_NO_SUCH_TABLE                      = VnetError(C.VNET_ERR_NO_SUCH_TABLE)
	VNET_ERROR_NO_SUCH_TABLE2                     = VnetError(C.VNET_ERR_NO_SUCH_TABLE2)
	VNET_ERROR_NO_SUCH_TABLE3                     = VnetError(C.VNET_ERR_NO_SUCH_TABLE3)
	VNET_ERROR_SUBIF_ALREADY_EXISTS               = VnetError(C.VNET_ERR_SUBIF_ALREADY_EXISTS)
	VNET_ERROR_SUBIF_CREATE_FAILED                = VnetError(C.VNET_ERR_SUBIF_CREATE_FAILED)
	VNET_ERROR_INVALID_MEMORY_SIZE                = VnetError(C.VNET_ERR_INVALID_MEMORY_SIZE)
	VNET_ERROR_INVALID_INTERFACE                  = VnetError(C.VNET_ERR_INVALID_INTERFACE)
	VNET_ERROR_INVALID_VLAN_TAG_COUNT             = VnetError(C.VNET_ERR_INVALID_VLAN_TAG_COUNT)
	VNET_ERROR_INVALID_ARGUMENT                   = VnetError(C.VNET_ERR_INVALID_ARGUMENT)
	VNET_ERROR_UNEXPECTED_INTF_STATE              = VnetError(C.VNET_ERR_UNEXPECTED_INTF_STATE)
	VNET_ERROR_TUNNEL_EXIST                       = VnetError(C.VNET_ERR_TUNNEL_EXIST)
	VNET_ERROR_INVALID_DECAP_NEXT                 = VnetError(C.VNET_ERR_INVALID_DECAP_NEXT)
	VNET_ERROR_RESPONSE_NOT_READY                 = VnetError(C.VNET_ERR_RESPONSE_NOT_READY)
	VNET_ERROR_NOT_CONNECTED                      = VnetError(C.VNET_ERR_NOT_CONNECTED)
	VNET_ERROR_IF_ALREADY_EXISTS                  = VnetError(C.VNET_ERR_IF_ALREADY_EXISTS)
	VNET_ERROR_BOND_SLAVE_NOT_ALLOWED             = VnetError(C.VNET_ERR_BOND_SLAVE_NOT_ALLOWED)
	VNET_ERROR_VALUE_EXIST                        = VnetError(C.VNET_ERR_VALUE_EXIST)
	VNET_ERROR_SAME_SRC_DST                       = VnetError(C.VNET_ERR_SAME_SRC_DST)
	VNET_ERROR_IP6_MULTICAST_ADDRESS_NOT_PRESENT  = VnetError(C.VNET_ERR_IP6_MULTICAST_ADDRESS_NOT_PRESENT)
	VNET_ERROR_SR_POLICY_NAME_NOT_PRESENT         = VnetError(C.VNET_ERR_SR_POLICY_NAME_NOT_PRESENT)
	VNET_ERROR_NOT_RUNNING_AS_ROOT                = VnetError(C.VNET_ERR_NOT_RUNNING_AS_ROOT)
	VNET_ERROR_ALREADY_CONNECTED                  = VnetError(C.VNET_ERR_ALREADY_CONNECTED)
	VNET_ERROR_UNSUPPORTED_JNI_VERSION            = VnetError(C.VNET_ERR_UNSUPPORTED_JNI_VERSION)
	VNET_ERROR_IP_PREFIX_INVALID                  = VnetError(C.VNET_ERR_IP_PREFIX_INVALID)
	VNET_ERROR_INVALID_WORKER                     = VnetError(C.VNET_ERR_INVALID_WORKER)
	VNET_ERROR_LISP_DISABLED                      = VnetError(C.VNET_ERR_LISP_DISABLED)
	VNET_ERROR_CLASSIFY_TABLE_NOT_FOUND           = VnetError(C.VNET_ERR_CLASSIFY_TABLE_NOT_FOUND)
	VNET_ERROR_INVALID_EID_TYPE                   = VnetError(C.VNET_ERR_INVALID_EID_TYPE)
	VNET_ERROR_CANNOT_CREATE_PCAP_FILE            = VnetError(C.VNET_ERR_CANNOT_CREATE_PCAP_FILE)
	VNET_ERROR_INCORRECT_ADJACENCY_TYPE           = VnetError(C.VNET_ERR_INCORRECT_ADJACENCY_TYPE)
	VNET_ERROR_EXCEEDED_NUMBER_OF_RANGES_CAPACITY = VnetError(C.VNET_ERR_EXCEEDED_NUMBER_OF_RANGES_CAPACITY)
	VNET_ERROR_EXCEEDED_NUMBER_OF_PORTS_CAPACITY  = VnetError(C.VNET_ERR_EXCEEDED_NUMBER_OF_PORTS_CAPACITY)
	VNET_ERROR_INVALID_ADDRESS_FAMILY             = VnetError(C.VNET_ERR_INVALID_ADDRESS_FAMILY)
	VNET_ERROR_INVALID_SUB_SW_IF_INDEX            = VnetError(C.VNET_ERR_INVALID_SUB_SW_IF_INDEX)
	VNET_ERROR_TABLE_TOO_BIG                      = VnetError(C.VNET_ERR_TABLE_TOO_BIG)
	VNET_ERROR_CANNOT_ENABLE_DISABLE_FEATURE      = VnetError(C.VNET_ERR_CANNOT_ENABLE_DISABLE_FEATURE)
	VNET_ERROR_BFD_EEXIST                         = VnetError(C.VNET_ERR_BFD_EEXIST)
	VNET_ERROR_BFD_ENOENT                         = VnetError(C.VNET_ERR_BFD_ENOENT)
	VNET_ERROR_BFD_EINUSE                         = VnetError(C.VNET_ERR_BFD_EINUSE)
	VNET_ERROR_BFD_NOTSUPP                        = VnetError(C.VNET_ERR_BFD_NOTSUPP)
	VNET_ERROR_ADDRESS_IN_USE                     = VnetError(C.VNET_ERR_ADDRESS_IN_USE)
	VNET_ERROR_ADDRESS_NOT_IN_USE                 = VnetError(C.VNET_ERR_ADDRESS_NOT_IN_USE)
	VNET_ERROR_QUEUE_FULL                         = VnetError(C.VNET_ERR_QUEUE_FULL)
	VNET_ERROR_APP_UNSUPPORTED_CFG                = VnetError(C.VNET_ERR_APP_UNSUPPORTED_CFG)
	VNET_ERROR_URI_FIFO_CREATE_FAILED             = VnetError(C.VNET_ERR_URI_FIFO_CREATE_FAILED)
	VNET_ERROR_LISP_RLOC_LOCAL                    = VnetError(C.VNET_ERR_LISP_RLOC_LOCAL)
	VNET_ERROR_BFD_EAGAIN                         = VnetError(C.VNET_ERR_BFD_EAGAIN)
	VNET_ERROR_INVALID_GPE_MODE                   = VnetError(C.VNET_ERR_INVALID_GPE_MODE)
	VNET_ERROR_LISP_GPE_ENTRIES_PRESENT           = VnetError(C.VNET_ERR_LISP_GPE_ENTRIES_PRESENT)
	VNET_ERROR_ADDRESS_FOUND_FOR_INTERFACE        = VnetError(C.VNET_ERR_ADDRESS_FOUND_FOR_INTERFACE)
	VNET_ERROR_SESSION_CONNECT                    = VnetError(C.VNET_ERR_SESSION_CONNECT)
	VNET_ERROR_ENTRY_ALREADY_EXISTS               = VnetError(C.VNET_ERR_ENTRY_ALREADY_EXISTS)
	VNET_ERROR_SVM_SEGMENT_CREATE_FAIL            = VnetError(C.VNET_ERR_SVM_SEGMENT_CREATE_FAIL)
	VNET_ERROR_APPLICATION_NOT_ATTACHED           = VnetError(C.VNET_ERR_APPLICATION_NOT_ATTACHED)
	VNET_ERROR_BD_ALREADY_EXISTS                  = VnetError(C.VNET_ERR_BD_ALREADY_EXISTS)
	VNET_ERROR_BD_IN_USE                          = VnetError(C.VNET_ERR_BD_IN_USE)
	VNET_ERROR_BD_NOT_MODIFIABLE                  = VnetError(C.VNET_ERR_BD_NOT_MODIFIABLE)
	VNET_ERROR_BD_ID_EXCEED_MAX                   = VnetError(C.VNET_ERR_BD_ID_EXCEED_MAX)
	VNET_ERROR_SUBIF_DOESNT_EXIST                 = VnetError(C.VNET_ERR_SUBIF_DOESNT_EXIST)
	VNET_ERROR_L2_MACS_EVENT_CLINET_PRESENT       = VnetError(C.VNET_ERR_L2_MACS_EVENT_CLINET_PRESENT)
	VNET_ERROR_INVALID_QUEUE                      = VnetError(C.VNET_ERR_INVALID_QUEUE)
	VNET_ERROR_UNSUPPORTED                        = VnetError(C.VNET_ERR_UNSUPPORTED)
	VNET_ERROR_DUPLICATE_IF_ADDRESS               = VnetError(C.VNET_ERR_DUPLICATE_IF_ADDRESS)
	VNET_ERROR_APP_INVALID_NS                     = VnetError(C.VNET_ERR_APP_INVALID_NS)
	VNET_ERROR_APP_WRONG_NS_SECRET                = VnetError(C.VNET_ERR_APP_WRONG_NS_SECRET)
	VNET_ERROR_APP_CONNECT_SCOPE                  = VnetError(C.VNET_ERR_APP_CONNECT_SCOPE)
	VNET_ERROR_APP_ALREADY_ATTACHED               = VnetError(C.VNET_ERR_APP_ALREADY_ATTACHED)
	VNET_ERROR_SESSION_REDIRECT                   = VnetError(C.VNET_ERR_SESSION_REDIRECT)
	VNET_ERROR_ILLEGAL_NAME                       = VnetError(C.VNET_ERR_ILLEGAL_NAME)
	VNET_ERROR_NO_NAME_SERVERS                    = VnetError(C.VNET_ERR_NO_NAME_SERVERS)
	VNET_ERROR_NAME_SERVER_NOT_FOUND              = VnetError(C.VNET_ERR_NAME_SERVER_NOT_FOUND)
	VNET_ERROR_NAME_RESOLUTION_NOT_ENABLED        = VnetError(C.VNET_ERR_NAME_RESOLUTION_NOT_ENABLED)
	VNET_ERROR_NAME_SERVER_FORMAT_ERROR           = VnetError(C.VNET_ERR_NAME_SERVER_FORMAT_ERROR)
	VNET_ERROR_NAME_SERVER_NO_SUCH_NAME           = VnetError(C.VNET_ERR_NAME_SERVER_NO_SUCH_NAME)
	VNET_ERROR_NAME_SERVER_NO_ADDRESSES           = VnetError(C.VNET_ERR_NAME_SERVER_NO_ADDRESSES)
	VNET_ERROR_NAME_SERVER_NEXT_SERVER            = VnetError(C.VNET_ERR_NAME_SERVER_NEXT_SERVER)
	VNET_ERROR_APP_CONNECT_FILTERED               = VnetError(C.VNET_ERR_APP_CONNECT_FILTERED)
	VNET_ERROR_ACL_IN_USE_INBOUND                 = VnetError(C.VNET_ERR_ACL_IN_USE_INBOUND)
	VNET_ERROR_ACL_IN_USE_OUTBOUND                = VnetError(C.VNET_ERR_ACL_IN_USE_OUTBOUND)
	VNET_ERROR_INIT_FAILED                        = VnetError(C.VNET_ERR_INIT_FAILED)
	VNET_ERROR_NETLINK_ERROR                      = VnetError(C.VNET_ERR_NETLINK_ERROR)
	VNET_ERROR_BIER_BSL_UNSUP                     = VnetError(C.VNET_ERR_BIER_BSL_UNSUP)
	VNET_ERROR_INSTANCE_IN_USE                    = VnetError(C.VNET_ERR_INSTANCE_IN_USE)
	VNET_ERROR_INVALID_SESSION_ID                 = VnetError(C.VNET_ERR_INVALID_SESSION_ID)
	VNET_ERROR_ACL_IN_USE_BY_LOOKUP_CONTEXT       = VnetError(C.VNET_ERR_ACL_IN_USE_BY_LOOKUP_CONTEXT)
	VNET_ERROR_INVALID_VALUE_3                    = VnetError(C.VNET_ERR_INVALID_VALUE_3)
	VNET_ERROR_NON_ETHERNET                       = VnetError(C.VNET_ERR_NON_ETHERNET)
	VNET_ERROR_BD_ALREADY_HAS_BVI                 = VnetError(C.VNET_ERR_BD_ALREADY_HAS_BVI)
	VNET_ERROR_INVALID_PROTOCOL                   = VnetError(C.VNET_ERR_INVALID_PROTOCOL)
	VNET_ERROR_INVALID_ALGORITHM                  = VnetError(C.VNET_ERR_INVALID_ALGORITHM)
	VNET_ERROR_RSRC_IN_USE                        = VnetError(C.VNET_ERR_RSRC_IN_USE)
	VNET_ERROR_KEY_LENGTH                         = VnetError(C.VNET_ERR_KEY_LENGTH)
	VNET_ERROR_FIB_PATH_UNSUPPORTED_NH_PROTO      = VnetError(C.VNET_ERR_FIB_PATH_UNSUPPORTED_NH_PROTO)
	VNET_ERROR_API_ENDIAN_FAILED                  = VnetError(C.VNET_ERR_API_ENDIAN_FAILED)
	VNET_ERROR_NO_CHANGE                          = VnetError(C.VNET_ERR_NO_CHANGE)
	VNET_ERROR_MISSING_CERT_KEY                   = VnetError(C.VNET_ERR_MISSING_CERT_KEY)
	VNET_ERROR_LIMIT_EXCEEDED                     = VnetError(C.VNET_ERR_LIMIT_EXCEEDED)
	VNET_ERROR_IKE_NO_PORT                        = VnetError(C.VNET_ERR_IKE_NO_PORT)
	VNET_ERROR_UDP_PORT_TAKEN                     = VnetError(C.VNET_ERR_UDP_PORT_TAKEN)
	VNET_ERROR_EAGAIN                             = VnetError(C.VNET_ERR_EAGAIN)
	VNET_ERROR_INVALID_VALUE_4                    = VnetError(C.VNET_ERR_INVALID_VALUE_4)
	VNET_ERROR_BUSY                               = VnetError(C.VNET_ERR_BUSY)
	VNET_ERROR_BUG                                = VnetError(C.VNET_ERR_BUG)
	VNET_ERROR_FEATURE_ALREADY_DISABLED           = VnetError(C.VNET_ERR_FEATURE_ALREADY_DISABLED)
	VNET_ERROR_FEATURE_ALREADY_ENABLED            = VnetError(C.VNET_ERR_FEATURE_ALREADY_ENABLED)
	VNET_ERROR_INVALID_PREFIX_LENGTH              = VnetError(C.VNET_ERR_INVALID_PREFIX_LENGTH)
)

var vnet_error_strings = [...]string{
	-VNET_ERROR_OK:                                 "OK",
	-VNET_ERROR_UNSPECIFIED:                        "Unspecified",
	-VNET_ERROR_INVALID_SW_IF_INDEX:                "Invalid software interface",
	-VNET_ERROR_NO_SUCH_FIB:                        "No such FIB / VRF",
	-VNET_ERROR_NO_SUCH_INNER_FIB:                  "No such inner FIB / VRF",
	-VNET_ERROR_NO_SUCH_LABEL:                      "No such label",
	-VNET_ERROR_NO_SUCH_ENTRY:                      "No such entry",
	-VNET_ERROR_INVALID_VALUE:                      "Invalid value",
	-VNET_ERROR_INVALID_VALUE_2:                    "Invalid value #2",
	-VNET_ERROR_UNIMPLEMENTED:                      "Unimplemented",
	-VNET_ERROR_INVALID_SW_IF_INDEX_2:              "Invalid sw_if_index #2",
	-VNET_ERROR_SYSCALL_ERROR_1:                    "System call error #1",
	-VNET_ERROR_SYSCALL_ERROR_2:                    "System call error #2",
	-VNET_ERROR_SYSCALL_ERROR_3:                    "System call error #3",
	-VNET_ERROR_SYSCALL_ERROR_4:                    "System call error #4",
	-VNET_ERROR_SYSCALL_ERROR_5:                    "System call error #5",
	-VNET_ERROR_SYSCALL_ERROR_6:                    "System call error #6",
	-VNET_ERROR_SYSCALL_ERROR_7:                    "System call error #7",
	-VNET_ERROR_SYSCALL_ERROR_8:                    "System call error #8",
	-VNET_ERROR_SYSCALL_ERROR_9:                    "System call error #9",
	-VNET_ERROR_SYSCALL_ERROR_10:                   "System call error #10",
	-VNET_ERROR_FEATURE_DISABLED:                   "Feature disabled by configuration",
	-VNET_ERROR_INVALID_REGISTRATION:               "Invalid registration",
	-VNET_ERROR_NEXT_HOP_NOT_IN_FIB:                "Next hop not in FIB",
	-VNET_ERROR_UNKNOWN_DESTINATION:                "Unknown destination",
	-VNET_ERROR_NO_PATHS_IN_ROUTE:                  "No paths specified in route",
	-VNET_ERROR_NEXT_HOP_NOT_FOUND_MP:              "Next hop not found (multipath)",
	-VNET_ERROR_NO_MATCHING_INTERFACE:              "No matching interface for probe",
	-VNET_ERROR_INVALID_VLAN:                       "Invalid VLAN",
	-VNET_ERROR_VLAN_ALREADY_EXISTS:                "VLAN subif already exists",
	-VNET_ERROR_INVALID_SRC_ADDRESS:                "Invalid src address",
	-VNET_ERROR_INVALID_DST_ADDRESS:                "Invalid dst address",
	-VNET_ERROR_ADDRESS_LENGTH_MISMATCH:            "Address length mismatch",
	-VNET_ERROR_ADDRESS_NOT_FOUND_FOR_INTERFACE:    "Address not found for interface",
	-VNET_ERROR_ADDRESS_NOT_DELETABLE:              "Address not deletable",
	-VNET_ERROR_IP6_NOT_ENABLED:                    "ip6 not enabled",
	-VNET_ERROR_NO_SUCH_NODE:                       "No such graph node",
	-VNET_ERROR_NO_SUCH_NODE2:                      "No such graph node #2",
	-VNET_ERROR_NO_SUCH_TABLE:                      "No such table",
	-VNET_ERROR_NO_SUCH_TABLE2:                     "No such table #2",
	-VNET_ERROR_NO_SUCH_TABLE3:                     "No such table #3",
	-VNET_ERROR_SUBIF_ALREADY_EXISTS:               "Subinterface already exists",
	-VNET_ERROR_SUBIF_CREATE_FAILED:                "Subinterface creation failed",
	-VNET_ERROR_INVALID_MEMORY_SIZE:                "Invalid memory size requested",
	-VNET_ERROR_INVALID_INTERFACE:                  "Invalid interface",
	-VNET_ERROR_INVALID_VLAN_TAG_COUNT:             "Invalid number of tags for requested operation",
	-VNET_ERROR_INVALID_ARGUMENT:                   "Invalid argument",
	-VNET_ERROR_UNEXPECTED_INTF_STATE:              "Unexpected interface state",
	-VNET_ERROR_TUNNEL_EXIST:                       "Tunnel already exists",
	-VNET_ERROR_INVALID_DECAP_NEXT:                 "Invalid decap-next",
	-VNET_ERROR_RESPONSE_NOT_READY:                 "Response not ready",
	-VNET_ERROR_NOT_CONNECTED:                      "Not connected to the data plane",
	-VNET_ERROR_IF_ALREADY_EXISTS:                  "Interface already exists",
	-VNET_ERROR_BOND_SLAVE_NOT_ALLOWED:             "Operation not allowed on slave of BondEthernet",
	-VNET_ERROR_VALUE_EXIST:                        "Value already exists",
	-VNET_ERROR_SAME_SRC_DST:                       "Source and destination are the same",
	-VNET_ERROR_IP6_MULTICAST_ADDRESS_NOT_PRESENT:  "IP6 multicast address required",
	-VNET_ERROR_SR_POLICY_NAME_NOT_PRESENT:         "Segment routing policy name required",
	-VNET_ERROR_NOT_RUNNING_AS_ROOT:                "Not running as root",
	-VNET_ERROR_ALREADY_CONNECTED:                  "Connection to the data plane already exists",
	-VNET_ERROR_UNSUPPORTED_JNI_VERSION:            "Unsupported JNI version",
	-VNET_ERROR_IP_PREFIX_INVALID:                  "IP prefix invalid (masked bits set in address",
	-VNET_ERROR_INVALID_WORKER:                     "Invalid worker thread",
	-VNET_ERROR_LISP_DISABLED:                      "LISP is disabled",
	-VNET_ERROR_CLASSIFY_TABLE_NOT_FOUND:           "Classify table not found",
	-VNET_ERROR_INVALID_EID_TYPE:                   "Unsupported LISP EID type",
	-VNET_ERROR_CANNOT_CREATE_PCAP_FILE:            "Cannot create pcap file",
	-VNET_ERROR_INCORRECT_ADJACENCY_TYPE:           "Invalid adjacency type for this operation",
	-VNET_ERROR_EXCEEDED_NUMBER_OF_RANGES_CAPACITY: "Operation would exceed configured capacity of ranges",
	-VNET_ERROR_EXCEEDED_NUMBER_OF_PORTS_CAPACITY:  "Operation would exceed capacity of number of ports",
	-VNET_ERROR_INVALID_ADDRESS_FAMILY:             "Invalid address family",
	-VNET_ERROR_INVALID_SUB_SW_IF_INDEX:            "Invalid sub-interface sw_if_index",
	-VNET_ERROR_TABLE_TOO_BIG:                      "Table too big",
	-VNET_ERROR_CANNOT_ENABLE_DISABLE_FEATURE:      "Cannot enable/disable feature",
	-VNET_ERROR_BFD_EEXIST:                         "Duplicate BFD object",
	-VNET_ERROR_BFD_ENOENT:                         "No such BFD object",
	-VNET_ERROR_BFD_EINUSE:                         "BFD object in use",
	-VNET_ERROR_BFD_NOTSUPP:                        "BFD feature not supported",
	-VNET_ERROR_ADDRESS_IN_USE:                     "Address in use",
	-VNET_ERROR_ADDRESS_NOT_IN_USE:                 "Address not in use",
	-VNET_ERROR_QUEUE_FULL:                         "Queue full",
	-VNET_ERROR_APP_UNSUPPORTED_CFG:                "Unsupported application config",
	-VNET_ERROR_URI_FIFO_CREATE_FAILED:             "URI FIFO segment create failed",
	-VNET_ERROR_LISP_RLOC_LOCAL:                    "RLOC address is local",
	-VNET_ERROR_BFD_EAGAIN:                         "BFD object cannot be manipulated at this time",
	-VNET_ERROR_INVALID_GPE_MODE:                   "Invalid GPE mode",
	-VNET_ERROR_LISP_GPE_ENTRIES_PRESENT:           "LISP GPE entries are present",
	-VNET_ERROR_ADDRESS_FOUND_FOR_INTERFACE:        "Address found for interface",
	-VNET_ERROR_SESSION_CONNECT:                    "Session failed to connect",
	-VNET_ERROR_ENTRY_ALREADY_EXISTS:               "Entry already exists",
	-VNET_ERROR_SVM_SEGMENT_CREATE_FAIL:            "Svm segment create fail",
	-VNET_ERROR_APPLICATION_NOT_ATTACHED:           "Application not attached",
	-VNET_ERROR_BD_ALREADY_EXISTS:                  "Bridge domain already exists",
	-VNET_ERROR_BD_IN_USE:                          "Bridge domain has member interfaces",
	-VNET_ERROR_BD_NOT_MODIFIABLE:                  "Bridge domain 0 can't be deleted/modified",
	-VNET_ERROR_BD_ID_EXCEED_MAX:                   "Bridge domain ID exceeds 16M limit",
	-VNET_ERROR_SUBIF_DOESNT_EXIST:                 "Subinterface doesn't exist",
	-VNET_ERROR_L2_MACS_EVENT_CLINET_PRESENT:       "Client already exist for L2 MACs events",
	-VNET_ERROR_INVALID_QUEUE:                      "Invalid queue",
	-VNET_ERROR_UNSUPPORTED:                        "Unsupported",
	-VNET_ERROR_DUPLICATE_IF_ADDRESS:               "Address already present on another interface",
	-VNET_ERROR_APP_INVALID_NS:                     "Invalid application namespace",
	-VNET_ERROR_APP_WRONG_NS_SECRET:                "Wrong app namespace secret",
	-VNET_ERROR_APP_CONNECT_SCOPE:                  "Connect scope",
	-VNET_ERROR_APP_ALREADY_ATTACHED:               "App already attached",
	-VNET_ERROR_SESSION_REDIRECT:                   "Redirect failed",
	-VNET_ERROR_ILLEGAL_NAME:                       "Illegal name",
	-VNET_ERROR_NO_NAME_SERVERS:                    "No name servers configured",
	-VNET_ERROR_NAME_SERVER_NOT_FOUND:              "Name server not found",
	-VNET_ERROR_NAME_RESOLUTION_NOT_ENABLED:        "Name resolution not enabled",
	-VNET_ERROR_NAME_SERVER_FORMAT_ERROR:           "Server format error (bug!)",
	-VNET_ERROR_NAME_SERVER_NO_SUCH_NAME:           "No such name",
	-VNET_ERROR_NAME_SERVER_NO_ADDRESSES:           "No addresses available",
	-VNET_ERROR_NAME_SERVER_NEXT_SERVER:            "Retry with new server",
	-VNET_ERROR_APP_CONNECT_FILTERED:               "Connect was filtered",
	-VNET_ERROR_ACL_IN_USE_INBOUND:                 "Inbound ACL in use",
	-VNET_ERROR_ACL_IN_USE_OUTBOUND:                "Outbound ACL in use",
	-VNET_ERROR_INIT_FAILED:                        "Initialization Failed",
	-VNET_ERROR_NETLINK_ERROR:                      "Netlink error",
	-VNET_ERROR_BIER_BSL_UNSUP:                     "BIER bit-string-length unsupported",
	-VNET_ERROR_INSTANCE_IN_USE:                    "Instance in use",
	-VNET_ERROR_INVALID_SESSION_ID:                 "Session ID out of range",
	-VNET_ERROR_ACL_IN_USE_BY_LOOKUP_CONTEXT:       "ACL in use by a lookup context",
	-VNET_ERROR_INVALID_VALUE_3:                    "Invalid value #3",
	-VNET_ERROR_NON_ETHERNET:                       "Interface is not an Ethernet interface",
	-VNET_ERROR_BD_ALREADY_HAS_BVI:                 "Bridge domain already has a BVI interface",
	-VNET_ERROR_INVALID_PROTOCOL:                   "Invalid Protocol",
	-VNET_ERROR_INVALID_ALGORITHM:                  "Invalid Algorithm",
	-VNET_ERROR_RSRC_IN_USE:                        "Resource In Use",
	-VNET_ERROR_KEY_LENGTH:                         "invalid Key Length",
	-VNET_ERROR_FIB_PATH_UNSUPPORTED_NH_PROTO:      "Unsupported FIB Path protocol",
	-VNET_ERROR_API_ENDIAN_FAILED:                  "Endian mismatch detected",
	-VNET_ERROR_NO_CHANGE:                          "No change in table",
	-VNET_ERROR_MISSING_CERT_KEY:                   "Missing certifcate or key",
	-VNET_ERROR_LIMIT_EXCEEDED:                     "limit exceeded",
	-VNET_ERROR_IKE_NO_PORT:                        "port not managed by IKE",
	-VNET_ERROR_UDP_PORT_TAKEN:                     "UDP port already taken",
	-VNET_ERROR_EAGAIN:                             "Retry stream call with cursor",
	-VNET_ERROR_INVALID_VALUE_4:                    "Invalid value #4",
	-VNET_ERROR_BUSY:                               "Busy",
	-VNET_ERROR_BUG:                                "Bug",
	-VNET_ERROR_FEATURE_ALREADY_DISABLED:           "Feature already disabled",
	-VNET_ERROR_FEATURE_ALREADY_ENABLED:            "Feature already enabled",
	-VNET_ERROR_INVALID_PREFIX_LENGTH:              "Invalid prefix length",
}

func (e VnetError) Format(b *std.Buffer) {
	n := -int(e)
	if n < len(vnet_error_strings) && vnet_error_strings[n] != "" {
		b.AddString(vnet_error_strings[n])
	} else {
		b.Format("unknown %d", n)
	}
}

type (
	CryptoEngine               uint8
	SessionError               int32 // error returns from vnet session code
	SessionState               uint8
	SessionEndpointConfigFlags uint8
	TransportProto             uint8
	VnetError                  int32
)

const (
	SwIntfAdminUp           = SwIntfFlags(C.VNET_SW_INTERFACE_FLAG_ADMIN_UP)
	SwIntfPunt              = SwIntfFlags(C.VNET_SW_INTERFACE_FLAG_PUNT)
	SwIntfUnnumbered        = SwIntfFlags(C.VNET_SW_INTERFACE_FLAG_UNNUMBERED)
	SwIntfHidden            = SwIntfFlags(C.VNET_SW_INTERFACE_FLAG_HIDDEN)
	SwIntfError             = SwIntfFlags(C.VNET_SW_INTERFACE_FLAG_ERROR)
	SwIntfDirectedBroadcast = SwIntfFlags(C.VNET_SW_INTERFACE_FLAG_DIRECTED_BCAST)
)

type (
	SwIntfFlags uint32
	HwIntfFlags uint32
)
