package main

import (
	"fmt"
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	codingPlansTableOpenRE = regexp.MustCompile(`(?i)<table\b[^>]*>`)
	codingPlansTableEndRE  = regexp.MustCompile(`(?i)</table\s*>`)
	codingPlansRowsRE      = regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr\s*>`)
	codingPlansHeaderRE    = regexp.MustCompile(`(?is)<th\b([^>]*)>(.*?)</th\s*>`)
	codingPlansSpanRE      = regexp.MustCompile(`(?is)<span\b([^>]*)>([^<]*)`)
	codingPlansCellsRE     = regexp.MustCompile(`(?is)<td\b[^>]*>(.*?)</td\s*>`)
	codingPlansTagsRE      = regexp.MustCompile(`(?is)<[^>]+>`)
	codingPlansAttrRE      = regexp.MustCompile(`(?i)\b([a-z_:][-a-z0-9_:.]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	codingPlansMoneyRE     = regexp.MustCompile(`^\$?(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)(?:\.[0-9]+)?$`)
)

func parseCodingPlans(data []byte) ([]subscription, error) {
	page := string(data)
	heading := strings.Index(page, "Plans ranked by value")
	if heading < 0 {
		return nil, fmt.Errorf("CodingPlans page has no Plans ranked by value table")
	}
	afterHeading := page[heading+len("Plans ranked by value"):]
	open := codingPlansTableOpenRE.FindStringIndex(afterHeading)
	if open == nil {
		return nil, fmt.Errorf("CodingPlans page has no Plans ranked by value table")
	}
	tableBody := afterHeading[open[1]:]
	close := codingPlansTableEndRE.FindStringIndex(tableBody)
	if close == nil {
		return nil, fmt.Errorf("CodingPlans Plans ranked by value table is not closed")
	}

	rows := codingPlansRowsRE.FindAllStringSubmatch(tableBody[:close[0]], -1)
	plans := make([]subscription, 0, len(rows))
	for rowIndex, rowMatch := range rows {
		row := rowMatch[1]
		header := codingPlansHeaderRE.FindStringSubmatch(row)
		if len(header) != 3 || !strings.EqualFold(codingPlansAttribute(header[1], "scope"), "row") {
			continue
		}
		name := codingPlansTextForClass(header[2], "font-medium")
		label := codingPlansTextForClass(header[2], "text-[11px]")
		model := strings.TrimSpace(strings.SplitN(label, " · ", 2)[0])
		if model == "" || strings.EqualFold(model, "model menu varies") {
			continue
		}
		if name == "" {
			return nil, fmt.Errorf("CodingPlans row %d has no plan name", rowIndex+1)
		}
		rowName := name
		cells := codingPlansCellsRE.FindAllStringSubmatch(row, -1)
		if len(cells) < 2 {
			return nil, fmt.Errorf("CodingPlans row %q has fewer than two price cells", rowName)
		}
		priceText := codingPlansCellText(cells[0][1])
		price, err := parseCodingPlansMoney(priceText)
		if err != nil {
			return nil, fmt.Errorf("CodingPlans row %q price %q: %w", rowName, priceText, err)
		}
		valueText := codingPlansCellText(cells[1][1])
		value, err := parseCodingPlansMoney(valueText)
		if err != nil {
			return nil, fmt.Errorf("CodingPlans row %q API value %q: %w", rowName, valueText, err)
		}
		plans = append(plans, subscription{
			Name:        name + " · CodingPlans",
			MonthlyUSD:  price,
			APIValueUSD: value,
			Models:      []string{model},
		})
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("CodingPlans Plans ranked by value table has no usable model rows")
	}
	return plans, nil
}

func codingPlansTextForClass(content, class string) string {
	for _, match := range codingPlansSpanRE.FindAllStringSubmatch(content, -1) {
		classes := strings.Fields(codingPlansAttribute(match[1], "class"))
		for _, candidate := range classes {
			if candidate == class {
				return strings.TrimSpace(html.UnescapeString(match[2]))
			}
		}
	}
	return ""
}

func codingPlansAttribute(attributes, name string) string {
	for _, match := range codingPlansAttrRE.FindAllStringSubmatch(attributes, -1) {
		if !strings.EqualFold(match[1], name) {
			continue
		}
		for _, value := range match[2:] {
			if value != "" {
				return value
			}
		}
		return ""
	}
	return ""
}

func codingPlansCellText(content string) string {
	return strings.TrimSpace(html.UnescapeString(codingPlansTagsRE.ReplaceAllString(content, "")))
}

func parseCodingPlansMoney(raw string) (float64, error) {
	if !codingPlansMoneyRE.MatchString(raw) {
		return 0, fmt.Errorf("invalid USD amount")
	}
	valueText := strings.ReplaceAll(strings.TrimPrefix(raw, "$"), ",", "")
	value, err := strconv.ParseFloat(valueText, 64)
	if err != nil || value <= 0 || math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, fmt.Errorf("USD amount must be positive")
	}
	return value, nil
}
