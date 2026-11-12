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
	prices := make(map[string]float64, len(handAPIPrices)+len(points))
	for _, point := range points {
		if point.Billing != "metered" || !positiveFinite(point.RealUSDPerMTok) {
			continue
		}
		key := canonicalModel(point.Model)
		if key == "" {
			continue
		}
		if prices[key] == 0 || point.RealUSDPerMTok < prices[key] {
			prices[key] = point.RealUSDPerMTok
		}
	}
	for model, price := range handAPIPrices {
		key := canonicalModel(model)
		if prices[key] == 0 {
			prices[key] = price.blendedUSDPerMTok()
		}
	}
	for _, point := range points {
		if point.Vendor == "Anthropic" || !positiveFinite(point.ListBlendedUSDPerMTok) {
			continue
		}
		key := canonicalModel(point.Model)
		if key == "" {
			continue
		}
		if prices[key] == 0 {
			prices[key] = point.ListBlendedUSDPerMTok
		}
	}
	return prices
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
	usdPerMTok float64
	confidence string
	monthlyUSD float64
}

// applySubscriptions sets model keys and offers, returning only plans used by an offer.
func applySubscriptions(benchmarks []benchmark, points []pricingPoint) []subscription {
	api := apiPrices(points)
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
		if !exists || point.RealUSDPerMTok < current.usdPerMTok {
			rates[name][model] = planRate{point.RealUSDPerMTok, point.Confidence, point.PriceUSD}
		}
	}

	planInfo := make(map[string]subscription)
	for i := range benchmarks {
		for j := range benchmarks[i].Points {
			point := &benchmarks[i].Points[j]
			point.ModelKey = canonicalModel(point.Model)
			point.Offers = nil
			for name, models := range rates {
				rate, matches := models[point.ModelKey]
				if !matches {
					continue
				}
				var cost float64
				if point.Tokens != nil {
					cost = *point.Tokens / 1_000_000 * rate.usdPerMTok
				} else if apiRate := api[point.ModelKey]; apiRate > 0 {
					cost = point.APICost * rate.usdPerMTok / apiRate
				} else {
					continue
				}
				point.Offers = append(point.Offers, offer{
					Plan: name, Cost: cost, USDPerMTok: rate.usdPerMTok, Confidence: rate.confidence,
				})
				if _, exists := planInfo[name]; !exists {
					planInfo[name] = subscription{Name: name, MonthlyUSD: rate.monthlyUSD}
				}
			}
			sort.Slice(point.Offers, func(i, j int) bool {
				if point.Offers[i].Cost == point.Offers[j].Cost {
					return point.Offers[i].Plan < point.Offers[j].Plan
				}
				return point.Offers[i].Cost < point.Offers[j].Cost
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
