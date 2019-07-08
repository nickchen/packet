package decode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// this is a collection of benchmark for decoder

func TestMarkUnmarshalSinglePacket(test *testing.T) {
	var i int8

	Unmarshal([]byte{0xA}, &i)
	assert.Equal(test, int8(10), i)
}

func BenchmarkUnmarshalSinglePacket(b *testing.B) {
	var i int8

	for n := 0; n < b.N; n++ {
		Unmarshal([]byte{0xA}, &i)
	}
}

type Obj struct {
	A int    `packet:"length=1B"`
	B uint   `packet:"length=1B"`
	C string `packet:"length=1B"`
	D uint   `packet:"length=1b"`
	E uint   `packet:"length=7b"`
}

func TestUnmarshalObject(test *testing.T) {
	o := &Obj{}

	Unmarshal([]byte{0xa, 0xa, 0x61, 0xFF}, o)
	assert.Equal(test, &Obj{5, 10, "a", 0x1, 0x7f}, o)
}

func BenchmarkUnmarshalObj(b *testing.B) {

	o := &Obj{}
	for n := 0; n < b.N; n++ {
		Unmarshal([]byte{0xa, 0xa, 0x61, 0xff}, o)
	}
}

type ObjWithBytesArray struct {
	A byte
	B byte
	C byte
	D [5]byte
}

var bytesWithArray = []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16}

func TestUnmarshalObjWithBytes(test *testing.T) {

	o := &ObjWithBytesArray{}
	Unmarshal(bytesWithArray, o)
	assert.Equal(test, &ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}}, o)
}

func BenchmarkUnmarshalObjWithBytes(b *testing.B) {

	o := &ObjWithBytesArray{}
	for n := 0; n < b.N; n++ {
		Unmarshal(bytesWithArray, o)
	}
}

func BenchmarkUnmarshalObjWithReader(b *testing.B) {
	o := &ObjWithBytesArray{}
	for n := 0; n < b.N; n++ {
		Unmarshal([]byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16}, o)
	}
}

type ObjWithBytesSlice struct {
	A byte
	B byte
	C byte
	D []byte `packet:"length=5B"`
}

func TestUnmarshalObjWithBytesSlice(test *testing.T) {

	o := &ObjWithBytesSlice{}
	Unmarshal(bytesWithArray, o)
	assert.Equal(test, &ObjWithBytesSlice{0xfa, 0x16, 0x3e, []byte{0x85, 0x92, 0x77, 0xfa, 0x16}}, o)
}

func BenchmarkUnmarshalObjWithBytesSlice(b *testing.B) {

	o := &ObjWithBytesSlice{}
	for n := 0; n < b.N; n++ {
		Unmarshal(bytesWithArray, o)
	}
}

func BenchmarkUnmarshalObjWithReaderSlice(b *testing.B) {
	o := &ObjWithBytesSlice{}
	for n := 0; n < b.N; n++ {
		Unmarshal([]byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16}, o)
	}
}

type ObjWithNested struct {
	A ObjWithBytesArray
	B [2]ObjWithBytesArray
}

var bytesWithArrayNested = []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16, 0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16, 0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16}

func TestUnmarshalObjWithNested(test *testing.T) {

	o := &ObjWithNested{}
	Unmarshal(bytesWithArrayNested, o)
	assert.Equal(test,
		&ObjWithNested{
			A: ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}},
			B: [2]ObjWithBytesArray{
				ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}},
				ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}},
			}}, o)
}

func BenchmarkUnmarshalObjWithNested(b *testing.B) {
	o := &ObjWithNested{}
	for n := 0; n < b.N; n++ {
		Unmarshal(bytesWithArrayNested, o)
	}
}
