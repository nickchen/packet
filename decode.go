package packet

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

const (
	// maxContextStack number of preallocated context for keeping track of range information
	maxContextStack = 128
)
var _structFields map[string][]*field

func init() {
	_structFields = make(map[string][]*field)
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

// UnmarshalBody allow unmarshalling of packet body content, with an interface that provides subsequent types
type UnmarshalBody interface {
	UnmarshalBody() interface{}
}

type context struct {
	start uint64
	end   uint64
}
type decoder struct {
	data    []byte
	current uint64
	offset  uint64
	bits    struct {
		data   uint64
		length uint64
	}
}

func newDecoder(data []byte) (*decoder, error) {
	return &decoder{data: data}, nil
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
	if fs, ok := _structFields[v.Name()]; !ok {
		fs = make([]*field, 0)
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			fs = append(fs, newField(f))
		}
		_structFields[v.Name()] = fs
		return fs
	} else {
		return fs
	}
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

// setStructFieldValueFromSize set value from non-standard size, as denoted by size spec
func (d *decoder) setStructFieldValueFromSize(v reflect.Value, f *field, l *length) error {
	switch l.unit {
	case _bits:
		if (l.length > d.bits.length) && (l.length+d.bits.length) > 64 {
			return &UnmarshalBitfieldOverflowError{Field: f}
		}
		value, err := d.getBitsByLength(l.length)
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
		v.SetBytes(d.data[d.current : d.current+l.length])
		d.current += l.length
	}
	return nil
}

func (d *decoder) growSlice(v reflect.Value, i int) {
	if v.Kind() == reflect.Slice {
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
}

func (d *decoder) setStructFieldValue(v reflect.Value, parent reflect.Value, f *field) error {
	switch {
	case f.length != nil:
		err := d.setStructFieldValueFromSize(v, f, f.length)
		if err != nil {
			return err
		}
	case f.when != nil:
		if !f.when.eval(parent) {
			return nil
		}
		fallthrough
	case f.restOf == true:
		// grab the rest of the data, using the ctx.end - d.current for how much
		if !(v.Kind() == reflect.Array || v.Kind() == reflect.Slice) {
			return nil
		}
		fallthrough
	default:
		switch v.Kind() {
		case reflect.Slice:
			// a common use cases are:
			// 	- set bytes -> array
			//  - dynamic bytes -> slice, base on length
			// other wise we should probably use an interface, use keyword=body
			if v.Type().Elem().Kind() == reflect.Uint8 {
				if v.Cap() == 0 {
					d.growSlice(v, 0)
					// slice
					// if ctx := d.currentContext(); ctx != nil && ctx.end != 0 {
					// 	if size := int(ctx.end - d.current); size > 0 {
					// 		newv := reflect.MakeSlice(v.Type(), v.Len(), size)
					// 		reflect.Copy(newv, v)
					// 		v.Set(newv)
					// 	}
					// }
				}
			}
			fallthrough
		case reflect.Array:
			if err := d._array(v); err != nil {
				return err
			}
		case reflect.Bool:
			value, err := d.getBitsByLength(1)
			if err != nil {
				return err
			}
			v.SetBool((0xfffffffffffffff1 & value) == 0x1)
		case reflect.Uint8:
			v.SetUint(uint64(d.data[d.current]))
			d.current++
		case reflect.Uint16:
			v.SetUint(uint64(binary.BigEndian.Uint16(d.data[d.current : d.current+2])))
			d.current += 2
		case reflect.Uint32:
			v.SetUint(uint64(binary.BigEndian.Uint32(d.data[d.current : d.current+4])))
			d.current += 4
		default:
			// fmt.Printf("UNHANDLE TYPE: %v|%v\n", f.Type, f.Type.Kind())
		}
	}
	return nil
}

// func (d *decoder) currentContext() *context {
// 	if len(d.contexts) > 0 {
// 		return d.contexts[len(d.contexts)-1]
// 	}
// 	return nil
// }

// _struct unmarshals the structure by setting the values base on type, special fields:
//		Length - use to calculate the end for the current structure, required can be use with tag:
//			     XXX to indicate the data field in which the length is for, as it is often done in packet formats
// 		Body - interface place holder for the next layer, optional
func (d *decoder) _struct(v reflect.Value) error {
	// ctx := &context{start: d.current, end: 0}
	// d.contexts = append(d.contexts, ctx)

	valueFields := d.getFields(v.Type())
	for i := 0; i < len(valueFields); i++ {
		f := valueFields[i]
		vf := v.Field(i)
		err := d.setStructFieldValue(vf, v, f)
		if err != nil {
			return err
		}
		switch f.Name {
		case "Length":
			// ctx.end = ctx.start + vf.Uint()
		case "Body":
			if m, ok := v.Interface().(UnmarshalBody); ok {
				b := m.UnmarshalBody()
				if b != nil {
					bv := reflect.ValueOf(b)
					if bv.Kind() != reflect.Ptr {
						panic(fmt.Errorf("invalid return from %s.UnmarshalBody for (%s)", v.Type().Name(), bv.Kind()))
					}
					if err := d._ptr(bv); err != nil {
						return err
					} else {
						vf.Set(bv)
					}				
				}
			}
		}
	}
	return nil
}

func (d *decoder) isDone() bool {
	// ctx := d.currentContext()
	// if ctx != nil && ctx.end != 0 {
	// 	return d.current >= ctx.end
	// }
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
			break
		}
		i++
		if d.isDone() {
			break
		}
	}
	return nil
}
