package packet

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

// Unmarshal parson the packet data and stores the result in value pointed by v.
// If v is nil or not a pointer, Unmarshal returns an InvalidUnmarshalError.
func Unmarshal(data []byte, v interface{}) error {
	decoder, err := newDecoder(data)
	if err != nil {
		return err
	}
	return decoder.decode(v)
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
	data     []byte
	imap     map[string]interface{}
	current  uint64
	offset   uint64
	bits     interface{}
	bitsLeft uint64
	contexts []*context
}

type reader struct {
	data    []byte
	current uint64
	offset  uint64
}

func newReader(data []byte) (*reader, error) {
	return &reader{data: data}, nil
}

func newDecoder(data []byte) (*decoder, error) {
	return &decoder{
		data:     data,
		imap:     make(map[string]interface{}),
		contexts: make([]*context, 0)}, nil
}

func (d *decoder) decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &UnmarshalPtrError{reflect.TypeOf(v)}
	}
	if err := d.ptr(rv); err != nil {
		return err
	}
	return nil
}

func (d *decoder) ptr(v reflect.Value) error {
	// XXX just here while debugging
	if v.Kind() != reflect.Ptr {
		return &UnmarshalPtrError{reflect.TypeOf(v)}
	}

	pv := v.Elem()
	switch pv.Kind() {
	case reflect.Ptr:
		// double pointer
		if err := d.ptr(pv); err != nil {
			return err
		}
	case reflect.Slice, reflect.Array:
		if err := d.array(pv); err != nil {
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
	fs := make([]*field, 0)
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		fs = append(fs, newField(f))
	}
	return fs
}

func (d *decoder) readNext8(length uint64, into *uint8) error {
	switch d.bits.(type) {
	case *uint8:
	case nil:
		if d.bitsLeft != 0 {
			return fmt.Errorf("have bits left, but no holder")
		}
		b := d.data[d.current]
		d.current++
		d.bitsLeft = 8
		d.bits = &b
	default:
		return fmt.Errorf("unknown bits type %+v (8)", d.bits)
	}
	if length > d.bitsLeft {
		return fmt.Errorf("have bits left (%d), not enough, wanted (%d)", d.bitsLeft, length)
	}
	// we want to read length X from the top of the left over bits, assuming 0 leading,
	// shift right bitsLeft - length, and read what's left over
	mask := makeMask8(d.bitsLeft)
	*into = mask & *d.bits.(*uint8)
	*into >>= (d.bitsLeft - length)
	d.bitsLeft -= length
	if d.bitsLeft == 0 {
		d.bits = nil
	}
	return nil
}

func (d *decoder) readNext16(length uint64, into *uint16) error {
	switch d.bits.(type) {
	case *uint16:
		// no op
	case *uint8:
		readLeft := length - d.bitsLeft
		b := uint8(8)
		err := d.readNext8(d.bitsLeft, &b)
		if err != nil {
			return err
		}
		if readLeft <= 8 {
			*into &= uint16(b)
			*into <<= length
			b = 0
			d.readNext8(readLeft, &b)
			*into &= uint16(b)
			return nil
		}
	case nil:
		b := binary.BigEndian.Uint16(d.data[d.current : d.current+1])
		d.current++
		d.bitsLeft = 16
		d.bits = &b
	default:
		return fmt.Errorf("unknown bits type %+v (16)", d.bits)
	}
	if length > d.bitsLeft {
		return fmt.Errorf("have bits left (%d), not enough, wanted (%d)", d.bitsLeft, length)
	}
	// we want to read length X from the top of the left over bits, assuming 0 leading,
	// shift right bitsLeft - length, and read what's left over
	mask := makeMask16(d.bitsLeft)
	*into &= mask & *d.bits.(*uint16)
	*into >>= (d.bitsLeft - length)
	d.bitsLeft -= length
	if d.bitsLeft == 0 {
		d.bits = nil
	}
	return nil
}

// setStructFieldValueFromSize set value from non-standard size, as denoted by size spec
func (d *decoder) setStructFieldValueFromSize(v reflect.Value, f *field, s *size) error {
	switch s.unit {
	case _bits:
		switch {
		case 8 >= s.length:
			b := uint8(0)
			err := d.readNext8(s.length, &b)
			if err != nil {
				return err
			}
			v.SetUint(uint64(b))
		case 16 >= s.length:
			b := uint16(0)
			err := d.readNext16(s.length, &b)
			if err != nil {
				return err
			}
			v.SetUint(uint64(b))
		default:
			return fmt.Errorf("can not handle bit size (%d) read", s.length)
		}
		// reading bits from byte data
	case _byte:
		v.SetBytes(d.data[d.current : d.current+s.length])
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
	case f.size != nil:
		err := d.setStructFieldValueFromSize(v, f, f.size)
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
					if ctx := d.currentContext(); ctx != nil && ctx.end != 0 {
						if size := int(ctx.end - d.current); size > 0 {
							newv := reflect.MakeSlice(v.Type(), v.Len(), size)
							reflect.Copy(newv, v)
							v.Set(newv)
						}
					}
				}
			}
			fallthrough
		case reflect.Array:
			if err := d.array(v); err != nil {
				return err
			}
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

func (d *decoder) currentContext() *context {
	if len(d.contexts) > 0 {
		return d.contexts[len(d.contexts)-1]
	}
	return nil
}

// _struct unmarshals the structure by setting the values base on type, special fields:
//		Length - use to calculate the end for the current structure, required can be use with tag:
//			     XXX to indicate the data field in which the length is for, as it is often done in packet formats
// 		Body - interface place holder for the next layer, optional
func (d *decoder) _struct(v reflect.Value) error {
	ctx := &context{start: d.current, end: 0}
	d.contexts = append(d.contexts, ctx)
	for i, f := range d.getFields(v.Type()) {
		vf := v.Field(i)
		err := d.setStructFieldValue(vf, v, f)
		if err != nil {
			return err
		}
		switch f.Name {
		case "Length":
			ctx.end = ctx.start + vf.Uint()
		case "Body":
			if m := v.MethodByName("UnmarshalBody"); m.IsValid() {
				// call the function UnmarshalBody, which should return an object to be set for Body as protocol dictates
				if mr := m.Call([]reflect.Value{}); len(mr) == 1 {
					// the return from call is an array, we only expect one element
					if !mr[0].IsValid() {
						panic(fmt.Errorf("invalid return from UnmarshalBody for (%s)", v.Kind()))
					}
					// the return should be an ptr
					mrt := mr[0].Elem()
					if err := d.ptr(mrt); err != nil {
						return err
					} else {
						vf.Set(mrt)
					}
				}
			}
		}
	}
	return nil
}

func (d *decoder) isDone() bool {
	ctx := d.currentContext()
	if ctx != nil && ctx.end != 0 {
		return d.current >= ctx.end
	}
	return int(d.current) >= len(d.data)
}

// element set value for array/slice element
func (d *decoder) element(v reflect.Value) error {
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
func (d *decoder) array(v reflect.Value) error {
	i := 0
	for {
		if v.Kind() == reflect.Slice {
			d.growSlice(v, i)
		}
		if i < v.Len() {
			// Decode into element.
			if err := d.element(v.Index(i)); err != nil {
				return err
			}
		} else {
			// Ran out of fixed array: skip.
			if err := d.element(reflect.Value{}); err != nil {
				return err
			}
		}
		i++
		if d.isDone() {
			break
		}
	}
	return nil
}
