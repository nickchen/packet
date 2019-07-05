package packet

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"
)

// context holds all the bytes for the current Array/Slice/Struct
// Note: parent are responsible for allocating the context before
// the _struct/_array call,
type context struct {
	current uint64
	data    []byte
}

var _structFields map[string][]*field
var _contextPool sync.Pool

func init() {
	_structFields = make(map[string][]*field)
	_contextPool = sync.Pool{
		New: func() interface{} {
			// allocate and return a new context
			return new(context)
		},
	}
}

// Unmarshal parson the packet data and stores the result in value pointed by v.
// If v is nil or not a pointer, Unmarshal returns an InvalidUnmarshalError.
func Unmarshal(data []byte, v interface{}) error {
	d, err := newDecoder(data)
	if err != nil {
		return err
	}
	return d.decode(v)
}

// BodyStruct allow unmarshalling of packet body content, with an interface that provides subsequent type from current value
type BodyStruct interface {
	BodyStruct() interface{}
}

type decoder struct {
	data    []byte
	start   uint64
	current uint64
	end     uint64
	bits    struct {
		data   uint64
		length uint64
	}
}

func newDecoder(data []byte) (*decoder, error) {
	return &decoder{data: data, end: uint64(len(data))}, nil
}

func (d *decoder) decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &UnmarshalPtrError{reflect.TypeOf(v)}
	}
	if err := d._ptr(rv); err != nil {
		return err
	}
	return nil
}

func (d *decoder) _ptr(v reflect.Value) error {
	// XXX just here while debugging
	if v.Kind() != reflect.Ptr {
		return &UnmarshalPtrError{reflect.TypeOf(v)}
	}

	pv := v.Elem()
	switch pv.Kind() {
	case reflect.Ptr:
		// double pointer
		if err := d._ptr(pv); err != nil {
			return err
		}
	case reflect.Slice, reflect.Array:
		if err := d._array(pv); err != nil {
			return err
		}
	case reflect.Struct:
		if err := d._struct(pv); err != nil {
			return err
		}
	default:
		return &UnmarshalTypeError{Value: "ptr", Type: pv.Type()}
	}
	return nil
}

