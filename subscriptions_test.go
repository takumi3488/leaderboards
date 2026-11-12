package main

import (
	"os"
	"testing"
)

func TestApplySubscriptionsUsesTokensAndMeteredRatios(t *testing.T) {
	pricing := readPricingFixture(t)
	benchmarks := []benchmark{{Points: []point{
		{Model: "GPT-6 Astra", Tokens: 500_000, APICost: 99},
		{Model: "gpt-5.6-sol", APICost: 1.176},
	}}}
	plans := applySubscriptions(benchmarks, pricing.Points)
	points := benchmarks[0].Points
	if got, want := offerPlanNames(points[0].Offers), []string{"Astra Plan", "Premium Plan"}; !equalStrings(got, want) {
		t.Fatalf("offers sorted by cost = %v, want %v", got, want)
	}
	assertNear(t, points[0].Offers[0].Cost, 3, 1e-12)
	assertNear(t, points[0].Offers[1].Cost, 4, 1e-12)
	assertNear(t, points[0].Offers[0].USDPerMTok, 6, 1e-12)
	if points[0].Offers[0].Confidence != "high" {
		t.Errorf("duplicate did not keep cheaper row's confidence: %+v", points[0].Offers[0])
	}
	if len(points[1].Offers) != 1 || points[1].Offers[0].Plan != "Measured English Plan" {
		t.Fatalf("metered ratio offer = %+v", points[1].Offers)
	}
	assertNear(t, points[1].Offers[0].Cost, 0.588, 1e-12)
	if len(plans) != 3 {
		t.Fatalf("used plans = %+v, want only three plans that produced offers", plans)
	}
	if got, want := []string{plans[0].Name, plans[1].Name, plans[2].Name}, []string{"Astra Plan", "Measured English Plan", "Premium Plan"}; !equalStrings(got, want) {
		t.Errorf("unique plans sorted by monthly price and name = %v, want %v", got, want)
	}
}

func TestApplySubscriptionsUsesHandAPIPricesAndSkipsMissingFallback(t *testing.T) {
	pricing := readPricingFixture(t)
	benchmarks := []benchmark{{Points: []point{
		{Model: "gpt-6-astra", APICost: 1},
		{Model: "model-without-api-price", APICost: 1},
	}}}
	applySubscriptions(benchmarks, pricing.Points)
	points := benchmarks[0].Points
	if len(points[0].Offers) != 2 {
		t.Fatalf("hand-table offers = %+v, want two offers", points[0].Offers)
	}
	assertNear(t, points[0].Offers[0].Cost, 6/1.47, 1e-12)
	if got := apiPrices(append(pricing.Points, pricingPoint{Billing: "metered", Model: "gpt-6-astra", RealUSDPerMTok: 0.123}))["gpt-6-astra"]; got != 0.123 {
		t.Errorf("metered API price = %g, want source rate 0.123 over the hand table", got)
	}
	if len(points[1].Offers) != 0 {
		t.Errorf("model without tokens or API price got offers: %+v", points[1].Offers)
	}
}

func TestAPIPricesUseNonAnthropicListBlendFallback(t *testing.T) {
	points := []pricingPoint{
		{Model: "list-only-model", Vendor: "OpenAI", ListBlendedUSDPerMTok: 0.123},
		{Model: "claude-opus-5.5", Vendor: "Anthropic", ListBlendedUSDPerMTok: 0.456},
	}
	prices := apiPrices(points)
	assertNear(t, prices["list-only-model"], 0.123, 1e-12)
	if _, ok := prices["claude-opus-5-5"]; ok {
		t.Errorf("used Anthropic list blend without cache-write pricing: %v", prices["claude-opus-5-5"])
	}
}

