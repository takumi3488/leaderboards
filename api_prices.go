package main

// These short-context list prices are blended for the standard workload below.
type apiPrice struct {
	CacheRead  float64
	FreshInput float64
	Output     float64
}

func (p apiPrice) blendedUSDPerMTok() float64 {
	return 0.97*p.CacheRead + 0.025*p.FreshInput + 0.005*p.Output
}

// FreshInput is the 5-minute cache-write price for Anthropic models.
var handAPIPrices = map[string]apiPrice{
	"gpt-6-astra":      {CacheRead: 1, FreshInput: 10, Output: 50},
	"gpt-6-sol":        {CacheRead: 0.2, FreshInput: 2, Output: 10},
	"gpt-6.1-sol":      {CacheRead: 0.1, FreshInput: 2, Output: 10},
	"gpt-6-luna":       {CacheRead: 0.01, FreshInput: 0.1, Output: 0.5},
	"gpt-5.5":          {CacheRead: 0.5, FreshInput: 5, Output: 30},
	"claude-fable-5.2": {CacheRead: 0.2, FreshInput: 10, Output: 40},
	"claude-opus-4.8":  {CacheRead: 0.5, FreshInput: 6.25, Output: 25},
}
