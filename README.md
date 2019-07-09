# packet

Packet encoding/decoding without writing specialized code, uses struct field tag to help with processing:

* `length` for length of the field, expecting a number with following specifiers 
    * `b` for bits
    * `B` for bytes
* `lengthfor` indicate the length for the attribute can be return from the object, which needs to provide the `LengthFor` interface
* `lengthtotal` indicate the attribute value is for the whole message stucture

When an `interface{}` field is encounted, `Unmarshal` will check to see if the `struct` satisfies the `InstanceFor` interface, and call the `InstanceFor(fieldname string)` function to get a instance object for the field.

see [fixture](./fixture/fixture.go), and [unittest](./decode_test.go) for example.