func TestScenarioPlanRowsAreSkipped(t *testing.T) {
	pricing := readPricingFixture(t)
	base := pricingPoint{Billing: "subscription", PriceUSD: 10, Confidence: "high"}
	rows := []pricingPoint{
		{Billing: base.Billing, Plan: "GLM Pro 忙时", Vendor: "Zhipu", Model: "gpt-6-astra", PriceUSD: base.PriceUSD, RealUSDPerMTok: 0.1, Confidence: base.Confidence},
		{Billing: base.Billing, Plan: "GLM Pro 闲时", Vendor: "Zhipu", Model: "gpt-6-astra", PriceUSD: base.PriceUSD, RealUSDPerMTok: 0.2, Confidence: base.Confidence},
		{Billing: base.Billing, Plan: "GLM Pro 中间值", Vendor: "Zhipu", Model: "gpt-6-astra", PriceUSD: base.PriceUSD, RealUSDPerMTok: 0.5, Confidence: base.Confidence},
		{Billing: base.Billing, Plan: "MiMo Token Plan Lite 夜间0.8×", PlanEN: new("MiMo Token Plan Lite (Night 0.8x)"), Vendor: "Xiaomi", Model: "mimo-v2.6-pro", PriceUSD: base.PriceUSD, RealUSDPerMTok: 0.1, Confidence: base.Confidence},
		{Billing: base.Billing, Plan: "MiMo Token Plan Lite 日间", PlanEN: new("MiMo Token Plan Lite (Day)"), Vendor: "Xiaomi", Model: "mimo-v2.6-pro", PriceUSD: base.PriceUSD, RealUSDPerMTok: 0.5, Confidence: base.Confidence},
	}
	pricing.Points = append(pricing.Points, rows...)
	benchmarks := []benchmark{{Points: []point{
		{Model: "gpt-6-astra", Tokens: 1_000_000},
		{Model: "mimo-v2.6-pro", Tokens: 1_000_000},
	}}}
	applySubscriptions(benchmarks, pricing.Points)
	if got, want := offerPlanNames(benchmarks[0].Points[0].Offers), []string{"GLM Pro 中间值", "Astra Plan", "Premium Plan"}; !equalStrings(got, want) {
		t.Errorf("busy/idle rows should be skipped while midpoint is retained: %v", got)
	}
	if got, want := offerPlanNames(benchmarks[0].Points[1].Offers), []string{"MiMo Token Plan Lite (Day)"}; !equalStrings(got, want) {
		t.Errorf("MiMo night row should be skipped while daytime remains: %v", got)
	}
}
func TestParseRealPricingValidationAndSkippedRows(t *testing.T) {
	fixture, err := os.ReadFile("testdata/real-api-pricing.json")
	if err != nil {
		t.Fatal(err)
	}
	pricing, err := parseRealPricing(fixture)
	if err != nil {
		t.Fatal(err)
	}
	benchmarks := []benchmark{{Points: []point{{Model: "gpt-6-astra", Tokens: 1_000_000}}}}
	applySubscriptions(benchmarks, pricing.Points)
	if got, want := offerPlanNames(benchmarks[0].Points[0].Offers), []string{"Astra Plan", "Premium Plan"}; !equalStrings(got, want) {
		t.Fatalf("unmetered/invalid rows were not skipped: offers=%v", got)
	}
	for _, raw := range []string{
		`{"generatedAt":"2026-11-10","points":[]}`,
		`{"generatedAt":"2026-11-10","points":[{"billing":"subscription","plan":"Bad","vendor":"OpenAI","model":"x","price_usd":10,"real_usd_per_mtok":0,"confidence":"high"}]}`,
		`{"generatedAt":"bad-date","points":[{"billing":"subscription","plan":"Okay","vendor":"OpenAI","model":"x","price_usd":10,"real_usd_per_mtok":1,"confidence":"high"}]}`,
	} {
		if _, err := parseRealPricing([]byte(raw)); err == nil {
			t.Errorf("accepted unusable real-api-pricing JSON: %s", raw)
		}
	}
}

func TestBlendAPIPricesUsesStandardWorkload(t *testing.T) {
	assertNear(t, (apiPrice{CacheRead: 0.4, FreshInput: 4, Output: 20}).blendedUSDPerMTok(), 0.588, 1e-12)
	assertNear(t, (apiPrice{CacheRead: 0.5, FreshInput: 6.25, Output: 25}).blendedUSDPerMTok(), 0.76625, 1e-12)
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

func readPricingFixture(t *testing.T) pricingData {
	t.Helper()
	data, err := os.ReadFile("testdata/real-api-pricing.json")
	if err != nil {
		t.Fatal(err)
	}
	pricing, err := parseRealPricing(data)
	if err != nil {
		t.Fatal(err)
	}
	return pricing
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

func TestRealPricingSourcePointUsesEnglishName(t *testing.T) {
	point := pricingPoint{Plan: "Chinese", PlanEN: new("English")}
	if got := displayPlan(point); got != "English" {
		t.Errorf("displayPlan = %q, want English", got)
	}
	point.PlanEN = new(" ")
	if got := displayPlan(point); got != "Chinese" {
		t.Errorf("displayPlan = %q, want fallback name", got)
	}
}
