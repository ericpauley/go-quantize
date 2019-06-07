module github.com/ericpauley/go-quantize/quantize/bench
// Note: We use a separate go.mod file here because comparison libraries should not be in top-level dependencies
go 1.12

require (
	github.com/ericpauley/go-quantize v0.0.0-20180803033130-bfdbba883ede
	github.com/esimov/colorquant v1.0.0
	github.com/soniakeys/quant v1.0.0
)

replace github.com/ericpauley/go-quantize => ../..
