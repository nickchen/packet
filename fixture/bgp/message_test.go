package bgp

import (
	"fmt"
	"testing"

	"github.com/nickchen/packet"
	"github.com/stretchr/testify/assert"

	"github.com/google/go-cmp/cmp"
)

var testBGPUpdateMessage = []byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0x00, 0x3d, 0x02, 0x00, 0x00, 0x00, 0x12, 0x40, 0x01, 0x01, 0x00, 0x40, 0x02, 0x04,
	0x02, 0x01, 0xfd, 0xe8, 0x40, 0x03, 0x04, 0xc0, 0xa8, 0x56, 0x64, 0x18, 0x0a, 0x01, 0x03, 0x18,
	0x0a, 0x01, 0x06, 0x18, 0x0a, 0x01, 0x07, 0x18, 0x0a, 0x01, 0x04, 0x18, 0x0a, 0x01, 0x05,
}

// testBGPKeepaliveMessage single BGP Keepalive
var testBGPKeepaliveMessage = []byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0x00, 0x13, 0x04,
}

// testBGPComboMessage has two UPDATE message
var testBGPComboMessage = []byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0x00, 0x13, 0x04, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0x00, 0x62, 0x02, 0x00, 0x00, 0x00, 0x48, 0x40, 0x01, 0x01, 0x02, 0x40, 0x02,
	0x0a, 0x01, 0x02, 0x01, 0xf4, 0x01, 0xf4, 0x02, 0x01, 0xfe, 0xbb, 0x40, 0x03, 0x04, 0xc0, 0xa8,
	0x00, 0x0f, 0x40, 0x05, 0x04, 0x00, 0x00, 0x00, 0x64, 0x40, 0x06, 0x00, 0xc0, 0x07, 0x06, 0xfe,
	0xba, 0xc0, 0xa8, 0x00, 0x0a, 0xc0, 0x08, 0x0c, 0xfe, 0xbf, 0x00, 0x01, 0x03, 0x16, 0x00, 0x04,
	0x01, 0x54, 0x00, 0xfa, 0x80, 0x09, 0x04, 0xc0, 0xa8, 0x00, 0x0f, 0x80, 0x0a, 0x04, 0xc0, 0xa8,
	0x00, 0xfa, 0x10, 0xac, 0x10, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x63, 0x02, 0x00, 0x00, 0x00, 0x48, 0x40, 0x01, 0x01, 0x00,
	0x40, 0x02, 0x0a, 0x01, 0x02, 0x01, 0xf4, 0x01, 0xf4, 0x02, 0x01, 0xfe, 0xbb, 0x40, 0x03, 0x04,
	0xc0, 0xa8, 0x00, 0x0f, 0x40, 0x05, 0x04, 0x00, 0x00, 0x00, 0x64, 0x40, 0x06, 0x00, 0xc0, 0x07,
	0x06, 0xfe, 0xba, 0xc0, 0xa8, 0x00, 0x0a, 0xc0, 0x08, 0x0c, 0xfe, 0xbf, 0x00, 0x01, 0x03, 0x16,
	0x00, 0x04, 0x01, 0x54, 0x00, 0xfa, 0x80, 0x09, 0x04, 0xc0, 0xa8, 0x00, 0x0f, 0x80, 0x0a, 0x04,
	0xc0, 0xa8, 0x00, 0xfa, 0x16, 0xc0, 0xa8, 0x04}

func printDetailErrorInformation(err error) {
	switch et := err.(type) {
	case *packet.UnmarshalUnexpectedEnd:
		fmt.Printf("Offset: %d End: %d\n", et.Offset, et.End)
	}
}

func checkBGP(t *testing.T, want interface{}, packetBytes []byte, MessageType MessageType) {
	switch want.(type) {
	case *Message:
		bgp := &Message{}
		err := packet.Unmarshal(packetBytes, bgp)
		if err != nil {
			t.Error("Failed to decode packet:", err)
			printDetailErrorInformation(err)
		}
		assert.Equal(t, bgp.Type, MessageType, "message type not equal")
		difference := cmp.Diff(want.(*Message), bgp)
		assert.Empty(t, difference, "diff found")
	case *[]Message:
		bgps := &[]Message{}
		err := packet.Unmarshal(packetBytes, bgps)
		if err != nil {
			t.Error("Failed to decode packet:", err)
			printDetailErrorInformation(err)
		}
		fmt.Printf("BGP: %+v\n", bgps)
		difference := cmp.Diff(want.(*[]Message), bgps)
		assert.Empty(t, difference, "diff found")
	default:
		assert.Fail(t, "unknown type")
	}
}

func TestBGPKeepaliveMessage(t *testing.T) {
	want := &Message{
		Marker: _16ByteMaker,
		Type:   _Keepalive,
		Length: 19,
		Body:   &Keepalive{},
	}
	checkBGP(t, want, testBGPKeepaliveMessage, _Keepalive)
}

func TestBGPUpdateMessage(t *testing.T) {
	want := &Message{
		Marker: _16ByteMaker,
		Type:   _Update,
		Length: 61,
		Body: &Update{
			WithdrawnLength:     0,
			PathAttributeLength: 18,
			PathAttributes: []PathAttribute{
				PathAttribute{
					Flags:  Transitive,
					Code:   Origin,
					Length: 1,
					Data:   &OriginAttribute{Origin: IBGP},
				},
				PathAttribute{
					Flags:  Transitive,
					Code:   AsPath,
					Length: 4,
					Data: &[]AsPathAttribute{
						AsPathAttribute{Type: AsSequence, Count: 1, List: []ASN{ASN(65000)}},
					},
				},
				PathAttribute{
					Flags:  Transitive,
					Code:   Nexthop,
					Length: 4,
					Data:   &NexthopAttribute{Nexthop: []byte{0xc0, 0xa8, 0x56, 0x64}},
				},
			},
			NLRI: []PrefixSpec{
				PrefixSpec{
					Length: 24,
					Prefix: []byte{0x0a, 0x01, 0x03},
				},
				PrefixSpec{
					Length: 24,
					Prefix: []byte{0x0a, 0x01, 0x06},
				},
				PrefixSpec{
					Length: 24,
					Prefix: []byte{0x0a, 0x01, 0x07},
				},
				PrefixSpec{
					Length: 24,
					Prefix: []byte{0x0a, 0x01, 0x04},
				},
				PrefixSpec{
					Length: 24,
					Prefix: []byte{0x0a, 0x01, 0x05},
				},
			},
		},
	}
	checkBGP(t, want, testBGPUpdateMessage, _Update)
}

