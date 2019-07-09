package bgp

import (
	"fmt"
	"strings"
)

var _16ByteMaker = [16]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

// Message Border Gateway Protocol (BGP) Message
type Message struct {
	Marker [16]byte
	Length uint16
	Type   MessageType
	Body   interface{}
}

// MessageType type of BGP message
type MessageType uint8

const (
	_Open MessageType = 1 + iota
	_Update
	_Notification
	_Keepalive
)

func (t MessageType) String() string {
	switch t {
	case _Open:
		return "OPEN"
	case _Update:
		return "UPDATE"
	case _Notification:
		return "NOTIFICATION"
	case _Keepalive:
		return "KEEPALIVE"
	}
	return fmt.Sprintf("Unknown(MessageType=%d)", int(t))
}

// InstanceFor interface implementation to provide struct for the body
func (bgp Message) InstanceFor(fieldname string) interface{} {
	switch bgp.Type {
	case _Open:
		return &Open{}
	case _Update:
		return &Update{}
	case _Notification:
		return &Notification{}
	case _Keepalive:
		return &Keepalive{}
	}
	return nil
}

// Open message of BGP
type Open struct {
	Version        uint8
	AS             uint16
	Holdtime       uint16
	RouterID       uint32
	OptionalLength uint8
	Optional       []OptionalParameter `packet:"lengthfor"`
}

func (o Open) LengthFor(fieldname string) uint64 {
	return 0
}

// OptionalParameter defines the optional parameter in BGP OPEN message as per https://tools.ietf.org/html/rfc4271#section-4.2
type OptionalParameter struct {
	Type   uint8
	Length uint8
	Data   interface{} `packet:"lengthfrom=ParameterLength"`
}

// PrefixSpec is a compact container for route specification in BGP messages,
// which consist of Length for how many bits are in a network Prefix
type PrefixSpec struct {
	Length uint8
	Prefix []byte `packet:"lengthfor"`
}

// LengthFor to return the byte length of Length value, which depends on Flags
func (p PrefixSpec) LengthFor(fieldname string) uint64 {
	l := uint64(p.Length / 8)
	if p.Length%8 != 0 {
		l++
	}
	return l
}

// AttributeFlag flags for Path Attributes
type AttributeFlag uint8

const (
	// ExtendedLength - attribute flag
	ExtendedLength AttributeFlag = 1 << iota
	// Partial - attribute flag
	Partial
	// Transitive - attribute flag
	Transitive
	// Optional - attribute flag
	Optional
)

// AttributeType Path Attribute Type as defined in https://tools.ietf.org/html/rfc4271#section-5.1
type AttributeType uint8

/* attribute type */
const (
	Origin AttributeType = 1 + iota
	AsPath
	Nexthop
	MultiExitDisc
	LocalPref
	AtomicAggregate
	Aggregator
	Community
	OriginatorID
	ClusterList
	MPReachNLRI   AttributeType = 14
	MPUnreachNLRI AttributeType = 15
)

// String conversions for CapabilityCode
func (t AttributeType) String() string {
	switch t {
	case Origin:
		return "ORIGIN"
	case AsPath:
		return "AS_PATH"
	case Nexthop:
		return "NEXTHOP"
	case MultiExitDisc:
		return "MULTI_EXIT_DISC"
	case LocalPref:
		return "LOCAL_PREF"
	case AtomicAggregate:
		return "ATOMIC_AGGREGATE"
	case Aggregator:
		return "AGGREGATOR"
	case Community:
		return "COMMUNITY"
	case OriginatorID:
		return "ORIGINATOR_ID"
	case ClusterList:
		return "CLUSTER_LIST"
	case MPReachNLRI:
		return "MP_REACH_NLRI"
	case MPUnreachNLRI:
		return "MP_UNREACH_NLRI"
	default:
		return fmt.Sprintf("AttributeType(%d)", int(t))
	}
}

// PathAttribute defines the Path Attribute in BGP UPDATE message, as per (https://tools.ietf.org/html/rfc4271#section-4.3)
// Length is 2 bytes when (Flags & 0x01) != 0, or 1 byte otherwise.
type PathAttribute struct {
	Flags  AttributeFlag
	Code   AttributeType
	Length uint16      `packet:"lengthfor"`
	Data   interface{} `packet:"lengthfor"`
}

// OriginCode origin code
type OriginCode uint8

const (
	// IBGP Internal Border Gateway Protocol
	IBGP OriginCode = iota
	// EBGP External Border Gateway Protocol
	EBGP
	// INCOMPLETE incomplete origin
	INCOMPLETE
)

func (o OriginCode) String() string {
	switch o {
	case IBGP:
		return "IBGP"
	case EBGP:
		return "EBGP"
	case INCOMPLETE:
		return "INCOMPLETE"
	}
	return fmt.Sprintf("Origin(%d)", int(o))
}

// OriginAttribute is Origin Path Attribute
type OriginAttribute struct {
	Origin OriginCode
}

// AsPathType AS type
type AsPathType uint8

const (
	// AsSet AS set
	AsSet AsPathType = 1
	// AsSequence AS sequence
	AsSequence AsPathType = 2
)

// ASN BGP Autonomous System Number
type ASN uint16

// AsPathAttribute AS path attribute
type AsPathAttribute struct {
	Type  AsPathType
	Count uint8
	List  []ASN `packet:"countfrom=Count"`
}

func (f AttributeFlag) String() string {
	s := make([]string, 0)
	if (f & Optional) != 0 {
		s = append(s, "Optional")
	}
	if (f & Transitive) != 0 {
		s = append(s, "Transitive")
	}
	if (f & Partial) != 0 {
		s = append(s, "Partial")
	}
	if (f & ExtendedLength) != 0 {
		s = append(s, "ExtendedLength")
	}
	return strings.Join(s, "|")
}

// InstanceFor interface implementation to provide struct for the body
func (p PathAttribute) InstanceFor(fieldname string) interface{} {
	switch p.Code {
	case Origin:
		return &OriginAttribute{}
	case AsPath:
		return &[]AsPathAttribute{}
	}
	b := make([]byte, p.Length)
	return &b
}

// LengthFor to return the byte length of Length value, which depends on Flags
func (p PathAttribute) LengthFor(fieldname string) uint64 {
	switch fieldname {
	case "Length":
		if (p.Flags & ExtendedLength) != 0 {
			return 2
		}
		return 1
	case "Data":
		return uint64(p.Length)
	}
	return 0
}

// Update message struct as defined in https://tools.ietf.org/html/rfc4271#section-4.3
type Update struct {
	WithdrawnLength     uint16
	WithdrawnRoutes     []PrefixSpec `packet:"lengthfor"`
	PathAttributeLength uint16
	PathAttributes      []PathAttribute `packet:"lengthfor"`
	NLRI                []PrefixSpec    `packet:"lengthrest"`
}

// LengthFor
func (u Update) LengthFor(fieldname string) uint64 {
	switch fieldname {
	case "WithdrawnRoutes":
		return uint64(u.WithdrawnLength)
	case "PathAttributes":
		return uint64(u.PathAttributeLength)
	}
	return 0
}

// ErrorType BGP error message type as defined in https://tools.ietf.org/html/rfc4271#section-4.5
type ErrorType uint8

// Notification struct from RFC 4271 - Section 4.5
type Notification struct {
	Code    ErrorType
	Subcode uint8
	Content []byte
}

// Keepalive is an intentionally empty struct as defined in https://tools.ietf.org/html/rfc4271#section-4.4
type Keepalive struct {
}
