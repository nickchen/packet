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

// EtherType custom function for String()
type EtherType uint16

const (
	_IPv4 EtherType = 0x0800
	_VLAN EtherType = 0x8100
	_IPv6 EtherType = 0x86DD
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

func bodyStructEtherType(t EtherType) interface{} {
	switch t {
	case _IPv4:
		return &IPv4{}
	case _VLAN:
		return &VLAN{}
	}
	return nil
}

// BodyStruct return the Body struct pointer for conversion
func (e EthernetII) BodyStruct() interface{} {
	return bodyStructEtherType(e.Type)
}

func (v VLAN) BodyStruct() interface{} {
	return bodyStructEtherType(v.Type)
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
	Body           interface{} `packet:"length_from=Length"`
}

type Port uint16

// Well know ports
const (
	_BGP Port = 179
)

// TCPFlag is 9 bits
type TCPFlag uint16

const (
	FIN TCPFlag = 1 << iota
	SYN
	RST
	PSH
	ACK
	URG
	ECE
	CWR
	NS
)

func (f TCPFlag) String() string {
	s := make([]string, 0)
	if f&NS != 0 {
		s = append(s, "NS")
	}
	if f&CWR != 0 {
		s = append(s, "CWR")
	}
	if f&ECE != 0 {
		s = append(s, "ECE")
	}
	if f&URG != 0 {
		s = append(s, "URG")
	}
	if f&ACK != 0 {
		s = append(s, "ACK")
	}
	if f&PSH != 0 {
		s = append(s, "PSH")
	}
	if f&RST != 0 {
		s = append(s, "RST")
	}
	if f&SYN != 0 {
		s = append(s, "SYN")
	}
	if f&FIN != 0 {
		s = append(s, "FIN")
	}
	return strings.Join(s, "|")
}

// TCP message
type TCP struct {
	Source        Port
	Dest          Port
	Sequence      uint32
	Ack           uint32
	DataOffset    uint8   `packet:"length=4b"`
	Flags         TCPFlag `packet:"length=12b"`
	WindowSize    uint16
	Checksum      Checksum
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

// BodyStruct return the Body struct pointer for conversion
func (tcp TCP) BodyStruct() interface{} {
	switch tcp.Dest {
	case _BGP:
		return &bgp.Message{}
	}
	return nil
}
