package packet

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

var _structFields map[string][]*field

func init() {
	_structFields = make(map[string][]*field)
}

// Unmarshal parson the packet data and stores the result in value pointed by v.
// If v is nil or not a pointer, Unmarshal returns an InvalidUnmarshalError.
func Unmarshal(data []byte, v interface{}) error {
	c := &context{data: data}
	c.ends.current = 0
	c.ends.stack[c.ends.current] = uint64(len(data))
	return c.decode(v)
}

// BodyStruct allow unmarshalling of packet body content, with an interface that provides subsequent type from current value
type BodyStruct interface {
	BodyStruct() interface{}
}

// LengthFor allow variable byte size for a field, this function should return the length in byte for the provided field
type LengthFor interface {
	LengthFor(fieldname string) uint64
}

const maxEnds = 16

type context struct {
	data    []byte
	start   uint64
	current uint64
	ends    struct {
		current uint64
		stack   [maxEnds]uint64
	}
	bits struct {
		data   uint64
		length uint64
	}
}

func (c *context) end() uint64 {
	return c.ends.stack[c.ends.current]
}

func (c *context) release() {
	if c.ends.current > 0 {
		c.ends.current--
	} else {
		panic(fmt.Errorf("release ends from zero"))
	}
}

// push a new end into the top, until it's released, use this end as data end
func (c *context) push(end uint64) {
	c.ends.current++
	c.ends.stack[c.ends.current] = end
}

func (c *context) decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &UnmarshalPtrError{reflect.TypeOf(v)}
	}
	return c._ptr(rv)
}

func (c *context) _ptr(v reflect.Value) error {
	// XXX just here while debugging
	if v.Kind() != reflect.Ptr {
		return &UnmarshalPtrError{reflect.TypeOf(v)}
	}

	pv := v.Elem()
	switch pv.Kind() {
	case reflect.Ptr:
		// double pointer
		return c._ptr(pv)
	case reflect.Slice, reflect.Array:
		return c._array(pv)
	case reflect.Struct:
		return c._struct(pv)
	}
	return &UnmarshalTypeError{Value: "ptr", Type: pv.Type()}
}

func (c *context) getFields(v reflect.Type) []*field {
	fs, ok := _structFields[v.Name()]
	if !ok {
		fs = make([]*field, 0)
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			fs = append(fs, newField(f))
		}
		_structFields[v.Name()] = fs
		return fs
	}
	return fs
}

func (c *context) getBitsByLength(length uint64) (uint64, error) {
	for c.bits.length < length {
		c.bits.data <<= 8
		c.bits.data |= uint64(c.data[c.current])
		c.bits.length += 8
		c.current++
	}
	mask := makeMask(c.bits.length)
	value := mask & c.bits.data
	value >>= (c.bits.length - length)
	c.bits.length -= length
	return value, nil
}

// setStructFieldValueFromLength set value from non-standard size, as denoted by size spec
func (c *context) setStructFieldValueFromLength(v reflect.Value, parent reflect.Value, f *field) error {
	switch f.length.unit {
	case _bits:
		if (f.length.length > c.bits.length) && (f.length.length+c.bits.length) > 64 {
			return &UnmarshalBitfieldOverflowError{Field: f}
		}
		value, err := c.getBitsByLength(f.length.length)
		if err != nil {
			return err
		}
		switch v.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v.SetUint(value)
		case reflect.Bool:
			v.SetBool((0xfffffffffffffff1 & value) == 0x1)
		}
	case _byte:
		if err := c.rangeCheck(parent, f, f.length.length); err != nil {
			return err
		}
		v.SetBytes(c.data[c.current : c.current+f.length.length])
		c.current += f.length.length
	}
	return nil
}

// getStructUnint get the uint64 attribute value for the current instance
func (c *context) getStructUnint(parent reflect.Value, fieldname string) uint64 {
	return parent.FieldByName(fieldname).Uint()
}

func (c *context) rangeCheck(parent reflect.Value, f *field, length uint64) error {
	if (c.current + length) > c.end() {
		return &UnmarshalUnexpectedEnd{Struct: parent.Type().Name(), Field: f.Name, Offset: int64(c.current), End: int64(c.end())}
	}
	return nil
}

