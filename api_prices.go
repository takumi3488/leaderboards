package main

// API prices use ordinary, uncached input and output rates.
type apiPrice struct {
	Input  float64
	Output float64
}

func (p apiPrice) usdPerMTok() float64 {
	return (5*p.Input + p.Output) / 6
}

const cnyPerUSD = 6.7787

var handAPIPrices = map[string]apiPrice{
	"gpt-6-astra":      {Input: 10, Output: 50},
	"gpt-6-sol":        {Input: 2, Output: 10},
	"gpt-6.1-sol":      {Input: 2, Output: 10},
	"gpt-6-luna":       {Input: 0.1, Output: 0.5},
	"claude-fable-5.2": {Input: 8, Output: 40},
	"claude-opus-4.8":  {Input: 5, Output: 25},
}

var handReferenceAPIPrices = map[string]float64{
	"gpt-6-astra":      1.47,
	"gpt-6-sol":        0.294,
	"gpt-6.1-sol":      0.197,
	"gpt-6-luna":       0.0147,
	"claude-fable-5.2": 0.644,
	"claude-opus-4.8":  0.76625,
}
