package packet

import (
	"bytes"
	"fmt"
	"reflect"
)

type encoder struct {
	bytes.Buffer
	scratch [64]byte
	current uint64
	bits    struct {
		data   uint64
		length uint64
	}
}

// Marshal encode object into binary bytes
func Marshal(v interface{}) ([]byte, error) {
	e := new(encoder)
	rv := reflect.ValueOf(v)
	err := e.encode(rv, nil)
	return e.Bytes(), err
}

func (e *encoder) encode(v reflect.Value, f *field) error {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Ptr:
		pv := v.Elem()
		return e.encode(pv, f)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e._primitives(v, v, f)
	case reflect.String:
		_, err := e.Write([]byte(v.String()))
		return err
	case reflect.Struct:
		return e._struct(v)
	case reflect.Slice, reflect.Array:
		for j := 0; j < v.Len(); j++ {
			vf := v.Index(j)
			if vf.CanSet() {
				if err := e.encode(vf, f); err != nil {
					return err
				}
			}
		}
	case reflect.Interface:
		return e.encode(v.Elem(), nil)
	default:
		panic(fmt.Errorf("not handled type %v", v.Type()))
	}
	return nil
}

func (e *encoder) fieldEncode(parent reflect.Value, v reflect.Value, f *field) error {
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e._primitives(parent, v, f)
	}
	return e.encode(v, f)
}

func (e *encoder) writeBits() error {
	for ; e.bits.length >= 8; e.bits.length -= 8 {
		e.scratch[e.current] = uint8(e.bits.data >> (e.bits.length - 8))
		e.current++
	}
	e.Write(e.scratch[0:e.current])
	e.current = 0
	return nil
}

func (e *encoder) encodeBitFieldValue(v reflect.Value, u unit, length uint64) error {
	switch u {
	case _bits:
		switch v.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			mask := makeMask(uint(length))
			value := v.Uint()
			e.bits.data <<= length
			e.bits.data |= (mask & value)
			e.bits.length += length
			return e.writeBits()
		}

	case _byte:
		switch v.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			rlength := length * 8
			mask := makeMask(uint(rlength + e.bits.length))
			value := v.Uint()
			e.bits.data <<= rlength
			e.bits.data |= (mask & value)
			e.bits.length += rlength
			return e.writeBits()
		}
	}
	return nil
}

func (e *encoder) _primitives(parent reflect.Value, v reflect.Value, f *field) error {
	length := uint64(0)
	if f != nil {
		switch {
		case f.length != nil:
			return e.encodeBitFieldValue(v, f.length.unit, f.length.length)
		case f.f.lengthfor:
			if m, ok := parent.Interface().(LengthFor); ok {
				length = m.LengthFor(f.Name)
				goto encode
			}
		}
	}
	switch v.Kind() {
	case reflect.Uint8:
		length = 1
	case reflect.Uint16:
		length = 2
	case reflect.Uint32:
		length = 4
	case reflect.Uint:
		length = 4
	case reflect.Uint64:
		length = 8
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		panic(fmt.Errorf("Int packet encoding not supported."))
	}
encode:
	return e.encodeBitFieldValue(v, _byte, length)
}

func (e *encoder) _struct(v reflect.Value) error {
	vf := getStructFields(v)
	for i := 0; i < len(*vf); i++ {
		if err := e.fieldEncode(v, v.Field(i), (*vf)[i]); err != nil {
			return err
		}
	}
	return nil
}