func (c *context) setStructFieldValue(v reflect.Value, parent reflect.Value, f *field) error {
	if f.length != nil {
		return c.setStructFieldValueFromLength(v, parent, f)
	}
	if f.when != nil && !f.when.eval(parent) {
		return nil
	}
	// primatives
	switch v.Kind() {
	case reflect.Bool:
		value, err := c.getBitsByLength(1)
		if err != nil {
			return err
		}
		v.SetBool((0xfffffffffffffff1 & value) == 0x1)
		return nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var length uint64
		if f.f.lengthfor {
			if m, ok := parent.Interface().(LengthFor); ok {
				length = m.LengthFor(f.Name)
			} else {
				return fmt.Errorf("no LengthFor function for struct %s", parent.Type().Name())
			}
		} else {
			length = map[reflect.Kind]uint64{reflect.Uint8: 1, reflect.Uint16: 2, reflect.Uint32: 4, reflect.Uint64: 8}[v.Kind()]
		}
		if err := c.rangeCheck(parent, f, length); err != nil {
			return err
		}
		var value uint64
		switch length {
		case 1:
			value = uint64(c.data[c.current])
		case 2:
			value = uint64(binary.BigEndian.Uint16(c.data[c.current : c.current+length]))
		case 4:
			value = uint64(binary.BigEndian.Uint32(c.data[c.current : c.current+length]))
		case 8:
			value = uint64(binary.BigEndian.Uint64(c.data[c.current : c.current+length]))
		}
		v.SetUint(value)
		c.current += length
		return nil
	}

	if v.Kind() == reflect.Interface {
		if f.Name == "Body" || f.Name == "Data" {
			// get the next struct instance from the BodyStruct interface
			if m, ok := parent.Interface().(BodyStruct); ok {
				b := m.BodyStruct()
				if b != nil {
					bv := reflect.ValueOf(b)
					if bv.Kind() != reflect.Ptr {
						panic(fmt.Errorf("invalid return from %s.BodyStruct for (%s)", v.Type().Name(), bv.Kind()))
					}
					// set body before the decoding process, so it should be returned along with error if any
					v.Set(bv)
					v = bv
					if v.Kind() == reflect.Interface {
						return fmt.Errorf("double interface type %s.%s", parent.Type().Name(), f.Name)
					}
				} else {
					return fmt.Errorf("function BodyStruct returned nil for %s.%s", parent.Type().Name(), f.Name)
				}
			}
		} else {
			panic(fmt.Errorf("unhandled interface type %s.%s", parent.Type().Name(), f.Name))
		}
	}

	var length uint64
	if f.lengthfrom != "" {
		// have length specification from another field in the same struct
		length = c.getStructUnint(parent, f.lengthfrom)
	} else if f.f.lengthfor {
		if m, ok := parent.Interface().(LengthFor); ok {
			length = m.LengthFor(f.Name)
		} else {
			panic(fmt.Errorf("no LengthFor function for struct %s", parent.Type().Name()))
		}
	}

	switch v.Kind() {
	case reflect.Slice:
		if length == 0 {
			return nil
		}
		c.push(c.current + length)
		defer c.release()
		fallthrough
	case reflect.Array:
		// a common use cases are:
		// 	- set bytes -> array
		//  - dynamic bytes -> slice, base on length
		// other wise we should probably use an interface, use keyword=body
		return c._array(v)
	case reflect.Ptr:
		return c._ptr(v)
	default:
		panic(fmt.Errorf("unhandled type (%s) for field %s.%s", v, parent.Type().Name(), f.Name))
	}
}

// _struct unmarshals the structure by setting the values base on type, special fields:
//		Length - use to calculate the end for the current structure, required can be use with tag:
//			     XXX to indicate the data field in which the length is for, as it is often done in packet formats
// 		Body - interface place holder for the next layer, optional
func (c *context) _struct(v reflect.Value) error {
	valueFields := c.getFields(v.Type())
	for i := 0; i < len(valueFields); i++ {
		f := valueFields[i]
		fv := v.Field(i)
		err := c.setStructFieldValue(fv, v, f)
		if err != nil {
			return err
		}
	}
	return nil
}

// element set value for array/slice element
func (c *context) _element(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		return c._struct(v)
	case reflect.Uint8:
		// bytes
		v.SetUint(uint64(c.data[c.current]))
		c.current++
	default:
		return &UnmarshalTypeError{Value: "element", Type: v.Type(), Offset: int64(c.current)}
	}
	return nil
}

const sliceInitialCapacity = 8

func (c *context) growSlice(v reflect.Value, i int) {
	if i >= v.Cap() {
		newcap := v.Cap() + v.Cap()/2
		if newcap < sliceInitialCapacity {
			newcap = sliceInitialCapacity
		}
		newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
		reflect.Copy(newv, v)
		v.Set(newv)
	}
	if i >= v.Len() {
		v.SetLen(i + 1)
	}
}

// _array handles slices or arrays
func (c *context) _array(v reflect.Value) error {
	for i := 0; c.current < c.end(); i++ {
		if v.Kind() == reflect.Slice && i >= v.Len() {
			c.growSlice(v, i)
		}
		if i >= v.Cap() {
			break
		}
		if err := c._element(v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}
