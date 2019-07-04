// Package fixture serve as unittest sample for basic packet encode/decode
package fixture

import (
	"fmt"
	"net"
	"strings"

	"github.com/nickchen/packet/fixture/bgp"
)

type Mac [6]byte

func (m Mac) String() string {
	return fmt.Sprintf("%x", m[:])
}

type EtherType uint16

const (
	_IPv4 EtherType = 0x0800
	_VLAN           = 0x8100
	_IPv6           = 0x86DD
)

func (t EtherType) String() string {
	switch t {
	case _IPv4:
		return "IPv4"
	case _VLAN:
		return "VLAN"
	case _IPv6:
		return "IPv6"
	}
	return fmt.Sprintf("0x%x", int(t))
}

type EthernetII struct {
	Source Mac
	Dest   Mac
	Type   EtherType
	Body   interface{}
}

type VLAN struct {
	Priority uint8 `packet:"length=3b"`
	DEI      bool
	ID       uint16 `packet:"length=12b"`
	Type     EtherType
	Body     interface{}
}

func unmarshalBodyFromEtherType(t EtherType) interface{} {
	switch t {
	case _IPv4:
		return &IPv4{}
	case _VLAN:
		return &VLAN{}
	}
	return nil
}

// UnmarshalBody return the Body struct pointer for conversion
func (e EthernetII) UnmarshalBody() interface{} {
	return unmarshalBodyFromEtherType(e.Type)
}

func (v VLAN) UnmarshalBody() interface{} {
	return unmarshalBodyFromEtherType(v.Type)
}

type IPProtocol uint8

const (
	_TCP IPProtocol = 6
	_UDP IPProtocol = 17
)

func (p IPProtocol) String() string {
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
	return strings.Join(s, "|")
}

// IPv4 packet
type IPv4 struct {
	Version        uint8 `packet:"length=4b"`
	IHL            uint8 `packet:"length=4b"`
	DSCP           uint8 `packet:"length=6b"`
	ECN            uint8 `packet:"length=2b"`
	Length         uint16
	Id             uint16
	Flags          IPv4Flag `packet:"length=3b"`
	FragmentOffset uint16   `packet:"length=13b"`
	TTL            uint8
	Protocol       IPProtocol
	Checksum       Checksum
	Source         net.IP      `packet:"length=4B"`
	Dest           net.IP      `packet:"length=4B"`
	Options        []byte      `packet:"when=IHL-gt-5"`
	Body           interface{} `packet:rest=Length`
}

type Port uint16

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
	Options       []byte `packet:"when=DataOffset-gt-5"`
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
		return &bgp.Message{}
	}
	return nil
}
