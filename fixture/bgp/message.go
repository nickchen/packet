package bgp

import "fmt"

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
	Length uint8
	Data   interface{} `packet:"lengthfrom=Length"`
}

// RouteSpec is a compact container for route specification in BGP messages,
// which consist of Length for how many bits are in a network Prefix
type RouteSpec struct {
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

// PathAttribute defines the Path Attribute in BGP UPDATE message, as per (https://tools.ietf.org/html/rfc4271#section-4.3)
type PathAttribute struct {
	Flags  AttributeFlag
	Code   AttributeType
	Length uint16
	Data   interface{} `packet:"lengthfrom=Length"`
}

// Update message struct as defined in https://tools.ietf.org/html/rfc4271#section-4.3
type Update struct {
	WithdrawnLength     uint16
	WithdrawnRoutes     []RouteSpec `packet:"lengthfrom=WithdrawnLength"`
	PathAttributeLength uint16
	PathAttributes      []PathAttribute `packet:"lengthfrom=PathAttributeLength"`
	NLRI                []RouteSpec
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
