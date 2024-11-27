package gotiny

import (
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type decEng func(*Decoder, unsafe.Pointer) // 解码器

var (
	// rt2decEng is a map that associates Go types with their corresponding decoding
	// engine functions. The map keys are reflect.Type objects representing various
	// Go types, and the values are decEng functions that handle the decoding of
	// values of those types. This map is used to dynamically select the appropriate
	// decoding function based on the type of the value being decoded.
	rt2decEng = map[reflect.Type]decEng{
		reflect.TypeFor[bool]():       decBool,
		reflect.TypeFor[int]():        decInt,
		reflect.TypeFor[int8]():       decInt8,
		reflect.TypeFor[int16]():      decInt16,
		reflect.TypeFor[int32]():      decInt32,
		reflect.TypeFor[int64]():      decInt64,
		reflect.TypeFor[uint]():       decUint,
		reflect.TypeFor[uint8]():      decUint8,
		reflect.TypeFor[uint16]():     decUint16,
		reflect.TypeFor[uint32]():     decUint32,
		reflect.TypeFor[uint64]():     decUint64,
		reflect.TypeFor[uintptr]():    decUintptr,
		reflect.TypeFor[float32]():    decFloat32,
		reflect.TypeFor[float64]():    decFloat64,
		reflect.TypeFor[complex64]():  decComplex64,
		reflect.TypeFor[complex128](): decComplex128,
		reflect.TypeFor[[]byte]():     decBytes,
		reflect.TypeFor[string]():     decString,
		reflect.TypeFor[time.Time]():  decTime,
		reflect.TypeFor[struct{}]():   decIgnore,
		reflect.TypeOf(nil):           decIgnore,
	}

	// decEngines is a slice of decEng functions indexed by reflect.Kind.
	// Each entry in the slice corresponds to a specific Go type and its associated decoding function.
	// The decoding functions are responsible for decoding values of their respective types.
	decEngines = []decEng{
		reflect.Bool:       decBool,
		reflect.Int:        decInt,
		reflect.Int8:       decInt8,
		reflect.Int16:      decInt16,
		reflect.Int32:      decInt32,
		reflect.Int64:      decInt64,
		reflect.Uint:       decUint,
		reflect.Uint8:      decUint8,
		reflect.Uint16:     decUint16,
		reflect.Uint32:     decUint32,
		reflect.Uint64:     decUint64,
		reflect.Uintptr:    decUintptr,
		reflect.Float32:    decFloat32,
		reflect.Float64:    decFloat64,
		reflect.Complex64:  decComplex64,
		reflect.Complex128: decComplex128,
		reflect.String:     decString,
	}
	decLock sync.RWMutex
)

// getDecEngine retrieves or builds a decoding engine for the given reflect.Type.
// It first attempts to retrieve the engine from a cache using a read lock.
// If the engine is not found in the cache, it acquires a write lock and builds the engine.
// The function returns the decoding engine for the specified type.
//
// Parameters:
//
//	reflectType - The reflect.Type for which the decoding engine is to be retrieved or built.
//
// Returns:
//
//	decEng - The decoding engine associated with the specified reflect.Type.
func getDecEngine(reflectType reflect.Type) decEng {
	decLock.RLock()
	engine := rt2decEng[reflectType]
	decLock.RUnlock()
	if engine != nil {
		return engine
	}
	decLock.Lock()
	buildDecEngine(reflectType, &engine)
	decLock.Unlock()
	return engine
}

// buildDecEngine constructs a decoding engine for a given reflect.Type and stores it in engPtr.
// It first checks if the engine already exists in the rt2decEng map. If it does, it assigns the
// existing engine to engPtr and returns. If not, it attempts to implement other serializers
// and stores the engine if successful.
//
// Depending on the kind of the reflect.Type, it builds the appropriate decoding engine:
// - For reflect.Ptr, it handles pointer types and recursively builds the engine for the element type.
// - For reflect.Array, it handles array types and recursively builds the engine for the element type.
// - For reflect.Slice, it handles slice types and recursively builds the engine for the element type.
// - For reflect.Map, it handles map types and recursively builds the engine for the key and value types.
// - For reflect.Struct, it handles struct types and recursively builds the engine for each field.
// - For reflect.Interface, it handles interface types and decodes the underlying concrete type.
//
// If the type is not supported (Chan, Func, Invalid, UnsafePointer), it panics.
//
// Finally, it stores the constructed engine in the rt2decEng map and assigns it to engPtr.
// buildDecEngine constructs a decoding engine for a given reflect.Type and assigns it to the provided decEng pointer.
// It first checks if a decoding engine for the type already exists in the cache (rt2decEng).
// If not, it attempts to implement another serializer for the type.
// Depending on the kind of the type (Ptr, Array, Slice, Map, Struct, Interface), it builds the appropriate decoding engine.
// The function uses deferred calls to recursively build decoding engines for element types in composite types (e.g., Ptr, Array, Slice, Map, Struct).
// Unsupported types (Chan, Func, Invalid, UnsafePointer) will cause a panic.
func buildDecEngine(reflectType reflect.Type, engPtr *decEng) {
	engine, has := rt2decEng[reflectType]
	if has {
		*engPtr = engine
		return
	}

	if _, engine = implementOtherSerializer(reflectType); engine != nil {
		rt2decEng[reflectType] = engine
		*engPtr = engine
		return
	}

	kind := reflectType.Kind()
	var encodingEngine decEng
	switch kind {
	case reflect.Ptr:
		elementType := reflectType.Elem()
		defer buildDecEngine(elementType, &encodingEngine)
		engine = func(d *Decoder, p unsafe.Pointer) {
			if d.decIsNotNil() {
				if isNil(p) {
					//*(*unsafe.Pointer)(p) = unsafe.Pointer(reflect.New(elementType).Elem().UnsafeAddr())
					*(*unsafe.Pointer)(p) = reflect.New(elementType).UnsafePointer()
				}
				encodingEngine(d, *(*unsafe.Pointer)(p))
			} else if !isNil(p) {
				*(*unsafe.Pointer)(p) = nil
			}
		}
	case reflect.Array:
		l, elementType := reflectType.Len(), reflectType.Elem()
		size := elementType.Size()
		defer buildDecEngine(elementType, &encodingEngine)
		engine = func(d *Decoder, p unsafe.Pointer) {
			for i := 0; i < l; i++ {
				encodingEngine(d, unsafe.Add(p, i*int(size)))
			}
		}
	case reflect.Slice:
		elementType := reflectType.Elem()
		size := elementType.Size()
		defer buildDecEngine(elementType, &encodingEngine)
		engine = func(d *Decoder, p unsafe.Pointer) {
			header := (*sliceHeader)(p)
			if d.decIsNotNil() {
				l := d.decLength()
				if isNil(p) || header.cap < l {
					*header = sliceHeader{data: reflect.MakeSlice(reflectType, l, l).UnsafePointer(), len: l, cap: l}
				} else {
					header.len = l
				}
				for i := 0; i < l; i++ {
					encodingEngine(d, unsafe.Add(header.data, uintptr(i)*size))
				}
			} else if !isNil(p) {
				*header = sliceHeader{data: nil, len: 0, cap: 0}
			}
		}
	case reflect.Map:
		keyType, valueType := reflectType.Key(), reflectType.Elem()
		var kEng, vEng decEng
		defer buildDecEngine(keyType, &kEng)
		defer buildDecEngine(valueType, &vEng)
		engine = func(d *Decoder, p unsafe.Pointer) {
			if d.decIsNotNil() {
				l := d.decLength()
				v := reflect.NewAt(reflectType, p).Elem()
				if isNil(p) {
					v = reflect.MakeMapWithSize(reflectType, l)
					*(*unsafe.Pointer)(p) = v.UnsafePointer()
				}
				key, val := reflect.New(keyType).Elem(), reflect.New(valueType).Elem()
				for i := 0; i < l; i++ {
					kEng(d, unsafe.Pointer(key.UnsafeAddr()))
					vEng(d, unsafe.Pointer(val.UnsafeAddr()))
					v.SetMapIndex(key, val)
					key.SetZero()
					val.SetZero()
				}
			} else if !isNil(p) {
				*(*unsafe.Pointer)(p) = nil
			}
		}
	case reflect.Struct:
		fields, offs := getFieldType(reflectType, 0)
		nf := len(fields)
		fEngines := make([]decEng, nf)
		defer func() {
			for i := 0; i < nf; i++ {
				buildDecEngine(fields[i], &fEngines[i])
			}
		}()
		engine = func(d *Decoder, p unsafe.Pointer) {
			for i := 0; i < nf && i < len(offs); i++ {
				fEngines[i](d, unsafe.Add(p, offs[i]))
			}
		}
	case reflect.Interface:
		engine = func(d *Decoder, p unsafe.Pointer) {
			if d.decIsNotNil() {
				var name string
				decString(d, unsafe.Pointer(&name))
				elementType, has := name2type[name]
				if !has {
					panic("unknown typ:" + name)
				}
				v := reflect.NewAt(reflectType, p).Elem()
				if v.IsNil() || v.Elem().Type() != elementType {
					ev := reflect.New(elementType).Elem()
					getDecEngine(elementType)(d, getUnsafePointer(ev))
					v.Set(ev)
				} else {
					getDecEngine(elementType)(d, getUnsafePointer(v.Elem()))
				}
			} else if !isNil(p) {
				*(*unsafe.Pointer)(p) = nil
			}
		}
	case reflect.Chan, reflect.Func, reflect.Invalid, reflect.UnsafePointer:
		panic("not support " + reflectType.String() + " type")
	default:
		engine = decEngines[kind]
	}
	rt2decEng[reflectType] = engine
	*engPtr = engine
}
