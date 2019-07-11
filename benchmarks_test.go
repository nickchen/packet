package packet

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
	A uint8  `packet:"length=1B"`
	B uint8  `packet:"length=1B"`
	C string `packet:"length=1B"`
	D uint8  `packet:"length=1b"`
	E uint8  `packet:"length=7b"`
}

var bytesForObj = []byte{0xa, 0xb, 0x61, 0xff}
var obj = &Obj{10, 11, "a", 0x1, 0x7f}

func TestUnmarshalObject(test *testing.T) {
	o := &Obj{}

	Unmarshal(bytesForObj, o)
	assert.Equal(test, obj, o)
}

func BenchmarkUnmarshalObj(b *testing.B) {

	o := &Obj{}
	for n := 0; n < b.N; n++ {
		Unmarshal(bytesForObj, o)
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

type ObjWithBytesArrayUnexported struct {
	A  byte
	b  byte
	BB byte
	C  byte
	D  [5]byte
	e  byte
	f  byte
}

func TestUnmarshalObjWithBytesUnexported(test *testing.T) {

	o := &ObjWithBytesArrayUnexported{}
	Unmarshal(bytesWithArray, o)
	assert.Equal(test, &ObjWithBytesArrayUnexported{A: 0xfa, BB: 0x16, C: 0x3e, D: [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}}, o)
}

func BenchmarkUnmarshalObjWithBytesUnexported(b *testing.B) {

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

var bytesWithArrayNested = []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16, 0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x17, 0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x18}

func TestUnmarshalObjWithNested(test *testing.T) {

	o := &ObjWithNested{}
	Unmarshal(bytesWithArrayNested, o)
	assert.Equal(test,
		&ObjWithNested{
			A: ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}},
			B: [2]ObjWithBytesArray{
				ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x17}},
				ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x18}},
			}}, o)
}

func BenchmarkUnmarshalObjWithNested(b *testing.B) {
	o := &ObjWithNested{}
	for n := 0; n < b.N; n++ {
		Unmarshal(bytesWithArrayNested, o)
	}
}

type ObjWithNestedPointer struct {
	A *ObjWithBytesArray
	B [2]ObjWithBytesArray
}

func TestUnmarshalObjWithNestedPointer(test *testing.T) {

	o := &ObjWithNestedPointer{}
	assert.NoError(test, Unmarshal(bytesWithArrayNested, o))
	assert.Equal(test,
		&ObjWithNestedPointer{
			A: &ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}},
			B: [2]ObjWithBytesArray{
				ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x17}},
				ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x18}},
			}}, o)
}

// BenchmarkUnmarshalObjWithPointer will allocate the pointer instance
func BenchmarkUnmarshalObjWithPointer(b *testing.B) {
	o := &ObjWithNestedPointer{}
	for n := 0; n < b.N; n++ {
		Unmarshal(bytesWithArrayNested, o)
	}
}

type ObjWithNestedSlice struct {
	A *[]ObjWithBytesArray
}

var objWithNestedSlice = &ObjWithNestedSlice{
	A: &[]ObjWithBytesArray{
		ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}},
		ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x17}},
		ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x18}},
	}}

func TestUnmarshalObjWithSlice(test *testing.T) {

	o := &ObjWithNestedSlice{}
	assert.NoError(test, Unmarshal(bytesWithArrayNested, o))
	assert.Equal(test, objWithNestedSlice, o)
}

func BenchmarkUnmarshalObjWithSlice(b *testing.B) {
	o := &ObjWithNestedSlice{}
	for n := 0; n < b.N; n++ {
		Unmarshal(bytesWithArrayNested, o)
	}
}

func TestMarhshalByte(test *testing.T) {
	b, err := Marshal(uint8(100))
	assert.NoError(test, err, "marshall successfully")
	assert.Equal(test, []byte{0x64}, b, "marshall correctly")
}

func TestMarhshalUint16(test *testing.T) {
	b, err := Marshal(uint16(100))
	assert.NoError(test, err, "marshall successfully")
	assert.Equal(test, []byte{0x0, 0x64}, b, "marshall correctly")
}

func TestMarhshalUint32(test *testing.T) {
	b, err := Marshal(uint32(100))
	assert.NoError(test, err, "marshall successfully")
	assert.Equal(test, []byte{0x0, 0x0, 0x0, 0x64}, b, "marshall correctly")
}

func TestMarhshalUint64(test *testing.T) {
	b, err := Marshal(uint64(100))
	assert.NoError(test, err, "marshall successfully")
	assert.Equal(test, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64}, b, "marshall correctly")
}

func TestMarhshalObj(test *testing.T) {
	b, err := Marshal(obj)
	assert.NoError(test, err, "marshall successfully")
	assert.Equal(test, bytesForObj, b, "marshall correctly")
}

func BenchmarkMarhshalObj(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = Marshal(obj)
	}
}

func TestMarhshalObjWithNestedSlice(test *testing.T) {
	b, err := Marshal(objWithNestedSlice)
	assert.NoError(test, err, "marshall successfully")
	assert.Equal(test, bytesWithArrayNested, b, "marshall correctly")
}

func BenchmarkMarhshalObjWithNestedSlice(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = Marshal(objWithNestedSlice)
	}
}
