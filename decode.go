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
	if !v.IsValid() {
		return fmt.Errorf("invalid ptr %+v", v)
	}
	k := v.Kind()
	switch k {
	case reflect.Ptr:
		return d.ptr(v.Elem())
	case reflect.Slice, reflect.Array:
		if err := d.array(v); err != nil {
			return err
		}
	case reflect.Struct:
		if err := d.value(v); err != nil {
			return err
		}
	default:
		return &UnmarshalTypeError{Value: "ptr", Type: v.Type()}
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
	if length > 8 {
		return fmt.Errorf("length exceeded, wanted 8, got %d", length)
	}
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
	shift := d.bitsLeft - length
	*into >>= shift
	d.bitsLeft -= length
	if d.bitsLeft == 0 {
		d.bits = nil
	}
	return nil
}

func (d *decoder) readNext16(length uint64, into *uint16) error {
	if length > 16 {
		return fmt.Errorf("length exceeded, wanted 16, got %d", length)
	}
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
	shift := d.bitsLeft - length
	*into >>= shift
	d.bitsLeft -= length
	if d.bitsLeft == 0 {
		d.bits = nil
	}
	return nil
}

// setFieldValueFromSize set value from non-standard size, as denoted by size spec
func (d *decoder) setFieldValueFromSize(v reflect.Value, f *field, s *size) error {
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

func (d *decoder) arrayKind(v reflect.Value) reflect.Kind {
	d.growSlice(v, 0)
	return v.Index(0).Kind()
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

func (d *decoder) setFieldValue(v reflect.Value, parent reflect.Value, f *field) error {
	switch {
	case f.size != nil:
		err := d.setFieldValueFromSize(v, f, f.size)
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
		case reflect.Array, reflect.Slice:

			// a common use cases are:
			// 	- set bytes -> array
			//  - dynamic bytes -> slice, base on length
			// other wise we should probably use an interface, use keyword=body
			arrayKind := d.arrayKind(v)
			if arrayKind == reflect.Uint8 {
				if v.Cap() == 0 {
					// slice
					if ctx := d.currentContext(); ctx != nil && ctx.end != 0 {
						if size := int(ctx.end - d.current); size > 0 {
							newv := reflect.MakeSlice(v.Type(), v.Len(), size)
							reflect.Copy(newv, v)
							v.Set(newv)
						}
					}
				}
			} else {
				if err := d.array(v); err != nil {
					return err
				}
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
			fmt.Printf("UNHANDLE TYPE: %v|%v\n", f.Type, f.Type.Kind())
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

// unmarshal value
func (d *decoder) value(v reflect.Value) error {
	if !v.IsValid() {
		return fmt.Errorf("invalid value %+v", v)
	}
	t := v.Type()
	switch v.Kind() {
	case reflect.Struct:
		fields := d.getFields(v.Type())
		ctx := &context{start: d.current, end: 0}
		d.contexts = append(d.contexts, ctx)
		for i, f := range fields {
			vf := v.Field(i)
			err := d.setFieldValue(vf, v, f)
			if err != nil {
				return err
			}
			if f.Name == "Length" {
				ctx.end = ctx.start + vf.Uint()
			}
		}
		if m := v.MethodByName("UnmarshalBody"); m.IsValid() {
			if mr := m.Call([]reflect.Value{}); len(mr) == 1 {
				if !mr[0].IsValid() {
					panic(fmt.Errorf("invalid return from UnmarshalBody for (%s)", v.Kind()))
				}
				if mrt := mr[0].Elem(); mrt.Kind() == reflect.Ptr {
					if err := d.ptr(mrt); err != nil {
						return err
					} else {
						body := v.FieldByName("Body")
						if body.IsValid() {
							body.Set(mrt)
						}
					}
				}
			}
		}
	case reflect.Uint8:
		v.SetUint(uint64(d.data[d.current]))
		d.current++
	default:
		return &UnmarshalTypeError{Value: "object", Type: t, Offset: int64(d.current)}
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

func (d *decoder) array(v reflect.Value) error {
	if !v.IsValid() {
		return fmt.Errorf("invalid array %+v", v)
	}
	i := 0
	for {
		d.growSlice(v, i)
		if i < v.Len() {
			// Decode into element.
			if err := d.value(v.Index(i)); err != nil {
				return err
			}
		} else {
			// Ran out of fixed array: skip.
			if err := d.value(reflect.Value{}); err != nil {
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
