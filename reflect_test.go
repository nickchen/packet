package packet

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// this is a collection of benchmark for refect Set*

func setSingleValue(i *int) {
	t := reflect.ValueOf(i)
	if t.Kind() == reflect.Ptr {
		v := t.Elem()
		v.SetInt(10)
	}
}

func TestMarkUnmarshalSingle(test *testing.T) {
	var i int

	setSingleValue(&i)
	assert.Equal(test, 10, i)
}

func BenchmarkUnmarshalSingle(b *testing.B) {
	var i int

	for n := 0; n < b.N; n++ {
		setSingleValue(&i)
	}
}

type Obj struct {
	A int
	B uint
	C string
}

func setObjValue(i *Obj) {
	t := reflect.ValueOf(i)
	if t.Kind() == reflect.Ptr {
		v := t.Elem()
		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				f := v.Field(i)
				switch f.Kind() {
				case reflect.Int:
					f.SetInt(10)
				case reflect.String:
					f.SetString("10")
				case reflect.Uint:
					f.SetUint(10)
				}
			}
		}
	}
}

func TestUnmarshalObject(test *testing.T) {
	o := &Obj{}

	setObjValue(o)
	assert.Equal(test, &Obj{10, 10, "10"}, o)
}

func BenchmarkUnmarshalObj(b *testing.B) {

	o := &Obj{}
	for n := 0; n < b.N; n++ {
		setObjValue(o)
	}
}

type ObjWithBytesArray struct {
	A byte
	B byte
	C byte
	D [5]byte
}

func setObjWithBytesValue(o interface{}, data []byte) {
	t := reflect.ValueOf(o)
	if t.Kind() == reflect.Ptr {
		v := t.Elem()
		if v.Kind() == reflect.Struct {
			k := 0
			for i := 0; i < v.NumField(); i++ {
				f := v.Field(i)
				switch f.Kind() {
				case reflect.Uint8:
					f.SetUint(uint64(data[k]))
					k++
				case reflect.Array:
					if f.Cap() > 0 && f.Index(0).Kind() == reflect.Uint8 {
						for j := 0; j < f.Cap(); j++ {
							fv := f.Index(j)
							fv.SetUint(uint64(data[k]))
							k++
						}
						k += f.Cap()
					}
				case reflect.Slice:
					// slink the rest of the array
					newCap := len(data) - k
					newf := reflect.MakeSlice(f.Type(), f.Len(), newCap)
					reflect.Copy(newf, f)
					f.Set(newf)
					f.SetBytes(data[k:len(data)])
				}
			}
		}
	} else {
		panic(fmt.Errorf("can not handle type %+v", t))
	}

}

var objectWithBytes = []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16}

func TestUnmarshalObjWithBytes(test *testing.T) {

	o := &ObjWithBytesArray{}
	setObjWithBytesValue(o, objectWithBytes)
	assert.Equal(test, &ObjWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}}, o)
}

func BenchmarkUnmarshalObjWithBytes(b *testing.B) {

	o := &ObjWithBytesArray{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}

func BenchmarkUnmarshalObjWithReader(b *testing.B) {
	o := &ObjWithBytesArray{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}

type ObjWithBytesSlice struct {
	A byte
	B byte
	C byte
	D []byte
}

func TestUnmarshalObjWithBytesSlice(test *testing.T) {

	o := &ObjWithBytesSlice{}
	setObjWithBytesValue(o, objectWithBytes)
	assert.Equal(test, &ObjWithBytesSlice{0xfa, 0x16, 0x3e, []byte{0x85, 0x92, 0x77, 0xfa, 0x16}}, o)
}

func BenchmarkUnmarshalObjWithBytesSlice(b *testing.B) {

	o := &ObjWithBytesSlice{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}

func BenchmarkUnmarshalObjWithReaderSlice(b *testing.B) {
	o := &ObjWithBytesSlice{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}
