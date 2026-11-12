package main

import (
	"math"
	"os"
	"strings"
	"testing"
)

func TestApplySubscriptionsExactMatchingAndPricing(t *testing.T) {
	data, err := os.ReadFile("testdata/subscriptions.md")
	if err != nil {
		t.Fatal(err)
	}
	subscriptions, _, err := parseSubscriptions(data)
	if err != nil {
		t.Fatal(err)
	}
	benchmarks := []benchmark{{Points: []point{
		{Model: "anthropic/CLAUDE_SONNET_4.6", APICost: 100, Trials: 20},
		{Model: "Sonnet 5", APICost: 100},
		{Model: "Opus 5.5", APICost: 100},
		{Model: "openai/gpt.6_luna", APICost: 0},
		{Model: "Claude Fable 5.2", APICost: 100},
		{Model: "Gemini 4 Carbon", APICost: 100},
		{Model: "claude-sonnet-5-preview", APICost: 100},
	}}}
	applySubscriptions(benchmarks, subscriptions)
	points := benchmarks[0].Points

	if got, want := offerCost(points[0].Offers, "Claude Pro"), 100*20/972.32; math.IsNaN(got) || math.Abs(got-want) > 1e-12 {
		t.Errorf("Claude Pro cost = %.12f, want %.12f", got, want)
	}
	if got, want := offerPlanNames(points[1].Offers), []string{"Claude Max", "Claude Pro", "Kiro", "Kiro Power"}; !equalStrings(got, want) {
		t.Errorf("Sonnet 5 plans = %v, want %v", got, want)
	}
	if got, want := offerCost(points[1].Offers, "Claude Max"), 100*200/17557.85; math.IsNaN(got) || math.Abs(got-want) > 1e-12 {
		t.Errorf("Claude Max cost = %.12f, want %.12f", got, want)
	}
	if got := offerPlanNames(points[2].Offers); !equalStrings(got, []string{"Claude Max"}) {
		t.Errorf("bare Opus 5.5 plans = %v, want [Claude Max]", got)
	}
	if got := offerPlanNames(points[3].Offers); !equalStrings(got, []string{"ChatGPT Plus"}) {
		t.Errorf("cosmetic GPT match plans = %v, want [ChatGPT Plus]", got)
	}
	for _, index := range []int{4, 5, 6} {
		if len(points[index].Offers) != 0 {
			t.Errorf("%q unexpectedly matched offers %v", points[index].Model, points[index].Offers)
		}
	}
}

func TestApplySubscriptionsZeroAPICostAndDuplicateAliases(t *testing.T) {
	plans := []subscription{{Name: "One", MonthlyUSD: 10, APIValueUSD: 20, Models: []string{"anthropic/claude-opus-5-5", "claude.opus_5 5"}}, {Name: "Two", MonthlyUSD: 5, APIValueUSD: 10, Models: []string{"claude-opus-5-5"}}}
	benchmarks := []benchmark{{Points: []point{{Model: "Opus 5.5", APICost: 0}}}}
	applySubscriptions(benchmarks, plans)
	offers := benchmarks[0].Points[0].Offers
	if len(offers) != 2 {
		t.Fatalf("got %d offers, want one per matching plan: %v", len(offers), offers)
	}
	if got, want := offerPlanNames(offers), []string{"One", "Two"}; !equalStrings(got, want) {
		t.Errorf("zero-cost offers = %v, want tie-sorted %v", got, want)
	}
	for _, got := range offers {
		if got.Cost != 0 {
			t.Errorf("zero API cost produced subscription cost %v", got.Cost)
		}
	}
}

func TestParseSubscriptionsRejectsInvalidOverviewAndDetails(t *testing.T) {
	base := validSubscriptionMarkdown()
	tests := []struct {
		name string
		data string
	}{
		{"zero API value", strings.Replace(base, "$25.00", "$0.00", 1)},
		{"malformed API value", strings.Replace(base, "$25.00", "$nope", 1)},
		{"empty API value", strings.Replace(base, "$25.00", "", 1)},
		{"missing API value column", strings.Replace(base, " | Avg API Value (USD)", "", 1)},
		{"missing details block", strings.Replace(base, validDetailsBlock(), "", 1)},
		{"unlinked details plan", strings.Replace(base, "Alpha - Long - per-model averages", "Other Plan - per-model averages", 1)},
		{"missing model table", strings.Replace(base, "| Model | Periods |", "| Name | Periods |", 1)},
		{"zero months", strings.Replace(base, "| Alpha - Long | 1 |", "| Alpha - Long | 0 |", 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := parseSubscriptions([]byte(test.data)); err == nil {
				t.Fatal("parseSubscriptions succeeded, want an error")
			}
		})
	}
}

func TestCanonicalModelCosmeticEquivalence(t *testing.T) {
	pairs := [][2]string{
		{"anthropic/Claude_Sonnet_4.6", "claude-sonnet-4-6"},
		{"chatgpt/gpt_5.4_mini", "gpt-5.4-mini"},
		{"OpenAI/gpt.6 luna", "gpt-6-luna"},
		{"Haiku 4.5", "claude-haiku-4-5"},
	}
	for _, pair := range pairs {
		if got, want := canonicalModel(pair[0]), canonicalModel(pair[1]); got != want {
			t.Errorf("canonicalModel(%q) = %q, want %q", pair[0], got, want)
		}
	}
	for _, pair := range [][2]string{
		{"claude-fable-5-2", "claude-fable-5-1"},
		{"gpt-6-1", "gpt-61"},
		{"gpt-6-preview", "gpt-6"},
	} {
		if canonicalModel(pair[0]) == canonicalModel(pair[1]) {
			t.Fatalf("different model versions/suffixes collapsed: %v", pair)
		}
	}
}

func validSubscriptionMarkdown() string {
	return `*Last updated: 2026-11-09*

## Subscription Performance Overview

| Subscription | Months | Avg Cost (USD) | Avg API Value (USD) |
|--------------|--------|----------------|---------------------|
| Alpha - Long | 1 | $10.00 | $25.00 |

` + validDetailsBlock()
}

func validDetailsBlock() string {
	return `<details>
<summary><strong>Alpha - Long - per-model averages (1 period)</strong></summary>

| Model | Periods |
|-------|---------|
| claude-opus-5 | 1 |

</details>
`
}

func offerCost(offers []offer, plan string) float64 {
	for _, candidate := range offers {
		if candidate.Plan == plan {
			return candidate.Cost
		}
	}
	return math.NaN()
}

func offerPlanNames(offers []offer) []string {
	plans := make([]string, len(offers))
	for i, candidate := range offers {
		plans[i] = candidate.Plan
	}
	return plans
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
