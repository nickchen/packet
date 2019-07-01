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

// UnmarshalPtrError error from expected pointer not found
type UnmarshalPtrError struct {
	Type reflect.Type
}

func (e *UnmarshalPtrError) Error() string {
	if e.Type == nil {
		return "packet: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "packet: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "packet: Unmarshal(nil " + e.Type.String() + ")"
}

// An UnmarshalTypeError describes a packet value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // description of packet value - "bool", "array", "number -5"
	Type   reflect.Type // type of Go value it could not be assigned to
	Offset int64        // error occurred after reading Offset bytes
	Struct string       // name of the struct type containing the field
	Field  string       // name of the field holding the Go value
}

func (e *UnmarshalTypeError) Error() string {
	if e.Struct != "" || e.Field != "" {
		return "packet: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String()
	}
	return "packet: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
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

var uint82mask = map[uint64]uint8{
	8: 0xff,
	7: 0x7f,
	6: 0x3f,
	5: 0x1f,
	4: 0x0f,
	3: 0x07,
	2: 0x03,
	1: 0x01,
	0: 0x00,
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
	shift := d.bitsLeft - length

	if mask, ok := uint82mask[d.bitsLeft]; ok {
		*into = mask & *d.bits.(*uint8)
		*into >>= shift
		d.bitsLeft -= length
		if d.bitsLeft == 0 {
			d.bits = nil
		}
	} else {
		return fmt.Errorf("unknown mask size (%d)", length)
	}
	return nil
}

var uint162mask = map[uint64]uint16{
	16: 0xffff,
	15: 0x7fff,
	14: 0x3fff,
	13: 0x1fff,
	12: 0x0fff,
	11: 0x07ff,
	10: 0x03ff,
	9:  0x01ff,
	8:  0x00ff,
	7:  0x007f,
	6:  0x003f,
	5:  0x001f,
	4:  0x000f,
	3:  0x0007,
	2:  0x0003,
	1:  0x0001,
	0:  0x0000,
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
	shift := d.bitsLeft - length
	if mask, ok := uint162mask[d.bitsLeft]; ok {
		*into &= mask & *d.bits.(*uint16)
		*into >>= shift
		d.bitsLeft -= length
		if d.bitsLeft == 0 {
			d.bits = nil
		}
	} else {
		return fmt.Errorf("unknown mask size (%d)", length)
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
		case 32 >= s.length:
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
	case f.rest != nil:
		// grab the rest of the data, using the ctx.end - d.current for how much
		if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {

		}
	case f.when != nil:
		if !f.when.eval(parent) {
			return nil
		}
		fallthrough
	default:
		switch v.Kind() {
		case reflect.Uint8:
			v.SetUint(uint64(d.data[d.current]))
			d.current++
		case reflect.Uint16:
			v.SetUint(uint64(binary.BigEndian.Uint16(d.data[d.current : d.current+2])))
			d.current += 2
		case reflect.Array, reflect.Slice:
			arrayKind := d.arrayKind(v)
			if arrayKind == reflect.Uint8 {

			}
			if err := d.array(v); err != nil {
				return err
			}
		default:
			fmt.Printf("TYPE: %v|%v\n", f.Type, f.Type.Kind())
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
		fmt.Printf("TYPE: %v\n", t)
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
		// grow slice
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
		if d.isDone() {
			break
		}
	}
	return nil
}
