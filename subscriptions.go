package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	lastUpdatedPattern = regexp.MustCompile(`(?m)^\s*\*Last updated:\s*([^*]+)\*\s*$`)
	summaryPattern     = regexp.MustCompile(`^<summary>\s*<strong>(.*?)</strong>\s*</summary>$`)
)

// parseSubscriptions reads the overview for pricing and the matching details
// blocks for the exact model names supported by each plan.
func parseSubscriptions(data []byte) ([]subscription, string, error) {
	text := string(data)
	updatedMatch := lastUpdatedPattern.FindStringSubmatch(text)
	if len(updatedMatch) != 2 {
		return nil, "", fmt.Errorf("subscription data is missing a valid Last updated date")
	}
	updated := strings.TrimSpace(updatedMatch[1])
	if _, err := time.Parse("2006-01-02", updated); err != nil {
		return nil, "", fmt.Errorf("invalid subscription Last updated date %q: %w", updated, err)
	}

	lines := strings.Split(text, "\n")
	plans, err := parseSubscriptionOverview(lines)
	if err != nil {
		return nil, "", err
	}
	if err := parseSubscriptionDetails(lines, plans); err != nil {
		return nil, "", err
	}
	return plans, updated, nil
}

func parseSubscriptionOverview(lines []string) ([]subscription, error) {
	headerAt := -1
	var header []string
	var columns map[string]int
	for i, line := range lines {
		cells, ok := markdownTableRow(line)
		if !ok {
			continue
		}
		indices := make(map[string]int, len(cells))
		for index, cell := range cells {
			indices[strings.ToLower(strings.TrimSpace(cell))] = index
		}
		if _, hasSubscription := indices["subscription"]; hasSubscription {
			_, hasCost := indices["avg cost (usd)"]
			_, hasValue := indices["avg api value (usd)"]
			if hasCost && hasValue {
				if headerAt >= 0 {
					return nil, fmt.Errorf("subscription overview contains more than one matching header")
				}
				headerAt, header, columns = i, cells, indices
			}
		}
	}
	if headerAt < 0 {
		return nil, fmt.Errorf("subscription overview table is missing required columns: Subscription, Avg Cost (USD), Avg API Value (USD)")
	}

	plans := make([]subscription, 0, 10)
	planIndices := make(map[string]struct{})
	for i := headerAt + 1; i < len(lines); i++ {
		cells, ok := markdownTableRow(lines[i])
		if !ok {
			break
		}
		if isMarkdownSeparator(cells) {
			continue
		}
		if len(cells) != len(header) {
			return nil, fmt.Errorf("malformed subscription overview row %q: got %d columns, want %d", strings.TrimSpace(lines[i]), len(cells), len(header))
		}
		name := strings.TrimSpace(cells[columns["subscription"]])
		if name == "" {
			return nil, fmt.Errorf("subscription overview contains a row with an empty plan name")
		}
		if _, duplicate := planIndices[name]; duplicate {
			return nil, fmt.Errorf("subscription overview contains duplicate plan %q", name)
		}
		monthlyUSD, err := parseSubscriptionAmount(cells[columns["avg cost (usd)"]])
		if err != nil || monthlyUSD <= 0 {
			return nil, fmt.Errorf("subscription %q has invalid Avg Cost (USD) %q", name, strings.TrimSpace(cells[columns["avg cost (usd)"]]))
		}
		apiValueUSD, err := parseSubscriptionAmount(cells[columns["avg api value (usd)"]])
		if err != nil || apiValueUSD <= 0 {
			return nil, fmt.Errorf("subscription %q has invalid Avg API Value (USD) %q: value must be positive", name, strings.TrimSpace(cells[columns["avg api value (usd)"]]))
		}
		planIndices[name] = struct{}{}
		plans = append(plans, subscription{Name: name, MonthlyUSD: monthlyUSD, APIValueUSD: apiValueUSD})
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("subscription overview table contains no plan rows")
	}
	return plans, nil
}

