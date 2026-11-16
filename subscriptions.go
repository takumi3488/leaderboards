package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
)

// parseRealPricing validates the downloaded source before it can be saved or used.
func parseRealPricing(data []byte) (pricingData, error) {
	var parsed pricingData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return pricingData{}, fmt.Errorf("decode real-api-pricing JSON: %w", err)
	}
	if _, err := time.Parse("2006-01-02", parsed.GeneratedAt); err != nil {
		return pricingData{}, fmt.Errorf("invalid real-api-pricing generatedAt %q: %w", parsed.GeneratedAt, err)
	}
	for _, point := range parsed.Points {
		if usableSubscriptionPoint(point) {
			return parsed, nil
		}
	}
	return pricingData{}, fmt.Errorf("real-api-pricing contains no usable subscription points")
}

func usableSubscriptionPoint(point pricingPoint) bool {
	return point.Billing == "subscription" && !point.Unmetered && !excludedPlanScenario(point) &&
		strings.TrimSpace(displayPlan(point)) != "" && strings.TrimSpace(point.Model) != "" &&
		monthlyPrice(point.PriceUSD) && positiveFinite(point.RealUSDPerMTok) &&
		(point.Confidence == "high" || point.Confidence == "medium" || point.Confidence == "low")
}

// Scenario-priced rows are not comparable with ordinary monthly plan rates.
func excludedPlanScenario(point pricingPoint) bool {
	name := strings.TrimSpace(point.Plan)
	return strings.HasSuffix(name, "忙时") || strings.HasSuffix(name, "闲时") || strings.Contains(name, "夜间")
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsInf(value, 0) && !math.IsNaN(value)
}

func monthlyPrice(value float64) bool {
	return value >= 0 && !math.IsInf(value, 0) && !math.IsNaN(value)
}

func displayPlan(point pricingPoint) string {
	if point.PlanEN != nil && strings.TrimSpace(*point.PlanEN) != "" {
		return strings.TrimSpace(*point.PlanEN)
	}
	return strings.TrimSpace(point.Plan)
}

func apiPrices(points []pricingPoint) map[string]float64 {
	meteredPrices := make(map[string]float64, len(points))
	for _, point := range points {
		if point.Billing == "metered" {
			setCheaperPrice(meteredPrices, canonicalModel(point.Model), sourceAPIPrice(point.ListPrice))
		}
	}
	prices := make(map[string]float64, len(handAPIPrices)+len(points))
	for model, price := range meteredPrices {
		prices[model] = price
	}
	for model, price := range handAPIPrices {
		key := canonicalModel(model)
		if _, exists := prices[key]; !exists {
			setCheaperPrice(prices, key, price.usdPerMTok())
		}
	}
	sourcePrices := make(map[string]float64, len(points))
	for _, point := range points {
		if point.Billing != "metered" {
			setCheaperPrice(sourcePrices, canonicalModel(point.Model), sourceAPIPrice(point.ListPrice))
		}
	}
	for key, price := range sourcePrices {
		if _, exists := prices[key]; !exists {
			prices[key] = price
		}
	}
	return prices
}

func referenceAPIPrices(points []pricingPoint) map[string]float64 {
	prices := make(map[string]float64, len(handReferenceAPIPrices)+len(points))
	for _, point := range points {
		if point.Billing == "metered" && positiveFinite(point.RealUSDPerMTok) {
			setCheaperPrice(prices, canonicalModel(point.Model), point.RealUSDPerMTok)
		}
	}
	for model, price := range handReferenceAPIPrices {
		key := canonicalModel(model)
		if _, exists := prices[key]; !exists {
			prices[key] = price
		}
	}
	for _, point := range points {
		if point.Vendor == "Anthropic" || !positiveFinite(point.ListBlendedUSDPerMTok) {
			continue
		}
		key := canonicalModel(point.Model)
		if _, exists := prices[key]; !exists {
			prices[key] = point.ListBlendedUSDPerMTok
		}
	}
	return prices
}

func sourceAPIPrice(price *listPrice) float64 {
	if price == nil || !positiveFinite(price.Input) || !positiveFinite(price.Output) {
		return 0
	}
	var rate float64
	switch strings.ToUpper(strings.TrimSpace(price.Currency)) {
	case "USD":
		rate = (5*price.Input + price.Output) / 6
	case "CNY":
		rate = (5*price.Input + price.Output) / 6 / cnyPerUSD
	default:
		return 0
	}
	if !positiveFinite(rate) {
		return 0
	}
	return rate
}

func setCheaperPrice(prices map[string]float64, key string, price float64) {
	if key == "" || !positiveFinite(price) {
		return
	}
	if prices[key] == 0 || price < prices[key] {
		prices[key] = price
	}
}

