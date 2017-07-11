# go-quantize
go-quantize is a highly-optimized and memory-efficient palette generator. It currently implements the Median Cut algorithm, including weighted color priority.

## Performance
go-quantize makes exactly two slice allocations per palette generated, the larger of which is efficiently pooled. It also uses performant direct pixel accesses for certain image types, reducing memory footprint and increasing throughput.

## Benchmarks
go-quantize performs significantly faster than existing quantization libraries:

```
# bench/bench_test.go
BenchmarkQuantize-4          	      30	  43248237 ns/op	  195526 B/op	     258 allocs/op
BenchmarkSoniakeysMedian-4   	       1	1032345645 ns/op	32961336 B/op	 9213800 allocs/op
BenchmarkSoniakeysMean-4     	       1	1062436240 ns/op	52969680 B/op	15692002 allocs/op
BenchmarkEsimov-4            	       1	1186818225 ns/op	35848312 B/op	 8872271 allocs/op
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
