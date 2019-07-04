package packet

import (
	"fmt"
	"testing"
	"strconv"
	"github.com/stretchr/testify/assert"
)

func Test_makeMask8(t *testing.T) {
	var uint82mask = map[uint64]uint8{
		8: 0xff,
		7: 0x7f,
		6: 0x3f,
		5: 0x1f,
		4: 0x0f,
		3: 0x07,
		2: 0x03,
		1: 0x01,
		0: 0x00,
	}
	lengthDiff := uint64(7 * 8)
	for length, expected := range uint82mask {
		v := makeMask(length + lengthDiff)
		fmt.Printf("0x%s\n", strconv.FormatUint(v, 16))
		vuint8 := uint8(v >> lengthDiff)
		assert.Equal(t, expected, vuint8, fmt.Sprintf("unexpected mask for length %d", length))
	}
}
func Test_makeMask16(t *testing.T) {
	var uint162mask = map[uint64]uint16{
		16: 0xffff,
		15: 0x7fff,
		14: 0x3fff,
		13: 0x1fff,
		12: 0x0fff,
		11: 0x07ff,
		10: 0x03ff,
		9:  0x01ff,
		8:  0x00ff,
		7:  0x007f,
		6:  0x003f,
		5:  0x001f,
		4:  0x000f,
		3:  0x0007,
		2:  0x0003,
		1:  0x0001,
		0:  0x0000,
	}
	lengthDiff := uint64(6 * 8)
	for length, expected := range uint162mask {
		v := makeMask(length + lengthDiff)
		vuint16 := uint16(v >> lengthDiff)

		assert.Equal(t, expected, vuint16, fmt.Sprintf("unexpected mask for length %d", length))
	}
}
