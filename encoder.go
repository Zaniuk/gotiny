package gotiny

import (
	"reflect"
)

// Encoder is a structure that holds the state and buffer for encoding operations.
// It contains the following fields:
// - buf: a byte slice that serves as the encoded target array.
// - off: an integer representing the current offset in the buffer.
// - boolPos: an integer indicating the index of the next boolean value to be set in the buffer (buf).
// - boolBit: a byte representing the bit position of the next boolean value to be set in buf[boolPos].
// - engines: a slice of encEng, which are the encoding engines used for encoding operations.
// - length: an integer representing the length of the encoded data.
type Encoder struct {
	buf     []byte // encoded target array
	off     int
	boolPos int  // the index of the next bool to be set in buf, i.e., buf[boolPos]
	boolBit byte // the bit position of the next bool to be set in buf[boolPos]

	engines []encEng
	length  int
}

/*
Marshal serializes the value pointed to by the incoming pointer.
The argument must be a pointer, similar to the form &value
which is serializing value. If value itself is a pointer,
then you can pass in value directly,
which serializes the value pointed to by value.
*/
func Marshal(ps ...any) []byte {
	return NewEncoderWithPtr(ps...).encode(ps...)
}

// Create an encoder for the types pointed to by ps
func NewEncoderWithPtr(ps ...any) *Encoder {
	l := len(ps)
	engines := make([]encEng, l)
	for i := 0; i < l; i++ {
		rt := reflect.TypeOf(ps[i])
		if rt.Kind() != reflect.Ptr {
			panic("must a pointer type!")
		}
		engines[i] = getEncEngine(rt.Elem())
	}
	return &Encoder{
		length:  l,
		engines: engines,
	}
}

// Create an encoder for the types of is
func NewEncoder(is ...any) *Encoder {
	l := len(is)
	engines := make([]encEng, l)
	for i := 0; i < l; i++ {
		engines[i] = getEncEngine(reflect.TypeOf(is[i]))
	}
	return &Encoder{
		length:  l,
		engines: engines,
	}
}

func NewEncoderWithType(ts ...reflect.Type) *Encoder {
	l := len(ts)
	engines := make([]encEng, l)
	for i := 0; i < l; i++ {
		engines[i] = getEncEngine(ts[i])
	}
	return &Encoder{
		length:  l,
		engines: engines,
	}
}

// The input parameter is a pointer to the value to be encoded
func (e *Encoder) encode(is ...any) []byte {
	engines := e.engines
	for i := 0; i < len(engines) && i < len(is); i++ {
		engines[i](e, reflect.ValueOf(is[i]).UnsafePointer())
	}
	return e.reset()
}

// vs holds the values to be encoded
func (e *Encoder) encodeValue(vs ...reflect.Value) []byte {
	engines := e.engines
	for i := 0; i < len(engines) && i < len(vs); i++ {
		engines[i](e, getUnsafePointer(vs[i]))
	}
	return e.reset()
}

// AppendTo appends the encoded data to buf
func (e *Encoder) AppendTo(buf []byte) {
	e.off = len(buf)
	e.buf = buf
}

// reset resets the encoder's buffer and boolean position tracking.
// It returns the original buffer before the reset.
// The buffer is truncated to the current offset, and boolean position
// tracking variables are reset to their initial state.
func (e *Encoder) reset() []byte {
	buf := e.buf
	e.buf = buf[:e.off]
	e.boolBit = 0
	e.boolPos = 0
	return buf
}
