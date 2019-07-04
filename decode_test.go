package packet

import (
	"fmt"
	"testing"

	"github.com/nickchen/packet/fixture"
	"github.com/nickchen/packet/fixture/bgp"
	"github.com/stretchr/testify/assert"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var frame = []byte{
	0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16, /* ..>..w.. */
	0x3e, 0x1a, 0x43, 0xcb, 0x81, 0x00, 0x0f, 0xfe, /* >.C..... */
	0x08, 0x00, 0x45, 0x00, 0x00, 0x6b, 0x9a, 0xaf, /* ..E..k.. */
	0x40, 0x00, 0x01, 0x06, 0xca, 0xa2, 0x0a, 0x14, /* @....... */
	0x00, 0x0a, 0x0a, 0x0a, 0x00, 0x14, 0x89, 0xce, /* ........ */
	0x00, 0xb3, 0x48, 0x0c, 0x55, 0x19, 0x8b, 0xd2, /* ..H.U... */
	0x47, 0x96, 0x80, 0x18, 0x00, 0x73, 0xfc, 0x5c, /* G....s.\ */
	0x00, 0x00, 0x01, 0x01, 0x08, 0x0a, 0x80, 0x02, /* ........ */
	0x3c, 0xbe, 0x00, 0x0a, 0xf2, 0x19, 0xff, 0xff, /* <....... */
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, /* ........ */
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x37, /* .......7 */
	0x01, 0x04, 0xfd, 0xea, 0x00, 0x5a, 0x0a, 0x28, /* .....Z.( */
	0x00, 0x0a, 0x1a, 0x02, 0x06, 0x01, 0x04, 0x00, /* ........ */
	0x01, 0x00, 0x01, 0x02, 0x02, 0x80, 0x00, 0x02, /* ........ */
	0x02, 0x02, 0x00, 0x02, 0x08, 0x40, 0x06, 0x00, /* .....@.. */
	0x78, 0x00, 0x01, 0x01, 0x00, 0xf5, 0xde, 0xb0, /* x....... */
	0xf5, 0x00, 0x14, 0x00, 0x01, 0x00, 0x01, 0x00, /* ........ */
	0x01, 0x00, 0x0c, 0x00, 0x02, 0x01, 0x00, 0x00, /* ........ */
	0x00, /* . */
}

func TestFixture(t *testing.T) {
	ether := &fixture.EthernetII{}

	err := Unmarshal(frame, ether)
	assert.NoError(t, err, "failed to decode etherframe")
	fmt.Printf("ether %+v\n", ether)

	vlan, ok := ether.Body.(*fixture.VLAN)
	assert.True(t, ok, "failed to find vlan")
	fmt.Printf("vlan %+v\n", vlan)

	ipv4, ok := vlan.Body.(*fixture.IPv4)
	assert.True(t, ok, "failed to find ipv4")
	fmt.Printf("ipv4 %+v\n", ipv4)

	tcp, ok := ipv4.Body.(*fixture.TCP)
	assert.True(t, ok, "failed to find TCP")
	fmt.Printf("tcp %+v\n", tcp)

	bgp, ok := tcp.Body.(*bgp.Message)
	assert.True(t, ok, "failed to find BGP")
	assert.NotNil(t, bgp, "bgp is null")
}
func TestCompare(t *testing.T) {
	// Decode a packet
	gp := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)

	ip := &fixture.IPv4{}
	err := Unmarshal(frame[18:], ip)
	assert.NoError(t, err, "failed to unmarshal packet")

	// expectedIP := fixture.IPv4{Version: 4, IHL: 5, Protocol: 6}
	// if !assert.ObjectsAreEqual(expectedIP, ip) {
	// 	fmt.Printf("object: *(%v)* *(%v)*", expectedIP, ip)
	// }
	gpIPLayer := gp.Layer(layers.LayerTypeIPv4)
	assert.NotNil(t, gpIPLayer, "tcp layer decoded")
	gpIP, _ := gpIPLayer.(*layers.IPv4)
	assert.True(t, IPPacketEqual(t, ip, gpIP))

	assert.NotNil(t, ip.Body, "body should be populated with TCP")
	tcp, ok := ip.Body.(*fixture.TCP)
	assert.True(t, ok, "tcp message from ip")
	assert.NotNil(t, tcp, "tcp body not found")
	fmt.Printf("TCP: %+v\n", tcp)

	gpTCPLayer := gp.Layer(layers.LayerTypeTCP)
	assert.NotNil(t, gpTCPLayer, "tcp layer decoded")
	gpTCP, _ := gpTCPLayer.(*layers.TCP)
	fmt.Printf("GP TCP: %+v\n", gpTCP)
	assert.NotNil(t, nil, "failure is expected")
}

func IPPacketEqual(t *testing.T, ip *fixture.IPv4, gp *layers.IPv4) bool {
	assert.Equal(t, gp.Version, ip.Version)
	assert.Equal(t, gp.IHL, ip.IHL)
	assert.Equal(t, gp.Length, ip.Length)
	assert.Equal(t, gp.Checksum, uint16(ip.Checksum))
	fmt.Printf("object: *(%v)* *(%v)*\n", ip, gp)
	return true
}

/*
Running tool: /usr/local/go/bin/go test -benchmem -run=^$ github.com/nickchen/packet -bench ^(BenchmarkGoPacket)$

goos: darwin
goarch: amd64
pkg: github.com/nickchen/packet
BenchmarkGoPacket-8   	 1000000	      1035 ns/op	    1240 B/op	      12 allocs/op
BenchmarkGoPacket-12    	 2000000	       877 ns/op	    1240 B/op	      12 allocs/op
PASS
ok  	github.com/nickchen/packet	1.396s
Success: Benchmarks passed.
*/
func BenchmarkGoPacket(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_ = gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)
	}
}

/*
for Etherframe -> VLAN -> IP -> TCP -> BGP -> Open
Running tool: /usr/local/go/bin/go test -benchmem -run=^$ github.com/nickchen/packet -bench ^(BenchmarkPacket)$

goos: darwin
goarch: amd64
pkg: github.com/nickchen/packet
BenchmarkPacket-12    	 1000000	      2168 ns/op	     624 B/op	      12 allocs/op
BenchmarkPacket-12    	 1000000	      1946 ns/op	     592 B/op	      12 allocs/op
PASS
ok  	github.com/nickchen/packet	2.742s
Success: Benchmarks passed.
*/
func BenchmarkPacket(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		ether := &fixture.EthernetII{}
		_ = Unmarshal(frame, ether)
	}
}

func TestReadPCAP(t *testing.T) {
	pcap, err := fixture.OpenPCAP("fixture/NTLM-wenchao.pcap")
	assert.NoError(t, err, "failed to open pcap")
	count := 0
	for p := range pcap.PacketData() {
		fmt.Printf("=====\n")
		ether := &fixture.EthernetII{}

		err = Unmarshal(p, ether)
		assert.NoError(t, err, "failed to decode")
		fmt.Printf("Packet: %+v\n", ether)
		ip, _ := ether.Body.(*fixture.IPv4)
		assert.NotNil(t, ip, "ether->ip")
		fmt.Printf("IP: %+v\n", ip)

		tcp, _ := ip.Body.(*fixture.TCP)
		assert.NotNil(t, tcp, "ip->tcp")
		fmt.Printf("TCP: %s\n", string(tcp.Body.([]byte)))
		count++
		if count >= 5 {
			break
		}
	}
	assert.NotNil(t, nil, "failure is expected")
}