func (d *decoder) getFields(v reflect.Type) []*field {
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

func (d *decoder) getBitsByLength(length uint64) (uint64, error) {
	for d.bits.length < length {
		d.bits.data <<= 8
		d.bits.data |= uint64(d.data[d.current])
		d.bits.length += 8
		d.current++
	}
	mask := makeMask(d.bits.length)
	value := mask & d.bits.data
	value >>= (d.bits.length - length)
	d.bits.length -= length
	return value, nil
}

// setStructFieldValueFromLength set value from non-standard size, as denoted by size spec
func (d *decoder) setStructFieldValueFromLength(v reflect.Value, parent reflect.Value, f *field) error {
	switch f.length.unit {
	case _bits:
		if (f.length.length > d.bits.length) && (f.length.length+d.bits.length) > 64 {
			return &UnmarshalBitfieldOverflowError{Field: f}
		}
		value, err := d.getBitsByLength(f.length.length)
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
		if err := d.rangeCheck(parent, f, f.length.length); err != nil {
			return err
		}
		v.SetBytes(d.data[d.current : d.current+f.length.length])
		d.current += f.length.length
	}
	return nil
}

func (d *decoder) growSlice(v reflect.Value, i int) {
	if i >= v.Cap() {
		newcap := v.Cap() + v.Cap()/2
		if newcap < 4 {
			newcap = 4
		}
		newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
		reflect.Copy(newv, v)
		v.Set(newv)
	}
	if i >= v.Len() {
		v.SetLen(i + 1)
	}
}

func (d *decoder) getStructUnint(parent reflect.Value, fieldname string) uint64 {
	return parent.FieldByName(fieldname).Uint()
}

func (d *decoder) rangeCheck(parent reflect.Value, f *field, length uint64) error {
	if (d.current + length) >= d.end {
		return &UnmarshalUnexpectedEnd{Struct: parent.Type().Name(), Field: f.Name, Offset: int64(d.current), End: int64(d.end)}
	}
	return nil
}

func (d *decoder) setStructFieldValue(v reflect.Value, parent reflect.Value, f *field) error {
	var length uint64
	switch {
	case f.length != nil:
		// have length specification
		err := d.setStructFieldValueFromLength(v, parent, f)
		if err != nil {
			return err
		}
	case f.when != nil:
		if !f.when.eval(parent) {
			return nil
		}
		fallthrough
	case f.lengthfrom != "":
		// have length specification from another field in the same struct
		length = d.getStructUnint(parent, f.lengthfrom)
		if length == 0 {
			return nil
		}
	default:
		k := v.Kind()
		switch k {
		case reflect.Slice, reflect.Array:
			// a common use cases are:
			// 	- set bytes -> array
			//  - dynamic bytes -> slice, base on length
			// other wise we should probably use an interface, use keyword=body
			if err := d._array(v); err != nil {
				return err
			}
		case reflect.Bool:
			value, err := d.getBitsByLength(1)
			if err != nil {
				return err
			}
			v.SetBool((0xfffffffffffffff1 & value) == 0x1)
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			length = map[reflect.Kind]uint64{reflect.Uint8: 1, reflect.Uint16: 2, reflect.Uint32: 4, reflect.Uint64: 8}[k]
			if err := d.rangeCheck(parent, f, length); err != nil {
				return err
			}
			var value uint64
			switch k {
			case reflect.Uint8:
				value = uint64(d.data[d.current])
			case reflect.Uint16, reflect.Uint32, reflect.Uint64:
				value = uint64(binary.BigEndian.Uint16(d.data[d.current : d.current+length]))
			}
			v.SetUint(value)
			d.current += length
		case reflect.Interface:
			if f.Name == "Body" {
				return nil
			}
			fallthrough
		default:
			panic(fmt.Errorf("unhandled type (%s) for field %s.%s", f.Type, parent.Type().Name(), f.Name))
		}
	}
	return nil
}

// _struct unmarshals the structure by setting the values base on type, special fields:
//		Length - use to calculate the end for the current structure, required can be use with tag:
//			     XXX to indicate the data field in which the length is for, as it is often done in packet formats
// 		Body - interface place holder for the next layer, optional
func (d *decoder) _struct(v reflect.Value) error {
	valueFields := d.getFields(v.Type())
	d.start = d.current
	for i := 0; i < len(valueFields); i++ {
		f := valueFields[i]
		vf := v.Field(i)
		err := d.setStructFieldValue(vf, v, f)
		if err != nil {
			return err
		}
		switch f.Name {
		case "Length":
			if f.isTotal {
				d.end = d.start + vf.Uint()
			}
		case "Body":
			if m, ok := v.Interface().(BodyStruct); ok {
				b := m.BodyStruct()
				if b != nil {
					bv := reflect.ValueOf(b)
					if bv.Kind() != reflect.Ptr {
						panic(fmt.Errorf("invalid return from %s.BodyStruct for (%s)", v.Type().Name(), bv.Kind()))
					}
					// set body before the decoding process, so it should be returned along with error if any
					vf.Set(bv)
					err := d._ptr(bv)
					if err != nil {
						return err
					}
				} else if d.end != 0 && d.end > d.current {
					vf.Set(reflect.ValueOf(d.data[d.current:d.end]))
				} else {
					panic(fmt.Errorf("unhandled body"))
				}
			} else {
				panic(fmt.Errorf("%s does not implment the BodyStruct interface", v.Type().Name()))
			}
		}
	}
	return nil
}

func (d *decoder) isDone() bool {
	return int(d.current) >= len(d.data)
}

// element set value for array/slice element
func (d *decoder) _element(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		if err := d._struct(v); err != nil {
			return err
		}
	case reflect.Uint8:
		v.SetUint(uint64(d.data[d.current]))
		d.current++
	default:
		return &UnmarshalTypeError{Value: "element", Type: v.Type(), Offset: int64(d.current)}
	}
	return nil
}

// array set array values, grow slices as necessary
func (d *decoder) _array(v reflect.Value) error {
	i := 0
	for {
		if v.Kind() == reflect.Slice {
			d.growSlice(v, i)
		}
		if i < v.Len() {
			// Decode into element.
			if err := d._element(v.Index(i)); err != nil {
				return err
			}
		} else {
			// reach capacity
			break
		}
		i++
	}
	return nil
}
