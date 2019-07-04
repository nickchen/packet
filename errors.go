package packet

import (
	"reflect"
)

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

type UnmarshalBitfieldOverflowError struct {
	Struct string
	Field  *field
}

func (e *UnmarshalBitfieldOverflowError) Error() string {
	if e.Struct != "" || e.Field != nil {
		return "packet: cannot unmarshal " + e.Field.Name + " into Go struct field " + e.Struct + ", size over 64bits"
	}
	return "packet: cannot unmarshal into Go value of type " + e.Struct
}
