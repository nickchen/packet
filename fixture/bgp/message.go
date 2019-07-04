package bgp

import "fmt"

// BGP Border Gateway Protocol
type Message struct {
	Marker [16]byte
	Length uint16
	Type   MessageType
	Body   interface{}
}

type MessageType uint8

const (
	_OPEN MessageType = 1 + iota
	_UPDATE
	_NOTIFICATION
	_KEEPALIVE
)

func (t MessageType) String() string {
	switch t {
	case _OPEN:
		return "OPEN"
	case _UPDATE:
		return "UPDATE"
	case _NOTIFICATION:
		return "NOTIFICATION"
	case _KEEPALIVE:
		return "KEEPALIVE"
	}
	return fmt.Sprintf("Unknown(MessageType=%d)", int(t))
}

func (bgp Message) UnmarshalBody() interface{} {
	switch bgp.Type {
	case _OPEN:
		return &Open{}
	case _UPDATE:
		return &Update{}
	case _NOTIFICATION:
		return &Notification{}
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

type OptionalParameter struct {
	Type   uint8
	Length uint8 `packet:"size_for=Value"`
	Value  interface{}
}

type Update struct {
	WithdrawnLength     uint16
	WithdrawnRoutes     interface{}
	PathAttributeLength uint16
	PathAttributes      interface{}
	NLRI                interface{}
}

type ErrorType uint8

// NotificationMessage struct from RFC 4271 - Section 4.5
type Notification struct {
	Code    ErrorType
	Subcode uint8
	Content []byte
}

// Keepalive is an intentionally empty struct
type Keepalive struct {
}
