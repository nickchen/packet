package fixture

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// PacketData raw bytes for the packet
type PacketData []byte

// Pcap helps with reading packet from pcap format
type Pcap struct {
	source  *gopacket.PacketSource
	channel chan PacketData
}

// OpenPCAP read file as gopacket.PacketSource
func OpenPCAP(file string) (*Pcap, error) {
	handle, err := pcap.OpenOffline(file)
	if err != nil {
		return nil, err
	}
	return &Pcap{source: gopacket.NewPacketSource(handle, handle.LinkType())}, nil
}

// PacketData use channel so it's possible to do range
func (p *Pcap) PacketData() chan PacketData {
	if p.channel == nil {
		p.channel = make(chan PacketData, 5)
		go func() {
			for gp := range p.source.Packets() {
				p.channel <- PacketData(gp.Data())
			}
		}()
	}
	return p.channel
}
