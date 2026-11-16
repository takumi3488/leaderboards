package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestApplySubscriptionsUsesNoCachePricesAndPreservesSourceMetrics(t *testing.T) {
	pricing := readPricingFixture(t)
	tokens := 500_000.0
	zeroTokens := 0.0
	benchmarks := []benchmark{
		{Points: []point{{Model: "GPT-6 Astra", Tokens: &tokens, APICost: 99}}},
		{Points: []point{{Model: "gpt-6-astra", Tokens: &zeroTokens, APICost: 1}}},
		{Points: []point{{Model: "gpt-6-astra", APICost: 17}}},
		{Points: []point{{Model: "gpt-5.6-sol", APICost: 1.176}}},
		{Points: []point{{Model: "model-without-api-price", Tokens: &tokens}}},
		{Points: []point{{Model: "model-without-api-price"}}},
	}
	plans := applySubscriptions(benchmarks, pricing.Points)
	expectedAstraAPI := 100.0 / 6
	expectedAstraOffer := 6.0 / 1.47 * expectedAstraAPI
	expectedPremiumOffer := 8.0 / 1.47 * expectedAstraAPI
	wantPlans := []string{"Astra Plan", "Premium Plan"}
	for i, bench := range benchmarks[:3] {
		point := bench.Points[0]
		if got := offerPlanNames(point.Offers); !equalStrings(got, wantPlans) {
			t.Fatalf("benchmark %d offers sorted by unit price = %v, want %v", i, got, wantPlans)
		}
		if point.APIUSDPerMTok == nil {
			t.Fatalf("API unit price missing for benchmark %d", i)
		}
		assertNear(t, *point.APIUSDPerMTok, expectedAstraAPI, 1e-12)
		for j, want := range []float64{expectedAstraOffer, expectedPremiumOffer} {
			if point.Offers[j].USDPerMTok == nil {
				t.Fatalf("offer %q has no estimated unit price", point.Offers[j].Plan)
			}
			assertNear(t, *point.Offers[j].USDPerMTok, want, 1e-12)
		}
	}
	if benchmarks[0].Points[0].Offers[0].Confidence != "high" {
		t.Errorf("duplicate did not keep cheaper row's confidence: %+v", benchmarks[0].Points[0].Offers[0])
	}
	if *benchmarks[0].Points[0].Tokens != tokens || benchmarks[0].Points[0].APICost != 99 ||
		*benchmarks[1].Points[0].Tokens != 0 || benchmarks[1].Points[0].APICost != 1 ||
		benchmarks[2].Points[0].Tokens != nil || benchmarks[2].Points[0].APICost != 17 {
		t.Errorf("raw benchmark usage metadata changed: %+v", benchmarks[:3])
	}
	if got := benchmarks[0].Points[0].APIRolloutCost; got == nil {
		t.Fatal("measured API rollout cost is nil")
	} else {
		assertNear(t, *got, tokens/1_000_000*expectedAstraAPI, 1e-12)
	}
	if got := benchmarks[1].Points[0].APIRolloutCost; got == nil || *got != 0 {
		t.Errorf("zero-token API rollout cost = %v, want zero", got)
	}
	if got := benchmarks[2].Points[0].APIRolloutCost; got == nil {
		t.Fatal("API-derived rollout cost is nil")
	} else {
		assertNear(t, *got, 17/1.47*expectedAstraAPI, 1e-12)
	}
	if got := benchmarks[0].Points[0].Offers[0].RolloutCost; got == nil {
		t.Fatal("measured token rollout cost is nil")
	} else {
		assertNear(t, *got, tokens/1_000_000*expectedAstraOffer, 1e-12)
	}
	if got := benchmarks[1].Points[0].Offers[0].RolloutCost; got == nil || *got != 0 {
		t.Errorf("zero-token subscription rollout cost = %v, want zero", got)
	}
	if got := benchmarks[2].Points[0].Offers[0].RolloutCost; got == nil {
		t.Fatal("source-calibrated rollout cost is nil")
	} else {
		assertNear(t, *got, 17/1.47*expectedAstraOffer, 1e-12)
	}
	sol := benchmarks[3].Points[0]
	if sol.APIUSDPerMTok == nil {
		t.Fatal("API component price missing for metered model")
	}
	assertNear(t, *sol.APIUSDPerMTok, 20.0/6, 1e-12)
	if len(sol.Offers) != 1 || sol.Offers[0].Plan != "Measured English Plan" {
		t.Fatalf("metered model offers = %+v", sol.Offers)
	}
	if sol.Offers[0].USDPerMTok == nil {
		t.Fatal("subscription API-equivalent unit price missing")
	}
	assertNear(t, *sol.Offers[0].USDPerMTok, 0.294/0.588*(20.0/6), 1e-12)
	if got := sol.APIRolloutCost; got == nil {
		t.Fatal("source-calibrated API rollout cost is nil")
	} else {
		assertNear(t, *got, 1.176/0.588*(20.0/6), 1e-12)
	}
	if got := sol.Offers[0].RolloutCost; got == nil {
		t.Fatal("source-calibrated subscription rollout cost is nil")
	} else {
		assertNear(t, *got, 1.176/0.588*(0.294/0.588*(20.0/6)), 1e-12)
	}
	missing := benchmarks[4].Points[0]
	if missing.APIUSDPerMTok != nil {
		t.Errorf("missing API price = %v, want nil", *missing.APIUSDPerMTok)
	}
	if len(missing.Offers) != 1 || missing.Offers[0].Plan != "No API Plan" || missing.Offers[0].USDPerMTok != nil {
		t.Fatalf("matching subscription with missing source price was changed: %+v", missing.Offers)
	}
	if missing.Offers[0].RolloutCost != nil || missing.APIRolloutCost != nil {
		t.Errorf("missing source prices produced rollout costs: %+v", missing)
	}
	bothMissing := benchmarks[5].Points[0]
	if len(bothMissing.Offers) != 1 || bothMissing.Offers[0].Plan != "No API Plan" ||
		bothMissing.Offers[0].USDPerMTok != nil || bothMissing.Offers[0].RolloutCost != nil {
		t.Fatalf("offer with missing API prices = %+v, want retained offer with null prices", bothMissing.Offers)
	}
	measuredJSON, err := json.Marshal(benchmarks[0].Points[0])
	if err != nil {
		t.Fatal(err)
	}
	var measuredFields struct {
		APIUSDPerMTok  *float64 `json:"apiUSDPerMTok"`
		APIRolloutCost *float64 `json:"apiRolloutCost"`
		Offers         []offer  `json:"offers"`
	}
	if err := json.Unmarshal(measuredJSON, &measuredFields); err != nil {
		t.Fatal(err)
	}
	if measuredFields.APIUSDPerMTok == nil || *measuredFields.APIUSDPerMTok != expectedAstraAPI ||
		measuredFields.APIRolloutCost == nil || *measuredFields.APIRolloutCost != tokens/1_000_000*expectedAstraAPI ||
		measuredFields.Offers[0].USDPerMTok == nil || *measuredFields.Offers[0].USDPerMTok != expectedAstraOffer {
		t.Errorf("no-cache prices missing from point JSON: %+v", measuredFields)
	}
	missingJSON, err := json.Marshal(bothMissing)
	if err != nil {
		t.Fatal(err)
	}
	var missingFields map[string]json.RawMessage
	if err := json.Unmarshal(missingJSON, &missingFields); err != nil {
		t.Fatal(err)
	}
	if string(missingFields["apiUSDPerMTok"]) != "null" || string(missingFields["apiRolloutCost"]) != "null" {
		t.Errorf("missing API unit/rollout JSON = %s / %s, want null", missingFields["apiUSDPerMTok"], missingFields["apiRolloutCost"])
	}
	var missingOffers []map[string]json.RawMessage
	if err := json.Unmarshal(missingFields["offers"], &missingOffers); err != nil {
		t.Fatal(err)
	}
	if string(missingOffers[0]["usdPerMTok"]) != "null" || string(missingOffers[0]["rolloutCost"]) != "null" {
		t.Errorf("unavailable offer JSON = %s / %s, want null", missingOffers[0]["usdPerMTok"], missingOffers[0]["rolloutCost"])
	}
	if got, want := []string{plans[0].Name, plans[1].Name, plans[2].Name, plans[3].Name}, []string{"No API Plan", "Astra Plan", "Measured English Plan", "Premium Plan"}; !equalStrings(got, want) {
		t.Errorf("used plans sorted by monthly price and name = %v, want %v", got, want)
	}
}

