package gotiny

import (
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type encEng func(*Encoder, unsafe.Pointer) //编码器

var (
	// rt2encEng is a map that associates Go types with their corresponding encoding functions.
	// It uses the reflect.Type as the key and an encEng function as the value.
	// The map includes encoding functions for various primitive types, slices, and structs.
	// The encIgnore function is used for types that should be ignored during encoding.
	rt2encEng = map[reflect.Type]encEng{
		reflect.TypeFor[bool]():       encBool,
		reflect.TypeFor[int]():        encInt,
		reflect.TypeFor[int8]():       encInt8,
		reflect.TypeFor[int16]():      encInt16,
		reflect.TypeFor[int32]():      encInt32,
		reflect.TypeFor[int64]():      encInt64,
		reflect.TypeFor[uint]():       encUint,
		reflect.TypeFor[uint8]():      encUint8,
		reflect.TypeFor[uint16]():     encUint16,
		reflect.TypeFor[uint32]():     encUint32,
		reflect.TypeFor[uint64]():     encUint64,
		reflect.TypeFor[uintptr]():    encUintptr,
		reflect.TypeFor[float32]():    encFloat32,
		reflect.TypeFor[float64]():    encFloat64,
		reflect.TypeFor[complex64]():  encComplex64,
		reflect.TypeFor[complex128](): encComplex128,
		reflect.TypeFor[[]byte]():     encBytes,
		reflect.TypeFor[string]():     encString,
		reflect.TypeFor[time.Time]():  encTime,
		reflect.TypeFor[struct{}]():   encIgnore,
		reflect.TypeOf(nil):           encIgnore,
	}

	// encEngines is an array of encEng functions indexed by reflect.Kind.
	// Each entry in the array corresponds to a specific Go type and its associated encoding function.
	// The encoding functions are used to serialize values of the respective types.
	encEngines = [...]encEng{
		reflect.Bool:       encBool,
		reflect.Int:        encInt,
		reflect.Int8:       encInt8,
		reflect.Int16:      encInt16,
		reflect.Int32:      encInt32,
		reflect.Int64:      encInt64,
		reflect.Uint:       encUint,
		reflect.Uint8:      encUint8,
		reflect.Uint16:     encUint16,
		reflect.Uint32:     encUint32,
		reflect.Uint64:     encUint64,
		reflect.Uintptr:    encUintptr,
		reflect.Float32:    encFloat32,
		reflect.Float64:    encFloat64,
		reflect.Complex64:  encComplex64,
		reflect.Complex128: encComplex128,
		reflect.String:     encString,
	}

	encLock sync.RWMutex
)

// UnusedUnixNanoEncodeTimeType removes the encoding and decoding engine
// for the time.Time type from the rt2encEng and rt2decEng maps, respectively.
// This function is used to disable the encoding and decoding of time.Time
// values using UnixNano format.
func UnusedUnixNanoEncodeTimeType() {
	delete(rt2encEng, reflect.TypeOf((*time.Time)(nil)).Elem())
	delete(rt2decEng, reflect.TypeOf((*time.Time)(nil)).Elem())
}

// getEncEngine retrieves or builds an encoding engine for the given reflect.Type.
// It first attempts to retrieve the engine from a cache using a read lock.
// If the engine is not found in the cache, it acquires a write lock, builds the engine,
// stores it in the cache, and then returns the newly built engine.
//
// Parameters:
//
//	rt - the reflect.Type for which the encoding engine is to be retrieved or built.
//
// Returns:
//
//	encEng - the encoding engine associated with the given reflect.Type.
func getEncEngine(rt reflect.Type) encEng {
	encLock.RLock()
	engine := rt2encEng[rt]
	encLock.RUnlock()
	if engine != nil {
		return engine
	}
	encLock.Lock()
	buildEncEngine(rt, &engine)
	encLock.Unlock()
	return engine
}

// buildEncEngine constructs an encoding engine for the given reflect.Type and assigns it to the provided encEng pointer.
// It first checks if an engine for the type already exists in the cache (rt2encEng).
// If not, it attempts to implement another serializer for the type.
// If neither is successful, it builds the engine based on the kind of the type (e.g., Ptr, Array, Slice, Map, Struct, Interface).
// The function uses deferred calls to recursively build encoding engines for element types as needed.
// Supported kinds include Ptr, Array, Slice, Map, Struct, and Interface.
// Unsupported kinds (Chan, Func, UnsafePointer, Invalid) will cause a panic.
func buildEncEngine(rt reflect.Type, engPtr *encEng) {
	engine := rt2encEng[rt]
	if engine != nil {
		*engPtr = engine
		return
	}

	if engine, _ = implementOtherSerializer(rt); engine != nil {
		rt2encEng[rt] = engine
		*engPtr = engine
		return
	}

	kind := rt.Kind()
	var eEng encEng
	switch kind {
	case reflect.Ptr:
		defer buildEncEngine(rt.Elem(), &eEng)
		engine = func(e *Encoder, p unsafe.Pointer) {
			isNotNil := !isNil(p)
			e.encIsNotNil(isNotNil)
			if isNotNil {
				eEng(e, *(*unsafe.Pointer)(p))
			}
		}
	case reflect.Array:
		et, l := rt.Elem(), rt.Len()
		size := et.Size()
		defer buildEncEngine(et, &eEng)
		engine = func(e *Encoder, p unsafe.Pointer) {
			for i := 0; i < l; i++ {
				eEng(e, unsafe.Add(p, i*int(size)))
			}
		}
	case reflect.Slice:
		et := rt.Elem()
		size := et.Size()
		defer buildEncEngine(et, &eEng)
		engine = func(e *Encoder, p unsafe.Pointer) {
			isNotNil := !isNil(p)
			e.encIsNotNil(isNotNil)
			if isNotNil {
				header := (*sliceHeader)(p)
				l := header.len
				e.encLength(l)
				for i := 0; i < l; i++ {
					eEng(e, unsafe.Add(header.data, i*int(size)))
				}
			}
		}
	case reflect.Map:
		var kEng encEng
		defer buildEncEngine(rt.Key(), &kEng)
		defer buildEncEngine(rt.Elem(), &eEng)
		engine = func(e *Encoder, p unsafe.Pointer) {
			isNotNil := !isNil(p)
			e.encIsNotNil(isNotNil)
			if isNotNil {
				v := reflect.NewAt(rt, p).Elem()
				e.encLength(v.Len())
				iter := v.MapRange()
				for iter.Next() {
					kEng(e, getUnsafePointer(iter.Key()))
					eEng(e, getUnsafePointer(iter.Value()))
				}
			}
		}
	case reflect.Struct:
		fields, offs := getFieldType(rt, 0)
		nf := len(fields)
		fEngines := make([]encEng, nf)
		defer func() {
			for i := 0; i < nf; i++ {
				buildEncEngine(fields[i], &fEngines[i])
			}
		}()
		engine = func(e *Encoder, p unsafe.Pointer) {
			for i := 0; i < len(fEngines) && i < len(offs); i++ {
				fEngines[i](e, unsafe.Add(p, offs[i]))
			}
		}
	case reflect.Interface:
		if rt.NumMethod() > 0 {
			engine = func(e *Encoder, p unsafe.Pointer) {
				isNotNil := !isNil(p)
				e.encIsNotNil(isNotNil)
				if isNotNil {
					v := reflect.ValueOf(*(*interface{ M() })(p))
					et := v.Type()
					e.encString(getNameOfType(et))
					getEncEngine(et)(e, getUnsafePointer(v))
				}
			}
		} else {
			engine = func(e *Encoder, p unsafe.Pointer) {
				isNotNil := !isNil(p)
				e.encIsNotNil(isNotNil)
				if isNotNil {
					v := reflect.ValueOf(*(*any)(p))
					et := v.Type()
					e.encString(getNameOfType(et))
					getEncEngine(et)(e, getUnsafePointer(v))
				}
			}
		}
	case reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Invalid:
		panic("not support " + rt.String() + " type")
	default:
		engine = encEngines[kind]
	}
	rt2encEng[rt] = engine
	*engPtr = engine
}