func parseSubscriptionDetails(lines []string, plans []subscription) error {
	planIndices := make(map[string]int, len(plans))
	for i := range plans {
		planIndices[plans[i].Name] = i
	}
	detailsSeen := make(map[string]bool, len(plans))
	inDetails := false
	var detailPlan string
	var models []string
	modelColumn := -1
	inModelTable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "<details>") {
			if inDetails {
				return fmt.Errorf("subscription details blocks are nested or unclosed")
			}
			inDetails = true
			detailPlan, models, modelColumn, inModelTable = "", nil, -1, false
		}
		if inDetails {
			if strings.HasPrefix(trimmed, "<summary>") {
				match := summaryPattern.FindStringSubmatch(trimmed)
				if len(match) != 2 {
					return fmt.Errorf("malformed subscription details summary %q", trimmed)
				}
				name, ok := detailPlanName(match[1], plans)
				if !ok {
					return fmt.Errorf("subscription details summary %q does not match an overview plan", match[1])
				}
				if detailsSeen[name] {
					return fmt.Errorf("subscription %q has more than one details block", name)
				}
				detailPlan = name
			}
			if cells, ok := markdownTableRow(trimmed); ok {
				if modelColumn < 0 {
					for i, cell := range cells {
						if strings.EqualFold(strings.TrimSpace(cell), "Model") {
							modelColumn = i
							inModelTable = true
							break
						}
					}
				} else if inModelTable {
					if isMarkdownSeparator(cells) {
						continue
					}
					if len(cells) <= modelColumn {
						return fmt.Errorf("malformed model table for subscription %q: row %q has too few columns", detailPlan, trimmed)
					}
					model := strings.TrimSpace(cells[modelColumn])
					if model == "" {
						return fmt.Errorf("model table for subscription %q contains an empty model name", detailPlan)
					}
					models = append(models, model)
				}
			} else if modelColumn >= 0 {
				inModelTable = false
			}
			if strings.Contains(trimmed, "</details>") {
				if detailPlan == "" {
					return fmt.Errorf("subscription details block is missing a matching summary")
				}
				if modelColumn < 0 || len(models) == 0 {
					return fmt.Errorf("subscription %q details block is missing a non-empty Model table", detailPlan)
				}
				plans[planIndices[detailPlan]].Models = models
				detailsSeen[detailPlan] = true
				inDetails = false
				detailPlan, models, modelColumn, inModelTable = "", nil, -1, false
			}
		}
	}
	if inDetails {
		return fmt.Errorf("subscription details block for %q is not closed", detailPlan)
	}
	for _, plan := range plans {
		if !detailsSeen[plan.Name] {
			return fmt.Errorf("subscription %q is missing its matching details block", plan.Name)
		}
	}
	return nil
}

func detailPlanName(summary string, plans []subscription) (string, bool) {
	matched := ""
	for _, plan := range plans {
		if strings.HasPrefix(summary, plan.Name+" - per-model ") {
			if matched != "" {
				return "", false
			}
			matched = plan.Name
		}
	}
	return matched, matched != ""
}

func markdownTableRow(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") {
		return nil, false
	}
	trimmed = strings.Trim(trimmed, "|")
	cells := strings.Split(trimmed, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells, true
}

func isMarkdownSeparator(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if cell == "" {
			return false
		}
		for _, r := range cell {
			if r != '-' && r != ':' && !unicode.IsSpace(r) {
				return false
			}
		}
		if !strings.Contains(cell, "-") {
			return false
		}
	}
	return true
}

func parseSubscriptionAmount(value string) (float64, error) {
	value = strings.ReplaceAll(strings.TrimPrefix(strings.TrimSpace(value), "$"), ",", "")
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, fmt.Errorf("invalid amount")
	}
	return amount, nil
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

// applySubscriptions sets each point's ModelKey and its exact-model offers.
func applySubscriptions(benchmarks []benchmark, subscriptions []subscription) {
	type planRate struct {
		name   string
		factor float64
	}
	rates := make(map[string][]planRate)
	for _, plan := range subscriptions {
		seen := make(map[string]bool)
		factor := plan.MonthlyUSD / plan.APIValueUSD
		for _, model := range plan.Models {
			key := canonicalModel(model)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			rates[key] = append(rates[key], planRate{plan.Name, factor})
		}
	}
	for i := range benchmarks {
		for j := range benchmarks[i].Points {
			p := &benchmarks[i].Points[j]
			p.ModelKey = canonicalModel(p.Model)
			p.Offers = nil
			for _, rate := range rates[p.ModelKey] {
				p.Offers = append(p.Offers, offer{Plan: rate.name, Cost: p.APICost * rate.factor})
			}
			sort.Slice(p.Offers, func(i, j int) bool {
				if p.Offers[i].Cost == p.Offers[j].Cost {
					return p.Offers[i].Plan < p.Offers[j].Plan
				}
				return p.Offers[i].Cost < p.Offers[j].Cost
			})
		}
	}
}
