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

// BodyStruct interface implementation to provide struct for the body
func (bgp Message) BodyStruct() interface{} {
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
	fmt.Printf("BGP: %+v\n", bgp)
	return nil
}

// Open message of BGP
type Open struct {
	Version        uint8
	AS             uint16
	Holdtime       uint16
	RouterID       uint32
	OptionalLength uint8
	Optional       []OptionalParameter
}

// OptionalParameter defines the optional parameter in BGP OPEN message as per https://tools.ietf.org/html/rfc4271#section-4.2
type OptionalParameter struct {
	Type   uint8
	Length uint8       `packet:"datalength"`
	Data   interface{} `packet:"lengthfrom=ParameterLength"`
}

// PrefixSpec is a compact container for route specification in BGP messages,
// which consist of Length for how many bits are in a network Prefix
type PrefixSpec struct {
	Length uint8
	Prefix []byte `packet:"lengthfrom=Length"`
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
	Length uint16 `packet:"lengthfor"`
	Body   []byte `packet:"lengthfrom=Length"`
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

// BodyStruct interface implementation to provide struct for the body
func (p PathAttribute) BodyStruct() interface{} {
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
	}
	return 0
}

// Update message struct as defined in https://tools.ietf.org/html/rfc4271#section-4.3
type Update struct {
	WithdrawnLength     uint16
	WithdrawnRoutes     []PrefixSpec `packet:"lengthfrom=WithdrawnLength"`
	PathAttributeLength uint16
	PathAttributes      []PathAttribute `packet:"lengthfrom=PathAttributeLength"`
	NLRI                []PrefixSpec
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