func TestApplySubscriptionsSortsEqualUnitPricesByPlan(t *testing.T) {
	pricing := readPricingFixture(t)
	pricing.Points = append(pricing.Points, pricingPoint{
		Billing: "subscription", Plan: "Aardvark Plan", Model: "gpt-6-astra",
		PriceUSD: 10, RealUSDPerMTok: 6, Confidence: "medium",
	})
	benchmarks := []benchmark{{Points: []point{{Model: "gpt-6-astra"}}}}
	applySubscriptions(benchmarks, pricing.Points)
	offers := benchmarks[0].Points[0].Offers
	if got, want := offerPlanNames(offers), []string{"Aardvark Plan", "Astra Plan", "Premium Plan"}; !equalStrings(got, want) {
		t.Fatalf("offers sorted by unit price then plan = %v, want %v", got, want)
	}
	if offers[0].USDPerMTok == nil || offers[1].USDPerMTok == nil {
		t.Fatalf("equal-rate offers have nil unit prices: %+v", offers)
	}
	assertNear(t, *offers[0].USDPerMTok, *offers[1].USDPerMTok, 0)
}

func TestScienceZeroTokensDoNotAffectSubscriptionOffers(t *testing.T) {
	pricing := readPricingFixture(t)
	expectedAstraOffer := 6.0 / 1.47 * (100.0 / 6)
	cases := []struct {
		name     string
		model    string
		metrics  string
		plan     string
		unitRate *float64
		hasAPI   bool
	}{
		{"model without API price", "model-without-api-price", `{"accuracy":0,"total_cost_usd":0,"total_tokens":0,"tasks":1}`, "No API Plan", nil, false},
		{"zero tokens", "GPT-6 Astra", `{"accuracy":0,"total_cost_usd":1,"total_tokens":0,"tasks":1}`, "Astra Plan", &expectedAstraOffer, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			response := []byte(`{"rows":[{"metadata":{"model_display":{"label":"` + test.model + `"},"agent_display":{"label":"Codex"},"reasoning_effort":"max"},"metrics":` + test.metrics + `}]}`)
			parsed, err := parseTerminalBenchScience(response, terminalSciencePageURL)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Points[0].Tokens == nil || *parsed.Points[0].Tokens != 0 {
				t.Fatalf("parsed tokens = %v, want measured zero", parsed.Points[0].Tokens)
			}
			benchmarks := []benchmark{parsed}
			applySubscriptions(benchmarks, pricing.Points)
			point := benchmarks[0].Points[0]
			if (point.APIUSDPerMTok != nil) != test.hasAPI {
				t.Errorf("API unit price = %v, hasAPI %v; want hasAPI %v", point.APIUSDPerMTok, point.APIUSDPerMTok != nil, test.hasAPI)
			}
			for _, candidate := range point.Offers {
				if candidate.Plan == test.plan {
					if (candidate.USDPerMTok != nil) != (test.unitRate != nil) {
						t.Errorf("offer rate = %v, want %v", candidate.USDPerMTok, test.unitRate)
					} else if test.unitRate != nil {
						assertNear(t, *candidate.USDPerMTok, *test.unitRate, 1e-12)
					}
					if candidate.RolloutCost == nil || *candidate.RolloutCost != 0 {
						t.Errorf("zero-token rollout cost = %v, want zero", candidate.RolloutCost)
					}
					return
				}
			}
			t.Fatalf("matching plan %q missing from offers: %+v", test.plan, point.Offers)
		})
	}
}

