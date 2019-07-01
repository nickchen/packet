// Package fixture serve as unittest sample for basic packet encode/decode
package fixture

import (
	"fmt"
	"net"
)

type Checksum uint16

func (c Checksum) String() string {
	return fmt.Sprintf("0x%x", int(c))
}

type IP struct {
	Version        uint8 `packet:"size=4b"`
	HeaderLength   uint8 `packet:"size=4b"`
	DSCP           uint8 `packet:"size=6b"`
	ECN            uint8 `packet:"size=2b"`
	Length         uint16
	Identification uint16
	Flags          uint8  `packet:"size=3b"`
	FragmentOffset uint16 `packet:"size=13b"`
	TTL            uint8
	Protocol       uint8
	Checksum       Checksum
	Source         net.IP `packet:"size=4B"`
	Dest           net.IP `packet:"size=4B"`
	Options        []byte `packet:"when=HeaderLength-gt-5"`
	Body           []byte `packet:rest=Length`
}

type BGP struct {
	Marker [16]byte
	Length uint16
	Type   uint8
	Body   interface{} `packet:"type=Open,type=Update,type=Notification,type=Keepalive,source=Type"`
}

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
