package packet

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ValueFields is an array of fields
type valueFields []*field

var _structFields map[string]*valueFields

func init() {
	_structFields = make(map[string]*valueFields)
}

const tagName = "packet"

type unit uint

const (
	_bits unit = iota
	_byte
)

func (u unit) String() string {
	switch u {
	case _bits:
		return "b"
	case _byte:
		return "B"
	}
	return "uncognized unit(" + string(int(u)) + ")"
}

type length struct {
	unit   unit
	length uint64
}

func (l *length) String() string {
	return fmt.Sprintf("%d%s", l.length, l.unit)
}

type when struct {
	field     string
	condition string
	value     uint64
}

type restFor struct {
	field string
}

type field struct {
	reflect.StructField
	length     *length
	when       *when
	lengthfrom string
	countfrom  string
	f          struct {
		lengthfor  bool
		lengthrest bool
	}
}

func getStructFields(v reflect.Value) *valueFields {
	if v.Kind() != reflect.Struct {
		panic(fmt.Errorf("%s not a struct", v.Kind()))
	}
	vf, ok := _structFields[v.Type().Name()]

	if !ok {
		vf = &valueFields{}
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			*vf = append(*vf, newField(f))
		}
		_structFields[v.Type().Name()] = vf
	}
	return vf
}

func newField(_f reflect.StructField) *field {
	f := &field{StructField: _f}
	f.populateTag()
	return f
}

func (w *when) eval(rv reflect.Value) bool {
	if f := rv.FieldByName(w.field); f.IsValid() {
		v := f.Uint()
		switch w.condition {
		case "gt":
			return w.value > v
		default:
			panic(fmt.Errorf("not handling condition (%s)", w.condition))
		}
	}
	return false
}

func (f *field) populateTag() {
	tags := f.Tag.Get(tagName)
	if tags == "" {
		return
	}
	for _, tag := range strings.Split(tags, ",") {
		head := tag
		if equalAt := strings.Index(tag, "="); equalAt != -1 {
			head = tag[0:equalAt]
			value := tag[equalAt+1:]
			switch head {
			case "length":
				u, err := strconv.ParseUint(value[0:len(value)-1], 10, 64)
				if err != nil {
					panic(fmt.Errorf("failed to parse length (%s)", value))
				}
				switch value[len(value)-1] {
				case 'b':
					f.length = &length{unit: _bits, length: u}
				case 'B':
					f.length = &length{unit: _byte, length: u}
				default:
					panic(fmt.Errorf("not handling unit spec (%s)", value))
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
			case "lengthfrom":
				f.lengthfrom = value
			case "countfrom":
				f.countfrom = value
			default:
				panic(fmt.Errorf("unrecogned header (%s)", head))
			}
		} else {
			switch head {
			case "lengthfor":
				f.f.lengthfor = true
			case "lengthrest":
				f.f.lengthrest = true
			default:
				panic(fmt.Errorf("unrecogned header (%s) tags (%s)", head, tags))
			}
		}
	}
}