func TestApplySubscriptionsUsesHandAPIPricesAndKeepsSubscriptionWithoutAPIPrice(t *testing.T) {
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
	if points[0].APIUSDPerMTok == nil {
		t.Fatal("hand-table API unit price missing")
	}
	assertNear(t, *points[0].APIUSDPerMTok, 100.0/6, 1e-12)
	legacyRows := append(pricing.Points, pricingPoint{
		Billing: "metered", Model: "gpt-6-astra", RealUSDPerMTok: 0.123, ListBlendedUSDPerMTok: 0.456,
	})
	if got := apiPrices(legacyRows)["gpt-6-astra"]; got != 100.0/6 {
		t.Errorf("legacy metered/blended rate changed displayed API price to %g", got)
	}
	if got := referenceAPIPrices(legacyRows)["gpt-6-astra"]; got != 0.123 {
		t.Errorf("metered legacy reference = %g, want 0.123", got)
	}
	if points[1].APIUSDPerMTok != nil {
		t.Errorf("model without API price = %v, want nil", *points[1].APIUSDPerMTok)
	}
	if len(points[1].Offers) != 1 || points[1].Offers[0].Plan != "No API Plan" || points[1].Offers[0].USDPerMTok != nil {
		t.Fatalf("subscription without API price was omitted or given an unsupported rate: %+v", points[1].Offers)
	}
}

