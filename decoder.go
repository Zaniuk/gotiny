package gotiny

import (
	"reflect"
	"unsafe"
)

// Decoder is a structure that holds the state for decoding operations.
// It contains a buffer, an index to track the next byte to be used,
// and fields to manage the position and bit of the next boolean value
// to be read. Additionally, it maintains a collection of decoders
// and the count of these decoders.
type Decoder struct {
	buf     []byte // buffer
	index   int    // index of the next byte to be used in the buffer
	boolPos byte   // index of the next bool to be read in the buffer, i.e., buf[boolPos]
	boolBit byte   // bit position of the next bool to be read in buf[boolPos]

	engines []decEng // collection of decoders
	length  int      // number of decoders
}

// Unmarshal decodes the provided byte buffer into the given variables.
// The variables to decode into are passed as variadic parameters.
//
// Parameters:
//   - buf: The byte buffer to decode.
//   - is: Variadic parameters representing the variables to decode into.
//
// Returns:
//
//	The number of bytes read from the buffer.
func Unmarshal(buf []byte, is ...any) int {
	return NewDecoderWithPtr(is...).decode(buf, is...)
}

// NewDecoderWithPtr creates a new Decoder instance with the provided pointers.
// Each argument must be a pointer type, otherwise the function will panic.
// The function initializes decoding engines for each provided pointer type.
//
// Parameters:
//
//	is ...any - A variadic parameter accepting any number of arguments, each of which must be a pointer.
//
// Returns:
//
//	*Decoder - A pointer to the newly created Decoder instance.
func NewDecoderWithPtr(is ...any) *Decoder {
	l := len(is)
	engines := make([]decEng, l)
	for i := 0; i < l; i++ {
		rt := reflect.TypeOf(is[i])
		if rt.Kind() != reflect.Ptr {
			panic("the argument must be a pointer type!")
		}
		engines[i] = getDecEngine(rt.Elem())
	}
	return &Decoder{
		length:  l,
		engines: engines,
	}
}

// NewDecoder creates a new Decoder instance with the provided input values.
// It takes a variadic parameter of any type and returns a pointer to a Decoder.
// The function initializes a slice of decoding engines based on the types of the input values.
//
// Parameters:
//
//	is ...any - A variadic parameter representing the input values of any type.
//
// Returns:
//
//	*Decoder - A pointer to the newly created Decoder instance.
func NewDecoder(is ...any) *Decoder {
	l := len(is)
	engines := make([]decEng, l)
	for i := 0; i < l; i++ {
		engines[i] = getDecEngine(reflect.TypeOf(is[i]))
	}
	return &Decoder{
		length:  l,
		engines: engines,
	}
}

// NewDecoderWithType creates a new Decoder instance with the provided types.
// It takes a variadic number of reflect.Type arguments and returns a pointer to a Decoder.
// Each type is used to generate a corresponding decoding engine which is stored in the Decoder.
//
// Parameters:
//
//	ts - A variadic number of reflect.Type arguments representing the types to decode.
//
// Returns:
//
//	*Decoder - A pointer to a Decoder instance initialized with decoding engines for the provided types.
func NewDecoderWithType(ts ...reflect.Type) *Decoder {
	l := len(ts)
	des := make([]decEng, l)
	for i := 0; i < l; i++ {
		des[i] = getDecEngine(ts[i])
	}
	return &Decoder{
		length:  l,
		engines: des,
	}
}

func (d *Decoder) reset() int {
	index := d.index
	d.index = 0
	d.boolPos = 0
	d.boolBit = 0
	return index
}

// Decode takes a byte slice and a variable number of pointers to variables.
// It decodes the byte slice into the variables.
// the arguments  must be a pointer type
// The return value is the number of bytes that were decoded.
func (d *Decoder) decode(buf []byte, is ...any) int {
	d.buf = buf
	engines := d.engines
	for i := 0; i < len(engines) && i < len(is); i++ {
		engines[i](d, reflect.ValueOf(is[i]).UnsafePointer())
	}
	return d.reset()
}

// DecodeValue takes a byte slice and a variable number of reflect.Values.
// It decodes the byte slice into the reflect.Values.
// The return value is the number of bytes that were decoded.
func (d *Decoder) decodeValue(buf []byte, vs ...reflect.Value) int {
	d.buf = buf
	engines := d.engines
	for i := 0; i < len(engines) && i < len(vs); i++ {
		engines[i](d, unsafe.Pointer(vs[i].UnsafeAddr()))
	}
	return d.reset()
}
