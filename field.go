package packet

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const tagName = "packet"

type unit uint

const (
	_bits unit = iota
	_byte
)

type size struct {
	unit   unit
	length uint64
}

type when struct {
	field     string
	condition string
	value     uint64
}

type rest struct {
	field string
}
type field struct {
	reflect.StructField
	size *size
	when *when
	rest *rest
}

func newField(_f reflect.StructField) *field {
	f := &field{_f, nil, nil, nil}
	f.getTag()
	return f
}

func (w *when) eval(rv reflect.Value) bool {
	if f := rv.FieldByName(w.field); f.IsValid() {
		v := f.Uint()
		switch w.condition {
		case "gt":
			return w.value > v
		}
		fmt.Printf("FIELD: %+v\n", f)
	}
	return false
}
func (f *field) getTag() {
	tags := f.Tag.Get(tagName)
	for _, tag := range strings.Split(tags, ",") {
		equalIndex := strings.Index(tag, "=")
		if equalIndex != -1 {
			head := tag[0:equalIndex]
			value := tag[equalIndex+1:]
			switch head {
			case "size":
				u, err := strconv.ParseUint(value[0:len(value)-1], 10, 64)
				if err != nil {
					panic(fmt.Errorf("failed to parse size (%s)", value))
				}
				switch value[len(value)-1] {
				case 'b':
					f.size = &size{unit: _bits, length: u}
				case 'B':
					f.size = &size{unit: _byte, length: u}
				default:
					panic(fmt.Errorf("invalid unit spec (%s)", value))
				}
			case "when":
				c := strings.Split(value, "-")
				if len(c) != 3 {
					panic(fmt.Errorf("(when) should have 3 words seprated by (-), but got %s", value))
				}
				v, err := strconv.ParseUint(c[2], 10, 32)
				if err != nil {
					panic(fmt.Errorf("failed to parse: %s", err))
				}
				f.when = &when{field: c[0], condition: c[1], value: v}
			case "rest":
				f.rest = &rest{field: value}
			default:
				panic(fmt.Errorf("unrecogned header (%s)", head))
			}
		}
	}
}
