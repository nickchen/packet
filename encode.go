package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
)

type encoder struct {
	bytes.Buffer
	scratch [64]byte
}

// Marshal encode object into binary bytes
func Marshal(v interface{}) ([]byte, error) {
	e := new(encoder)
	rv := reflect.ValueOf(v)
	err := e.encode(rv)
	return e.Bytes(), err
}

func (e *encoder) encode(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Ptr:
		pv := v.Elem()
		return e.encode(pv)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e._primitives(v)
	case reflect.String:
		_, err := e.Write([]byte(v.String()))
		return err
	case reflect.Struct:
		return e._struct(v)
	default:
		panic(fmt.Errorf("not handled type %v", v.Type()))
	}
	return nil
}

func (e *encoder) _primitives(v reflect.Value) error {
	length := int(0)
	switch v.Kind() {
	case reflect.Uint8:
		return e.WriteByte(v.Interface().(uint8))
	case reflect.Uint16:
		length = 2
		binary.BigEndian.PutUint16(e.scratch[0:length], v.Interface().(uint16))
	case reflect.Uint32:
		length = 4
		binary.BigEndian.PutUint32(e.scratch[0:length], v.Interface().(uint32))
	case reflect.Uint:
		length = 4
		binary.BigEndian.PutUint32(e.scratch[0:length], uint32(v.Interface().(uint)))
	case reflect.Uint64:
		length = 8
		binary.BigEndian.PutUint64(e.scratch[0:length], v.Interface().(uint64))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		length = binary.PutVarint(e.scratch[0:], v.Int())
	}

	if l, err := e.Write(e.scratch[0:length]); err != nil {
		return err
	} else if l != length {
		return fmt.Errorf("write unexpected bytes wants %d, got %d", length, l)
	}
	return nil
}

func (e *encoder) _struct(v reflect.Value) error {
	vf := getStructFields(v)
	for i := 0; i < len(*vf); i++ {
		if err := e.encode(v.Field(i)); err != nil {
			return err
		}
	}
	return nil
}
