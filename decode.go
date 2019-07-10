package packet

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
		length uint64
	}
}

// InstanceFor interface helps the unmarshaller to figure out the right type base on message data, by returning the object reference for the attribute in question
type InstanceFor interface {
	InstanceFor(fieldname string) interface{}
}

// LengthFor interface helps the unmarshaller to figure out the right length for bytes
type LengthFor interface {
	LengthFor(fieldname string) uint64
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
	case reflect.Array, reflect.Slice:
		if v.Kind() == reflect.Slice {
			// grow initial slice
			d.growSlice(v, 0, 0)
		}
		for j := 0; j <= v.Cap() && c.current < c.end; j++ {
			if v.Kind() == reflect.Slice {
				if j >= v.Cap() {
					// Set the len
					d.growSlice(v, j, 0)
				}
				if j >= v.Len() {
					v.SetLen(j + 1)
				}
			} else if j >= v.Cap() {
				break
			}
			fv := v.Index(j)
			if err := d.setValue(c, reflect.StructField{}, reflect.Value{}, fv); err != nil {
				return err
			}
		}
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

func (d *decoder) getBitsByLength(c *cursor, length uint64) uint64 {
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

func (d *decoder) growSlice(v reflect.Value, i, len int) {
	if i >= v.Cap() {
		newcap := len
		if newcap == 0 {
			newcap = v.Cap() + v.Cap()/2
			if newcap < sliceInitialCapacity {
				newcap = sliceInitialCapacity
			}
		}
		newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
		reflect.Copy(newv, v)
		v.Set(newv)
	}
}

func (d *decoder) setUintValue(c *cursor, f reflect.StructField, parent reflect.Value, v reflect.Value) error {
	length := uint64(0)
	switch v.Kind() {
	case reflect.Uint8:
		length = 1
	case reflect.Uint16:
		length = 2
	case reflect.Uint32:
		length = 4
	case reflect.Uint64:
		length = 8
	}
	if (c.end - c.current) < length {
		length = c.end - c.current
	}
	value := uint64(0)
	switch length {
	case 1:
		value = uint64(d.data[c.current])
	case 2:
		value = uint64(binary.BigEndian.Uint16(d.data[c.current : c.current+length]))
	case 4:
		value = uint64(binary.BigEndian.Uint32(d.data[c.current : c.current+length]))
	case 8:
		value = binary.BigEndian.Uint64(d.data[c.current : c.current+length])
	}
	v.SetUint(value)
	c.current += length
	return nil
}

func (d *decoder) setValue(c *cursor, f reflect.StructField, parent reflect.Value, v reflect.Value) error {
	if c.current >= c.end && v.Kind() != reflect.Interface {
		return nil
	}
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return d.setUintValue(c, f, parent, v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// not guarantee to read the length of bytes, probably doesn't make sense
		// to have Int in a packet message, but here we are
		length := uint64(v.Type().Bits() / 8)
		ivalue, read := binary.Varint(d.data[c.current : c.current+length])
		if read != int(length) {
			panic(fmt.Errorf("trying to read %d bytes from data, but got %d instead", length, read))
		}
		v.SetInt(ivalue)
		c.current += length
	case reflect.Slice:
		// grow initial capacity
		d.growSlice(v, 0, 0)
		fallthrough
	case reflect.Array:
		// Len for number of existing elements
		// Cap for how big slice can grow
		for j := 0; j <= v.Cap() && c.current < c.end; j++ {
			if v.Kind() == reflect.Slice {
				if j >= v.Cap() {
					// Set the len
					d.growSlice(v, j, 0)
				}
				if j >= v.Len() {
					v.SetLen(j + 1)
				}
			} else if j >= v.Cap() {
				break
			}
			fv := v.Index(j)
			if err := d.setValue(c, f, parent, fv); err != nil {
				return err
			}
		}
	case reflect.Struct:
		return d._struct(c, v)
	case reflect.Ptr:
		// get underlying pointer type, and instantiate a new instance, set it as value, then use it as struct
		pv := reflect.New(v.Type().Elem())
		v.Set(pv)
		return d.setValue(c, f, parent, pv.Elem())
	case reflect.Interface:
		if m, ok := parent.Interface().(InstanceFor); ok {
			i := m.InstanceFor(f.Name)
			if i != nil {
				iv := reflect.ValueOf(i)
				// set body before the decoding process, so it should be returned along with error if any
				v.Set(iv)
				return d.setValue(c, f, parent, iv.Elem())
			}
			return nil
		}
		return fmt.Errorf("no valid InstanceFor returned for %s.%s %v", parent.Type().Name(), f.Name, v)
	case reflect.Bool:
		return d.setBitFieldValue(c, f, _bits, 1, parent, v)
	default:
		panic(fmt.Errorf("unhandled type %s.%s %+v", parent.Type().Name(), v.Type(), v))
	}
	return nil
}

func (d *decoder) setBitFieldValue(c *cursor, f reflect.StructField, u unit, length uint64, parent reflect.Value, v reflect.Value) error {
	switch u {
	case _bits:
		if (length + d.bits.length) > 64 {
			return &UnmarshalBitfieldOverflowError{Field: f}
		}
		value := d.getBitsByLength(c, length)
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
			value, read := binary.Uvarint(d.data[c.current : c.current+length])
			if read != int(length) {
				panic(fmt.Errorf("trying to read %d bytes from data, but got %d instead", length, read))
			}
			v.SetUint(value)
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			value, read := binary.Varint(d.data[c.current : c.current+length])
			if read != int(length) {
				panic(fmt.Errorf("trying to read %d bytes from data, but got %d instead", length, read))
			}
			v.SetInt(value)
		case reflect.String:
			v.SetString(string(d.data[c.current : c.current+length]))
		case reflect.Slice:
			d.growSlice(v, 0, int(length))
			v.SetLen(int(length))
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
		c.current += length
	}
	return nil
}

func (d *decoder) setFieldValue(c *cursor, f *field, parent reflect.Value, v reflect.Value) error {
	newc := c
	switch {
	case f.length != nil:
		return d.setBitFieldValue(c, f.StructField, f.length.unit, f.length.length, parent, v)
	case f.f.lengthfor:
		// call length for interface
		if m, ok := parent.Interface().(LengthFor); ok {
			length := m.LengthFor(f.Name)
			newc = &cursor{start: c.current, end: c.current + length, current: c.current}
			// if length is zero or the new boundry is after previous end
			if newc.end > c.end {
				return nil
			}
		}
		fallthrough
	default:
		err := d.setValue(newc, f.StructField, parent, v)
		if newc != c {
			c.current += (newc.current - newc.start)
		}
		return err
	}
}
