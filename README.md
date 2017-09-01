# places
fast and primitive templating for go

[![Build Status](https://secure.travis-ci.org/metakeule/places.png)](http://travis-ci.org/metakeule/places)

If you need to simply replace placeholders in a template without escaping or logic, places might be for you.

Performance
-----------

Runing benchmarks in the benchmark directory, I get the following results (go1.8, linux/amd64):

replacing 2 placeholders that occur 2500x in the template


    BenchmarkNaive    1000      1263008 ns/op      4 allocs/op   4x  (    strings.Replace)   
    BenchmarkNaive2   2000      1042809 ns/op     13 allocs/op   3x  (    strings.Replacer)  
    BenchmarkReg       100     15384938 ns/op     25 allocs/op  48x  (    regexp.ReplaceAllStringFunc)  
    BenchmarkByte     2000      1053584 ns/op      2 allocs/op   3x  (    bytes.Replace)  
    BenchmarkTemplate  500      3608738 ns/op  10001 allocs/op  11x  (    template.Execute)  
    BenchmarkPlaces   5000       315520 ns/op      0 allocs/op   1x  (    places.ReplaceString)
                                  


replacing 5000 placeholders that occur 1x in the template

    BenchmarkNaiveM        1    1954624037 ns/op     10001 allocs/op  3481.58x  (    strings.Replace)
    BenchmarkNaive2M     500       3467816 ns/op     11007 allocs/op     6.18x  (    strings.Replacer)
    BenchmarkRegM        100      17515006 ns/op        26 allocs/op    31.20x  (    regexp.ReplaceAllStringFunc)
    BenchmarkByteM      2000        823151 ns/op         2 allocs/op     1.47x  (    bytes.Replace)
    BenchmarkTemplateM   500       3736012 ns/op     10001 allocs/op     6.65x  (    template.Execute)
    BenchmarkPlacesM    2000        561419 ns/op         0 allocs/op     1.00x  (    places.ReplaceString)
           

replacing 2 placeholders that occur 1x in the template, parsing template each time (you should not do this until you need it)


    BenchmarkOnceNaive    1000   1242754 ns/op      4 allocs/op     1.16x  (    strings.Replace)  
    BenchmarkOnceNaive2   2000   1069273 ns/op     13 allocs/op     1.00x  (    strings.Replacer) 
    BenchmarkOnceReg       100  15208299 ns/op     25 allocs/op    14.22x  (    regexp.ReplaceAllStringFunc)  
    BenchmarkOnceByte     1000   1208239 ns/op      4 allocs/op     1.13x  (    bytes.Replace)  
    BenchmarkOnceTemplate   50  31201995 ns/op  55054 allocs/op    29.18x  (    template.Execute)  
    BenchmarkOncePlaces   2000   1112133 ns/op     26 allocs/op     1.04x  (    places.ReplaceString)  

Usage
-----

```go
package main

import (
    "bytes"
    "fmt"
    "net/http"
    "github.com/metakeule/places"
)

// parse the template once and reuse it to speed up replacement
var template = places.NewTemplate([]byte("<@name@>: <@animal@>"))

// you can also create the replacements ad hoc in the handler
// also map[string]string, map[string]io.ReadSeeker etc. possible
var replacements = map[string][]byte{
    "animal": []byte("Duck"),
    "name": []byte("Donald"),
}

func handle (wr http.ResponseWriter rq *http.Request) {
    // no error checking on write (for performance)
    // if you need error checking wrap the io.Writer
    template.ReplaceBytes(wr, replacements)
}

func main() {
    http.ListenAndServe("localhost:8080", http.HandlerFunc(handle))
}
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

