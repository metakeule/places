# places
fast and primitive templating for go

[![Build Status](https://secure.travis-ci.org/metakeule/places.png)](http://travis-ci.org/metakeule/places)

If you need to simply replace placeholders in a template without escaping or logic, places might be for you.

Performance
-----------

Runing benchmarks in the benchmark directory, I get the following results (go1.4, linux/amd64):

replacing 2 placeholders that occur 2500x in the template

    BenchmarkNaive     500    2780308 ns/op     4 allocs/op   7,54x (strings.Replace)
    BenchmarkNaive2   2000    1098662 ns/op    13 allocs/op   2,98x (strings.Replacer)
    BenchmarkReg        50   22916309 ns/op  5024 allocs/op  62,18x (regexp.ReplaceAllStringFunc)
    BenchmarkByte     1000    2087658 ns/op     4 allocs/op   5,66x (bytes.Replace)
    BenchmarkTemplate  300    5566514 ns/op 15002 allocs/op  15,10x (template.Execute)
    BenchmarkPlaces   3000     368525 ns/op     0 allocs/op   1,00x (places.ReplaceString)
                                


replacing 5000 placeholders that occur 1x in the template

    BenchmarkNaiveM       1 4286720867 ns/op 10000 allocs/op 6673,23x (strings.Replace)
    BenchmarkNaive2M    500    4019384 ns/op 11007 allocs/op    6,26x (strings.Replacer)
    BenchmarkRegM        50   27298490 ns/op  5025 allocs/op   42,50x (regexp.ReplaceAllStringFunc)
    BenchmarkByteM     1000    1626838 ns/op     4 allocs/op    2,53x (bytes.Replace)
    BenchmarkTemplateM  300    5667141 ns/op 15002 allocs/op    8,82x (template.Execute)
    BenchmarkPlacesM   2000     642376 ns/op     0 allocs/op    1,00x (places.ReplaceString)
                                

replacing 2 placeholders that occur 1x in the template, parsing template each time (you should not do this until you need it)

    BenchmarkOnceNaive    500   2759530 ns/op     4 allocs/op   2,45x (strings.Replace)
    BenchmarkOnceNaive2  2000   1127832 ns/op    13 allocs/op   1,00x (strings.Replacer)
    BenchmarkOnceReg       50  23076371 ns/op  5024 allocs/op  20,46x (regexp.ReplaceAllStringFunc)
    BenchmarkOnceByte    1000   2336374 ns/op     6 allocs/op   2,07x (bytes.Replace)
    BenchmarkOnceTemplate   2 917598881 ns/op 60058 allocs/op 813,60x (template.Execute)
    BenchmarkOncePlaces  1000   1808015 ns/op    26 allocs/op   1,60x (places.ReplaceString)

Usage
-----

```go
package main

import (
    "bytes"
    "fmt"
    "github.com/metakeule/places"
)

func main() {
    // parse the template once
    template := places.NewTemplate([]byte("<@name@>: <@animal@>")) 
    
    // reuse it to speed up replacement
    var buffer bytes.Buffer
    template.ReplaceString(&buffer, map[string]string{"animal": "Duck","name": "Donald"})

    // there are alternative methods for Bytes, io.ReadSeeker etc.
    
    // after the replacement you may use the buffer methods Bytes(), String(), Write() or WriteTo()
    // and reuse the same buffer after calling buffer.Reset()
    fmt.Println(buffer.String())
}
```


results in

```
Donald: Duck
```

Documentation (GoDoc)
---------------------

see https://godoc.org/github.com/metakeule/places


Status
------

The package is stable and ready for consumption.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.

see LICENSE file.