func TestBGPComboPacket(t *testing.T) {
	wants := &[]Message{
		Message{
			Marker: _16ByteMaker,
			Type:   _Keepalive,
			Length: 19,
			Body:   &Keepalive{},
		},
		Message{
			Marker: _16ByteMaker,
			Type:   _Update,
			Length: 98,
			Body: &Update{
				WithdrawnLength:     0,
				PathAttributeLength: 72,
				PathAttributes: []PathAttribute{
					PathAttribute{
						Flags:  Transitive,
						Code:   Origin,
						Length: 1,
						Data:   &OriginAttribute{Origin: INCOMPLETE},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   AsPath,
						Length: 10,
						Data: &[]AsPathAttribute{
							AsPathAttribute{Type: AsSet, Count: 2, List: []ASN{ASN(500), ASN(500)}},
							AsPathAttribute{Type: AsSequence, Count: 1, List: []ASN{ASN(65211)}},
						},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   Nexthop,
						Length: 4,
						Data:   &NexthopAttribute{Nexthop: []byte{0xc0, 0xa8, 0x00, 0x0f}},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   LocalPref,
						Length: 4,
						Data:   &LocalPrefAttribute{LocalPref: 100},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   AtomicAggregate,
						Length: 0,
					},
					PathAttribute{
						Flags:  Transitive | Optional,
						Code:   Aggregator,
						Length: 6,
						Data:   &AggregatorAttribute{AS: 65210, Origin: []byte{0xc0, 0xa8, 0x00, 0x0a}},
					},
					PathAttribute{
						Flags:  Transitive | Optional,
						Code:   Community,
						Length: 12,
						Data: &[]CommunityAttribute{
							CommunityAttribute{
								Attribute: uint32((65215 << 16) | (1)),
							},
							CommunityAttribute{
								Attribute: uint32((790 << 16) | (4)),
							},
							CommunityAttribute{
								Attribute: uint32((340 << 16) | (250)),
							},
						},
					},
					PathAttribute{
						Flags:  Optional,
						Code:   OriginatorID,
						Length: 4,
						Data:   &[]byte{0xc0, 0xa8, 0x00, 0x0f},
					},
					PathAttribute{
						Flags:  Optional,
						Code:   ClusterList,
						Length: 4,
						Data:   &[]byte{0xc0, 0xa8, 0x00, 0xfa},
					},
				},
				NLRI: []PrefixSpec{
					PrefixSpec{
						Length: 16,
						Prefix: []byte{0xac, 0x10},
					},
				},
			},
		},
		Message{
			Marker: _16ByteMaker,
			Type:   _Update,
			Length: 99,
			Body: &Update{
				WithdrawnLength:     0,
				PathAttributeLength: 72,
				PathAttributes: []PathAttribute{
					PathAttribute{
						Flags:  Transitive,
						Code:   Origin,
						Length: 1,
						Data:   &OriginAttribute{Origin: IBGP},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   AsPath,
						Length: 10,
						Data: &[]AsPathAttribute{
							AsPathAttribute{Type: AsSet, Count: 2, List: []ASN{ASN(500), ASN(500)}},
							AsPathAttribute{Type: AsSequence, Count: 1, List: []ASN{ASN(65211)}},
						},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   Nexthop,
						Length: 4,
						Data:   &NexthopAttribute{Nexthop: []byte{0xc0, 0xa8, 0x00, 0x0f}},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   LocalPref,
						Length: 4,
						Data:   &LocalPrefAttribute{LocalPref: 100},
					},
					PathAttribute{
						Flags:  Transitive,
						Code:   AtomicAggregate,
						Length: 0,
					},
					PathAttribute{
						Flags:  Transitive | Optional,
						Code:   Aggregator,
						Length: 6,
						Data:   &AggregatorAttribute{AS: 65210, Origin: []byte{0xc0, 0xa8, 0x00, 0x0a}},
					},
					PathAttribute{
						Flags:  Transitive | Optional,
						Code:   Community,
						Length: 12,
						Data: &[]CommunityAttribute{
							CommunityAttribute{
								Attribute: uint32((65215 << 16) | (1)),
							},
							CommunityAttribute{
								Attribute: uint32((790 << 16) | (4)),
							},
							CommunityAttribute{
								Attribute: uint32((340 << 16) | (250)),
							},
						},
					},
					PathAttribute{
						Flags:  Optional,
						Code:   OriginatorID,
						Length: 4,
						Data:   &[]byte{0xc0, 0xa8, 0x00, 0x0f},
					},
					PathAttribute{
						Flags:  Optional,
						Code:   ClusterList,
						Length: 4,
						Data:   &[]byte{0xc0, 0xa8, 0x00, 0xfa},
					},
				},
				NLRI: []PrefixSpec{
					PrefixSpec{
						Length: 22,
						Prefix: []byte{0xc0, 0xa8, 0x04},
					},
				},
			},
		},
	}
	checkBGP(t, wants, testBGPComboMessage, _Update)
}
