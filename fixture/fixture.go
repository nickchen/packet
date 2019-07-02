// Package fixture serve as unittest sample for basic packet encode/decode
package fixture

import (
	"fmt"
	"net"
	"strings"
)

type IpProtocol uint8

const (
	_TCP IpProtocol = 6
	_UDP IpProtocol = 17
)

func (p IpProtocol) String() string {
	switch p {
	case _TCP:
		return "TCP"
	case _UDP:
		return "UDP"
	}
	return fmt.Sprintf("Protocol(unknown:%d)", int(p))
}

// Checksum conversion so it can display in hex
type Checksum uint16

func (c Checksum) String() string {
	return fmt.Sprintf("0x%x", int(c))
}

type IPv4Flag uint8

const (
	MFrag IPv4Flag = 1 << iota
	DFrag
	Reserved
)

func (f IPv4Flag) String() string {
	s := make([]string, 0)
	if Reserved&f != 0 {
		s = append(s, "Reserved")
	}
	if DFrag&f != 0 {
		s = append(s, "DFrag")
	}
	if MFrag&f != 0 {
		s = append(s, "MFrag")
	}
	return fmt.Sprintf("%s", strings.Join(s, "|"))
}

// IPv4 packet
type IPv4 struct {
	Version        uint8 `packet:"size=4b"`
	IHL            uint8 `packet:"size=4b"`
	DSCP           uint8 `packet:"size=6b"`
	ECN            uint8 `packet:"size=2b"`
	Length         uint16
	Id             uint16
	Flags          IPv4Flag `packet:"size=3b"`
	FragmentOffset uint16   `packet:"size=13b"`
	TTL            uint8
	Protocol       IpProtocol
	Checksum       Checksum
	Source         net.IP      `packet:"size=4B"`
	Dest           net.IP      `packet:"size=4B"`
	Options        []byte      `packet:"when=IHL-gt-5"`
	Body           interface{} `packet:rest=Length`
}

type Port uint8

// Well know ports
const (
	_BGP Port = 179
)

// TCP message
type TCP struct {
	Source        Port
	Dest          Port
	Sequence      uint32
	Ack           uint32
	DataOffset    uint8
	Reserved      uint8
	FlagNS        bool
	FlagCWR       bool
	FlagECE       bool
	FlagURG       bool
	FlagACK       bool
	FlagPSH       bool
	FlagRST       bool
	FlagSYN       bool
	FlagFIN       bool
	WindowSize    uint16
	Checksum      uint16
	UrgentPointer uint16
	Options       []byte `packet:when=Offset`
	Body          interface{}
}

// UnmarshalBody return the Body struct pointer for conversion
func (ip IPv4) UnmarshalBody() interface{} {
	switch ip.Protocol {
	case _TCP:
		return &TCP{}
	case _UDP:
	}
	// panic(fmt.Errorf("unhandle protocol (%s)", ip.Protocol))
	return nil
}

// UnmarshalBody return the Body struct pointer for conversion
func (tcp TCP) UnmarshalBody() interface{} {
	switch tcp.Dest {
	case _BGP:
		return &BGP{}
	}
	return nil
}

// BGP Border Gateway Protocol
type BGP struct {
	Marker [16]byte
	Length uint16
	Type   uint8
	Body   interface{} `packet:"type=Open,type=Update,type=Notification,type=Keepalive,source=Type"`
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
