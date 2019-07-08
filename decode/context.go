package decode

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type cursor struct {
	start   uint64
	end     uint64
	current uint64
}

const _maxCursors = 16

type decoder struct {
	data     []byte
	cursor   [_maxCursors]cursor
	currentC int
	bits     struct {
		data   uint64
		length uint
	}
}

// Unmarshal parson the packet data and stores the result in value pointed by v.
// If v is nil or not a pointer, Unmarshal returns an InvalidUnmarshalError.
func Unmarshal(data []byte, v interface{}) error {
	d := &decoder{data: data, currentC: 0}
	c := &d.cursor[d.currentC]
	c.start = 0
	c.current = 0
	c.end = uint64(len(d.data))
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &UnmarshalPtrError{reflect.TypeOf(v)}
	}

	return d._ptr(c, rv.Elem())
}

func (d *decoder) _ptr(c *cursor, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Int8:
		v.SetInt(int64(d.data[c.current]))
	case reflect.Struct:
		return d._struct(c, v)
	}
	return nil
}

func (d *decoder) _struct(c *cursor, v reflect.Value) error {
	vf := getStructFields(v)
	for i := 0; i < len(*vf); i++ {
		if err := d.setFieldValue(c, (*vf)[i], v, v.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func (d *decoder) getBitsByLength(c *cursor, length uint) uint64 {
	for d.bits.length < length {
		d.bits.data <<= 8
		d.bits.data |= uint64(d.data[c.current])
		d.bits.length += 8
		c.current++
	}
	mask := makeMask(uint(d.bits.length))
	value := mask & d.bits.data
	value >>= (d.bits.length - length)
	d.bits.length -= length
	return value
}

const sliceInitialCapacity = 8

func (d *decoder) growSlice(v reflect.Value, i, newcap int) {
	if i >= v.Cap() {
		if newcap == 0 {
			newcap = v.Cap() + v.Cap()/2
			if newcap < sliceInitialCapacity {
				newcap = sliceInitialCapacity
			}
		} else {
			defer v.SetLen(newcap)
		}
		newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
		reflect.Copy(newv, v)
		v.Set(newv)
	}
	if i >= v.Len() {
		v.SetLen(i + 1)
	}
}

func (d *decoder) setValue(c *cursor, parent reflect.Value, v reflect.Value) error {
	length := uint64(0)
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8:
		length = 1
		v.SetUint(uint64(d.data[c.current]))
	case reflect.Uint16:
		length = 2
		v.SetUint(uint64(binary.BigEndian.Uint16(d.data[c.current : c.current+length])))
	case reflect.Uint32:
		length = 4
		v.SetUint(uint64(binary.BigEndian.Uint32(d.data[c.current : c.current+length])))
	case reflect.Uint64:
		length = 8
		v.SetUint(binary.BigEndian.Uint64(d.data[c.current : c.current+length]))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// not guarantee to read the length of bytes
		length = uint64(v.Type().Bits() / 8)
		ivalue, read := binary.Varint(d.data[c.current : c.current+length])
		if read != int(length) {
			panic(fmt.Errorf("trying to read %d bytes from data, but got %d instead", length, read))
		}
		v.SetInt(ivalue)
	case reflect.Slice:
		panic(fmt.Errorf("can not figure out slice length"))
	case reflect.Array:
		if v.Cap() > 0 {
			for j := 0; j < v.Cap(); j++ {
				fv := v.Index(j)
				if err := d.setValue(c, parent, fv); err != nil {
					return err
				}
			}
		}
	case reflect.Struct:
		return d._struct(c, v)
	}
	c.current += length
	return nil
}

func (d *decoder) setFieldValue(c *cursor, f *field, parent reflect.Value, v reflect.Value) error {
	switch {
	case f.length != nil:
		switch f.length.unit {
		case _bits:
			if (uint(f.length.length) + d.bits.length) > 64 {
				return &UnmarshalBitfieldOverflowError{Field: f}
			}
			value := d.getBitsByLength(c, uint(f.length.length))
			switch v.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				v.SetUint(value)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				v.SetInt(int64(value))
			case reflect.Bool:
				v.SetBool((0xfffffffffffffff1 & value) == 0x1)
			}
		case _byte:
			switch v.Kind() {
			case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				value, read := binary.Uvarint(d.data[c.current : c.current+f.length.length])
				if read != int(f.length.length) {
					panic(fmt.Errorf("trying to read %d bytes from data, but got %d instead", f.length.length, read))
				}
				v.SetUint(value)
			case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
				value, read := binary.Varint(d.data[c.current : c.current+f.length.length])
				if read != int(f.length.length) {
					panic(fmt.Errorf("trying to read %d bytes from data, but got %d instead", f.length.length, read))
				}
				v.SetInt(value)
			case reflect.String:
				v.SetString(string(d.data[c.current : c.current+f.length.length]))
			case reflect.Slice:
				d.growSlice(v, 0, int(f.length.length))
				fallthrough
			case reflect.Array:
				// when type is really array, the byte length specifier is redundant
				for j := 0; j < v.Cap(); j++ {
					fv := v.Index(j)
					// will panic if fv.Kind() is not uint8
					// we are expecting byte spec to be used with bytes
					fv.SetUint(uint64(d.data[int(c.current)+j]))
				}
			}
			c.current += f.length.length
		}
	default:
		return d.setValue(c, parent, v)
	}
	return nil
}
