package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	codingPlansTableRE = regexp.MustCompile(`(?s)Plans ranked by value.*?<table\b[^>]*>(.*?)</table>`)
	codingPlansRowRE   = regexp.MustCompile(`(?s)<tr\b[^>]*>(.*?)</tr>`)
	codingPlansHeadRE  = regexp.MustCompile(`(?s)<th scope="row"[^>]*>(.*?)</th>`)
	codingPlansNameRE  = regexp.MustCompile(`<span class="[^"]*\bfont-medium\b[^"]*">([^<]*)`)
	codingPlansModelRE = regexp.MustCompile(`<span class="[^"]*text-\[11px\][^"]*">([^<]*)`)
	codingPlansCellRE  = regexp.MustCompile(`(?s)<td\b[^>]*>(.*?)</td>`)
	codingPlansTagRE   = regexp.MustCompile(`<[^>]+>`)
)

// parseCodingPlans reads the "Plans ranked by value" table. Each plan covers
// only the model shown under its name; rows priced at another model's API
// rate ("(assumed)") are skipped because their factor does not fit that model.
func parseCodingPlans(data []byte) ([]subscription, error) {
	table := codingPlansTableRE.FindSubmatch(data)
	if table == nil {
		return nil, fmt.Errorf("CodingPlans page has no Plans ranked by value table")
	}
	var plans []subscription
	for index, row := range codingPlansRowRE.FindAllSubmatch(table[1], -1) {
		head := codingPlansHeadRE.FindSubmatch(row[1])
		if head == nil || strings.Contains(string(head[1]), "(assumed)") {
			continue
		}
		name := firstText(codingPlansNameRE, head[1])
		model := firstText(codingPlansModelRE, head[1])
		if name == "" || model == "" {
			return nil, fmt.Errorf("CodingPlans row %d has no plan name or model", index+1)
		}
		cells := codingPlansCellRE.FindAllSubmatch(row[1], 2)
		if len(cells) < 2 {
			return nil, fmt.Errorf("CodingPlans row %q has fewer than two price cells", name)
		}
		var amounts [2]float64
		for i, label := range []string{"price", "API value"} {
			raw := strings.TrimSpace(html.UnescapeString(codingPlansTagRE.ReplaceAllString(string(cells[i][1]), "")))
			amount, err := parseSubscriptionAmount(raw)
			if err != nil || amount <= 0 {
				return nil, fmt.Errorf("CodingPlans row %q %s %q is not a positive USD amount", name, label, raw)
			}
			amounts[i] = amount
		}
		plans = append(plans, subscription{Name: name + " · CodingPlans", MonthlyUSD: amounts[0], APIValueUSD: amounts[1], Models: []string{model}})
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("CodingPlans Plans ranked by value table has no usable model rows")
	}
	return plans, nil
}

func firstText(pattern *regexp.Regexp, content []byte) string {
	match := pattern.FindSubmatch(content)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(string(match[1])))
}
