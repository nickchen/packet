# packet

Packet encoding/decoding without writing specialized code, uses struct field tag to help with processing:

* `length` for length of the field, expecting a number with following specifiers 
    * `b` for bits
    * `B` for bytes
* `when` for optional fields, will perform logical evaluation with following expections: `<field>-<logical test>-<value>`, where `field` is the name of of a field in the current struct, where a uint value is retrieved; `logical test` can be one of `gt`; `value` is the limit which logical test is perform with the retrieved value.

Following struct fields are have special meanings:

* Length indicate the total length of the current struct, can be use to calculate the Body as []byte when 
* Body should be a `interface{}` field unless a specific type is know, then it should be a pointer to the type in question, will test to see if the struct satisfy `BodyStruct` interface by calling `BodyStruct()`

see [fixture](./fixture/fixture.go), and [unittest](./decode_test.go) for example.