// canonicalModel folds only provider prefixes and cosmetic separators; model
// versions and suffixes remain part of the exact comparison key.
func canonicalModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, prefix := range []string{"anthropic/", "chatgpt/", "openai/"} {
		if strings.HasPrefix(model, prefix) {
			model = strings.TrimPrefix(model, prefix)
			break
		}
	}
	var normalized strings.Builder
	normalized.Grow(len(model))
	separator := false
	for _, r := range model {
		if r == '.' || r == '_' || r == '-' || unicode.IsSpace(r) {
			separator = normalized.Len() > 0
			continue
		}
		if separator {
			normalized.WriteByte('-')
			separator = false
		}
		normalized.WriteRune(r)
	}
	key := normalized.String()
	for _, family := range []string{"opus", "sonnet", "fable", "haiku"} {
		if key == family || strings.HasPrefix(key, family+"-") {
			return "claude-" + key
		}
	}
	return key
}

type planRate struct {
	sourceUSDPerMTok float64
	usdPerMTok       *float64
	confidence       string
	monthlyUSD       float64
}

// applySubscriptions sets model keys and offers, returning only plans used by an offer.
func applySubscriptions(benchmarks []benchmark, points []pricingPoint) []subscription {
	api := apiPrices(points)
	referenceAPI := referenceAPIPrices(points)
	rates := make(map[string]map[string]planRate)
	for _, point := range points {
		if !usableSubscriptionPoint(point) {
			continue
		}
		name := displayPlan(point)
		model := canonicalModel(point.Model)
		if model == "" {
			continue
		}
		if rates[name] == nil {
			rates[name] = make(map[string]planRate)
		}
		current, exists := rates[name][model]
		if !exists || point.RealUSDPerMTok < current.sourceUSDPerMTok {
			var unitRate *float64
			if apiRate, hasAPI := api[model]; hasAPI && positiveFinite(referenceAPI[model]) {
				estimated := point.RealUSDPerMTok / referenceAPI[model] * apiRate
				if positiveFinite(estimated) {
					unitRate = &estimated
				}
			}
			rates[name][model] = planRate{
				sourceUSDPerMTok: point.RealUSDPerMTok,
				usdPerMTok:       unitRate,
				confidence:       point.Confidence,
				monthlyUSD:       point.PriceUSD,
			}
		}
	}

	planInfo := make(map[string]subscription)
	for i := range benchmarks {
		for j := range benchmarks[i].Points {
			point := &benchmarks[i].Points[j]
			point.ModelKey = canonicalModel(point.Model)
			point.APIUSDPerMTok = nil
			if apiRate, exists := api[point.ModelKey]; exists && positiveFinite(apiRate) {
				rate := apiRate
				point.APIUSDPerMTok = &rate
			}
			point.APIRolloutCost = estimateRolloutCost(
				point.Tokens, point.APICost, point.APIUSDPerMTok, referenceAPI[point.ModelKey],
			)
			point.Offers = nil
			for name, models := range rates {
				rate, matches := models[point.ModelKey]
				if !matches {
					continue
				}
				rolloutCost := estimateRolloutCost(point.Tokens, point.APICost, rate.usdPerMTok, referenceAPI[point.ModelKey])
				point.Offers = append(point.Offers, offer{
					Plan: name, USDPerMTok: rate.usdPerMTok, RolloutCost: rolloutCost, Confidence: rate.confidence,
				})
				if _, exists := planInfo[name]; !exists {
					planInfo[name] = subscription{Name: name, MonthlyUSD: rate.monthlyUSD}
				}
			}
			sort.Slice(point.Offers, func(i, j int) bool {
				left, right := point.Offers[i], point.Offers[j]
				if (left.USDPerMTok != nil) != (right.USDPerMTok != nil) {
					return left.USDPerMTok != nil
				}
				if left.USDPerMTok == nil || *left.USDPerMTok == *right.USDPerMTok {
					return left.Plan < right.Plan
				}
				return *left.USDPerMTok < *right.USDPerMTok
			})
		}
	}
	plans := make([]subscription, 0, len(planInfo))
	for _, plan := range planInfo {
		plans = append(plans, plan)
	}
	sort.Slice(plans, func(i, j int) bool {
		if plans[i].MonthlyUSD != plans[j].MonthlyUSD {
			return plans[i].MonthlyUSD < plans[j].MonthlyUSD
		}
		return plans[i].Name < plans[j].Name
	})
	return plans
}

func estimateRolloutCost(tokens *float64, apiCost float64, rate *float64, referenceAPI float64) *float64 {
	if tokens != nil {
		if *tokens == 0 {
			zero := 0.0
			return &zero
		}
		if rate == nil {
			return nil
		}
		cost := *tokens / 1_000_000 * *rate
		if !math.IsInf(cost, 0) && !math.IsNaN(cost) {
			return &cost
		}
		return nil
	}
	if rate == nil || !positiveFinite(referenceAPI) {
		return nil
	}
	cost := apiCost / referenceAPI * *rate
	if !math.IsInf(cost, 0) && !math.IsNaN(cost) {
		return &cost
	}
	return nil
}
