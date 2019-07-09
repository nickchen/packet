package packet

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// this is a collection of benchmark for refect Set*

func setSingleValueReflect(i *int) {
	t := reflect.ValueOf(i)
	if t.Kind() == reflect.Ptr {
		v := t.Elem()
		v.SetInt(10)
	}
}

func TestMarkUnmarshalSingle(test *testing.T) {
	var i int

	setSingleValueReflect(&i)
	assert.Equal(test, 10, i)
}

func BenchmarkUnmarshalSingle(b *testing.B) {
	var i int

	for n := 0; n < b.N; n++ {
		setSingleValueReflect(&i)
	}
}

type ObjReflect struct {
	A int
	B uint
	C string
}

func setObjValue(i *ObjReflect) {
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

func TestUnmarshalObjectReflect(test *testing.T) {
	o := &ObjReflect{}

	setObjValue(o)
	assert.Equal(test, &ObjReflect{10, 10, "10"}, o)
}

func BenchmarkUnmarshalObjReflect(b *testing.B) {

	o := &ObjReflect{}
	for n := 0; n < b.N; n++ {
		setObjValue(o)
	}
}

type ObjReflectWithBytesArray struct {
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
					f.SetBytes(data[k:])
				}
			}
		}
	} else {
		panic(fmt.Errorf("can not handle type %+v", t))
	}

}

var objectReflectWithBytes = []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16}

func TestReflectUnmarshalObjWithBytes(test *testing.T) {

	o := &ObjReflectWithBytesArray{}
	setObjWithBytesValue(o, objectReflectWithBytes)
	assert.Equal(test, &ObjReflectWithBytesArray{0xfa, 0x16, 0x3e, [5]byte{0x85, 0x92, 0x77, 0xfa, 0x16}}, o)
}

func BenchmarkReflectUnmarshalObjWithBytes(b *testing.B) {

	o := &ObjReflectWithBytesArray{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}

func BenchmarkReflectUnmarshalObjWithReader(b *testing.B) {
	o := &ObjWithBytesArray{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}

type ObjReflectWithBytesSlice struct {
	A byte
	B byte
	C byte
	D []byte
}

func TestReflectUnmarshalObjWithBytesSlice(test *testing.T) {

	o := &ObjWithBytesSlice{}
	setObjWithBytesValue(o, objectReflectWithBytes)
	assert.Equal(test, &ObjWithBytesSlice{0xfa, 0x16, 0x3e, []byte{0x85, 0x92, 0x77, 0xfa, 0x16}}, o)
}

func BenchmarkReflectUnmarshalObjWithBytesSlice(b *testing.B) {

	o := &ObjWithBytesSlice{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}

func BenchmarkReflectUnmarshalObjWithReaderSlice(b *testing.B) {
	o := &ObjWithBytesSlice{}
	for n := 0; n < b.N; n++ {
		setObjWithBytesValue(o, []byte{0xfa, 0x16, 0x3e, 0x85, 0x92, 0x77, 0xfa, 0x16})
	}
}
