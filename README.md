# go-quantize
go-quantize is a highly-optimized and memory-efficient palette generator. It currently implements the Median Cut algorithm, including weighted color priority.

## Performance
go-quantize makes exactly two slice allocations per palette generated, the larger of which is efficiently pooled. It also uses performant direct pixel accesses for certain image types, reducing memory footprint and increasing throughput.

## Benchmarks
go-quantize performs significantly faster than existing quantization libraries:

```
# bench/bench_test.go
BenchmarkQuantize-8          	      50	  29169882 ns/op	  134954 B/op	     259 allocs/op
BenchmarkSoniakeysMedian-8   	       3	 489195141 ns/op	 3479624 B/op	     782 allocs/op
BenchmarkSoniakeysMean-8     	       3	 358811870 ns/op	 2755680 B/op	     262 allocs/op
BenchmarkEsimov-8            	       2	 620675784 ns/op	35848320 B/op	 8872271 allocs/op
```

## Example Usage
```go
file, err := os.Open("test_image.jpg")
if err != nil {
    fmt.Println("Couldn't open test file")
    return
}
i, _, err := image.Decode(file)
if err != nil {
    fmt.Println("Couldn't decode test file")
    return
}
q := MedianCutQuantizer{}
p := q.Quantize(make([]color.Color, 0, 256), i)
fmt.Println(p)
```
