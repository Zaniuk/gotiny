## <font color="#FF4500" >gotiny 只是个玩具，不建议使用。</font>


# gotiny   [![Build status][travis-img]][travis-url] [![License][license-img]][license-url] [![GoDoc][doc-img]][doc-url] [![Go Report Card](https://goreportcard.com/badge/github.com/niubaoshu/gotiny)](https://goreportcard.com/report/github.com/niubaoshu/gotiny)
gotiny es una biblioteca de serialización en Go que se enfoca en la eficiencia. gotiny mejora la eficiencia generando previamente motores de codificación y reduciendo el uso de la biblioteca reflect, alcanzando casi la misma velocidad que las bibliotecas de serialización que generan código.
## hello word 
    package main
    import (
   	    "fmt"
   	    "github.com/niubaoshu/gotiny"
    )
    
    func main() {
   	    src1, src2 := "hello", []byte(" world!")
   	    ret1, ret2 := "", []byte{}
   	    gotiny.Unmarshal(gotiny.Marshal(&src1, &src2), &ret1, &ret2)
   	    fmt.Println(ret1 + string(ret2)) // print "hello world!"
    }

Características
- Muy alta eficiencia, más de 3 veces la velocidad de la biblioteca de serialización gob incluida en Go, y al mismo nivel que los marcos de serialización que generan código, incluso superior a ellos.
- Cero asignaciones de memoria excepto para el tipo map.
- Soporta la codificación de todos los tipos incorporados en Go y tipos personalizados, excepto func y chan.
- Los campos no exportados de los tipos struct se codificarán, se puede configurar para no codificarlos usando etiquetas de Go.
- Conversión de tipos estricta. En gotiny, solo los tipos completamente idénticos se codificarán y decodificarán correctamente.
- Codificación de valores nil con tipo.
- Puede manejar tipos cíclicos, pero no puede codificar valores cíclicos, lo que causará un desbordamiento de pila.
- Todos los tipos que se pueden codificar se decodificarán completamente, sin importar cuál sea el valor original y el valor objetivo.
- Las cadenas de bytes generadas por la codificación no contienen información de tipo, lo que resulta en arrays de bytes muy pequeños.
## No puede manejar valores cíclicos, no soporta referencias cíclicas *TODO*
	type a *a
	var b a
	b = &b

## Instalación
```bash
$ go get -u github.com/niubaoshu/gotiny
```

## Protocolo de codificación

### Tipo booleano
El tipo bool ocupa un bit, el valor verdadero se codifica como 1 y el valor falso se codifica como 0. La primera vez que se encuentra un tipo bool, se asigna un byte y el valor se codifica en el bit menos significativo. La segunda vez que se encuentra, se codifica en el siguiente bit menos significativo. La novena vez que se encuentra un valor bool, se asigna otro byte y se codifica en el bit menos significativo, y así sucesivamente.
### Enteros
- Los tipos uint8 e int8 se codifican como un byte en el siguiente byte de la cadena
- Los tipos uint16, uint32, uint64, uint y uintptr se codifican utilizando el método[Varints](https://developers.google.com/protocol-buffers/docs/encoding#varints)
- Los tipos int16, int32, int64 e int se convierten a un número sin signo utilizando ZigZag y luego se codifican utilizando el método[Varints](https://developers.google.com/protocol-buffers/docs/encoding#varints)

### Float
- Los tipos float32 y float64 se codifican utilizando el método de codificación de tipos de punto flotante de [gob](https://golang.org/pkg/encoding/gob/)
### Complex
- El tipo complex64 se convierte a uint64 y se codifica como uint64.
- El tipo complex128 se codifica por separado para las partes real e imaginaria como float64.

### String
El tipo de cadena se codifica primero convirtiendo la longitud de la cadena a uint64 y luego codificando el array de bytes de la cadena tal cual.
### Pointer
El tipo de puntero se verifica si es nil. Si es nil, se codifica un valor false de tipo bool y se termina. Si no es nil, se codifica un valor true de tipo bool y luego se desreferencia el puntero y se codifica según el tipo desreferenciado.
### Array y Slice
Primero se convierte la longitud a uint64 y se codifica como uint64, luego se codifica cada elemento según su tipo.

### Map
Similar a los arrays y slices, primero se codifica la longitud, luego se codifica cada clave seguida de su valor correspondiente.
### Struct
Todos los campos del struct se codifican según su tipo, independientemente de si son exportados o no. El struct se restaurará estrictamente.
### Tipos que implementan interfaces
- Los tipos que implementan las interfaces BinaryMarshaler/BinaryUnmarshaler del paquete encoding o las interfaces GobEncoder/GobDecoder del paquete gob se codificarán utilizando los métodos implementados.
- Los tipos que implementan la interfaz GoTinySerialize del paquete gotiny se codificarán y decodificarán utilizando los métodos implementados.

## benchmark
[benchmark](https://github.com/niubaoshu/go_serialization_benchmarks)


### License
MIT

[travis-img]: https://travis-ci.org/niubaoshu/gotiny.svg?branch=master
[travis-url]: https://travis-ci.org/niubaoshu/gotiny
[license-img]: http://img.shields.io/badge/license-MIT-green.svg?style=flat-square
[license-url]: http://opensource.org/licenses/MIT
[doc-img]: http://img.shields.io/badge/GoDoc-reference-blue.svg?style=flat-square
[doc-url]: https://godoc.org/github.com/niubaoshu/gotiny