func TestAPIPricesUseUncachedComponentsAndConvertCNY(t *testing.T) {
	var anthropic pricingPoint
	if err := json.Unmarshal([]byte(`{"billing":"subscription","vendor":"Anthropic","model":"claude-opus-5.5","list_price":{"input":4,"output":20,"cached":0.01,"currency":"USD"},"list_blended_usd_per_mtok":0.456}`), &anthropic); err != nil {
		t.Fatal(err)
	}
	points := []pricingPoint{
		{Billing: "metered", Model: "gpt-6-astra", RealUSDPerMTok: 0.123, ListBlendedUSDPerMTok: 0.456, ListPrice: &listPrice{Input: 20, Output: 100, Currency: "USD"}},
		{Billing: "subscription", Model: "component-only-model", ListPrice: &listPrice{Input: 3, Output: 9, Currency: "USD"}},
		{Billing: "metered", Model: "cny-model", ListPrice: &listPrice{Input: 2 * cnyPerUSD, Output: 10 * cnyPerUSD, Currency: "CNY"}},
		{Billing: "metered", Model: "invalid-currency", ListPrice: &listPrice{Input: 2, Output: 10, Currency: "EUR"}},
		anthropic,
		{Model: "blend-only-model", Vendor: "OpenAI", ListBlendedUSDPerMTok: 0.456},
		{Model: "anthropic-blend-only", Vendor: "Anthropic", ListBlendedUSDPerMTok: 0.456},
	}
	prices := apiPrices(points)
	assertNear(t, prices["gpt-6-astra"], 200.0/6, 1e-12)
	assertNear(t, prices["component-only-model"], 24.0/6, 1e-12)
	assertNear(t, prices["cny-model"], 20.0/6, 1e-12)
	assertNear(t, prices["claude-opus-5-5"], 40.0/6, 1e-12)
	if _, ok := prices["invalid-currency"]; ok {
		t.Errorf("accepted unsupported API price currency: %v", prices["invalid-currency"])
	}
	if _, ok := prices["blend-only-model"]; ok {
		t.Errorf("used legacy blended price as a displayed API price: %v", prices["blend-only-model"])
	}
	if _, ok := prices["anthropic-blend-only"]; ok {
		t.Errorf("used Anthropic list blend without cache-write pricing: %v", prices["anthropic-blend-only"])
	}
	if got := referenceAPIPrices(points)["blend-only-model"]; got != 0.456 {
		t.Errorf("legacy list blend was not kept as a calibration reference: %v", got)
	}
}

func TestHandAPIPricesUseOrdinaryAnthropicInputRates(t *testing.T) {
	prices := apiPrices(nil)
	assertNear(t, prices["claude-fable-5-2"], 80.0/6, 1e-12)
	assertNear(t, prices["claude-opus-4-8"], 50.0/6, 1e-12)
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
	tokens := 1_000_000.0
	benchmarks := []benchmark{{Points: []point{
		{Model: "gpt-6-astra", Tokens: &tokens},
		{Model: "mimo-v2.6-pro", Tokens: &tokens},
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
	tokens := 1_000_000.0
	benchmarks := []benchmark{{Points: []point{{Model: "gpt-6-astra", Tokens: &tokens}}}}
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
