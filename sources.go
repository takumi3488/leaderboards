package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	terminalHTMLFile        = "terminal-bench.html"
	terminalActionFile      = "terminal-bench-action.js"
	terminalRSCFile         = "terminal-bench.rsc"
	frontierJSONFile        = "frontiercode.json"
	realPricingJSONFile     = "real-api-pricing.json"
	sourcesMetadataFile     = "sources.json"
	frontierCodePageURL     = "https://cognition.com/frontiercode"
	terminalBenchmarkID     = "terminal-bench"
	frontierCodeBenchmarkID = "frontier-code"
)

var (
	flightStringRE = regexp.MustCompile(`(?s)\.push\(\[\s*1\s*,\s*("(?:\\.|[^"\\])*")\s*\]\s*\)`)
	scriptSourceRE = regexp.MustCompile(`(?i)<script\b[^>]*\bsrc\s*=\s*(?:"([^"]+)"|'([^']+)')[^>]*>`)
	actionRE       = regexp.MustCompile(`(?s)createServerReference\s*\)?\s*\(\s*["']([0-9a-fA-F]+)["'][^)]*["']fetchLeaderboardWithRows["']`)
)

type fetchedDocument struct {
	body      []byte
	url       string
	fetchedAt string
}

type leaderboardReference struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// fetchSources downloads the source payloads and saves them unchanged for offline builds.
func fetchSources(ctx context.Context, opts fetchOptions) error {
	if strings.TrimSpace(opts.DataDir) == "" {
		return errors.New("data directory is required")
	}
	terminalURL := opts.TerminalURL
	frontierURL := opts.FrontierURL
	pricingURL := opts.PricingURL
	requestedLeaderboard, err := requestedLeaderboardName(terminalURL)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 45 * time.Second}
	page, err := fetchDocument(ctx, client, http.MethodGet, terminalURL, nil, nil)
	if err != nil {
		return fmt.Errorf("fetch Terminal-Bench page: %w", err)
	}
	pageRequestURL, err := url.Parse(page.url)
	if err != nil {
		return fmt.Errorf("parse fetched Terminal-Bench URL %q: %w", page.url, err)
	}
	leaderboard, err := findLeaderboardReference(page.body, requestedLeaderboard)
	if err != nil {
		return fmt.Errorf("find Terminal-Bench leaderboard %q: %w", requestedLeaderboard, err)
	}

	actionURL, actionID, actionBody, actionFetchedAt, err := fetchServerActionBundle(ctx, client, page.body, pageRequestURL)
	if err != nil {
		return fmt.Errorf("find Terminal-Bench server action: %w", err)
	}
	requestBody, err := json.Marshal([]string{leaderboard.ID})
	if err != nil {
		return fmt.Errorf("encode Terminal-Bench action request: %w", err)
	}
	headers := make(http.Header)
	headers.Set("next-action", actionID)
	headers.Set("accept", "text/x-component")
	headers.Set("content-type", "text/plain;charset=UTF-8")
	rsc, err := fetchDocument(ctx, client, http.MethodPost, page.url, headers, requestBody)
	if err != nil {
		return fmt.Errorf("fetch Terminal-Bench rows: %w", err)
	}
	if _, err := parseTerminalActionResponse(rsc.body); err != nil {
		return fmt.Errorf("parse Terminal-Bench server-action response: %w", err)
	}

	frontier, err := fetchDocument(ctx, client, http.MethodGet, frontierURL, nil, nil)
	if err != nil {
		return fmt.Errorf("fetch FrontierCode data: %w", err)
	}
	if _, err := parseFrontierCode(frontier.body, frontierCodePageURL); err != nil {
		return fmt.Errorf("parse downloaded FrontierCode data: %w", err)
	}
	pricing, err := fetchDocument(ctx, client, http.MethodGet, pricingURL, nil, nil)
	if err != nil {
		return fmt.Errorf("fetch real-api-pricing data: %w", err)
	}
	if _, err := parseRealPricing(pricing.body); err != nil {
		return fmt.Errorf("parse downloaded real-api-pricing data: %w", err)
	}

	sources := []source{
		{Name: "Terminal-Bench page", URL: page.url, File: terminalHTMLFile, FetchedAt: page.fetchedAt},
		{Name: "Terminal-Bench server-action bundle", URL: actionURL, File: terminalActionFile, FetchedAt: actionFetchedAt},
		{Name: "Terminal-Bench leaderboard response", URL: rsc.url, File: terminalRSCFile, FetchedAt: rsc.fetchedAt},
		{Name: "FrontierCode data", URL: frontier.url, File: frontierJSONFile, FetchedAt: frontier.fetchedAt},
		{Name: "Real API pricing data", URL: pricing.url, File: realPricingJSONFile, FetchedAt: pricing.fetchedAt},
	}
	if err := os.MkdirAll(opts.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	files := []struct {
		name string
		body []byte
	}{
		{terminalHTMLFile, page.body},
		{terminalActionFile, actionBody},
		{terminalRSCFile, rsc.body},
		{frontierJSONFile, frontier.body},
		{realPricingJSONFile, pricing.body},
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(opts.DataDir, file.name), file.body, 0o644); err != nil {
			return fmt.Errorf("save %s: %w", file.name, err)
		}
	}
	metadata, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		return fmt.Errorf("encode source metadata: %w", err)
	}
	metadata = append(metadata, '\n')
	if err := os.WriteFile(filepath.Join(opts.DataDir, sourcesMetadataFile), metadata, 0o644); err != nil {
		return fmt.Errorf("save %s: %w", sourcesMetadataFile, err)
	}
	return nil
}

func fetchDocument(ctx context.Context, client *http.Client, method, rawURL string, headers http.Header, body []byte) (fetchedDocument, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return fetchedDocument{}, fmt.Errorf("request %s: %w", rawURL, err)
	}
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fetchedDocument{}, fmt.Errorf("request %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fetchedDocument{}, fmt.Errorf("request %s: HTTP %s", rawURL, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fetchedDocument{}, fmt.Errorf("read response from %s: %w", rawURL, err)
	}
	actualURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		actualURL = resp.Request.URL.String()
	}
	return fetchedDocument{body: data, url: actualURL, fetchedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func requestedLeaderboardName(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse Terminal-Bench URL: %w", err)
	}
	name := parsed.Query().Get("leaderboard")
	if name == "" {
		return "", errors.New("Terminal-Bench URL must include a leaderboard query parameter")
	}
	return name, nil
}

func findLeaderboardReference(html []byte, name string) (leaderboardReference, error) {
	var joined strings.Builder
	for _, match := range flightStringRE.FindAllSubmatch(html, -1) {
		var flightText string
		if err := json.Unmarshal(match[1], &flightText); err == nil {
			joined.WriteString(flightText)
		}
	}
	for _, raw := range jsonArraysForKey(joined.String(), "leaderboards") {
		var leaderboards []leaderboardReference
		if err := json.Unmarshal(raw, &leaderboards); err != nil {
			continue
		}
		for _, item := range leaderboards {
			if item.Name == name && item.ID != "" {
				return item, nil
			}
		}
	}
	return leaderboardReference{}, fmt.Errorf("no leaderboard named %q in Next flight data", name)
}

func jsonArraysForKey(text, key string) [][]byte {
	needle := `"` + key + `"`
	var values [][]byte
	for offset := 0; offset < len(text); {
		relative := strings.Index(text[offset:], needle)
		if relative < 0 {
			break
		}
		keyStart := offset + relative
		pos := keyStart + len(needle)
		for pos < len(text) && isJSONSpace(text[pos]) {
			pos++
		}
		if pos < len(text) && text[pos] == ':' {
			pos++
			for pos < len(text) && isJSONSpace(text[pos]) {
				pos++
			}
			if pos < len(text) && text[pos] == '[' {
				if end, ok := jsonValueEnd(text, pos); ok {
					values = append(values, []byte(text[pos:end]))
				}
			}
		}
		offset = keyStart + len(needle)
	}
	return values
}

func jsonValueEnd(text string, start int) (int, bool) {
	if start >= len(text) || (text[start] != '[' && text[start] != '{') {
		return 0, false
	}
	stack := []byte{text[start]}
	inString, escaped := false, false
	for pos := start + 1; pos < len(text); pos++ {
		ch := text[pos]
		if inString {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '[', '{':
			stack = append(stack, ch)
		case ']', '}':
			if len(stack) == 0 || (ch == ']' && stack[len(stack)-1] != '[') || (ch == '}' && stack[len(stack)-1] != '{') {
				return 0, false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return pos + 1, true
			}
		}
	}
	return 0, false
}

func isJSONSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func scriptURLsFromHTML(html []byte, pageURL *url.URL) []string {
	seen := make(map[string]struct{})
	var scripts []string
	for _, match := range scriptSourceRE.FindAllSubmatch(html, -1) {
		src := match[1]
		if len(src) == 0 {
			src = match[2]
		}
		resolved, err := pageURL.Parse(string(src))
		if err != nil || (resolved.Scheme != "http" && resolved.Scheme != "https") {
			continue
		}
		normalized := resolved.String()
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		scripts = append(scripts, normalized)
	}
	return scripts
}

func fetchServerActionBundle(ctx context.Context, client *http.Client, html []byte, pageURL *url.URL) (string, string, []byte, string, error) {
	scripts := scriptURLsFromHTML(html, pageURL)
	if len(scripts) == 0 {
		return "", "", nil, "", errors.New("Terminal-Bench page contains no JavaScript script references")
	}
	var firstFetchError error
	firstFailedScript := ""
	for _, script := range scripts {
		doc, err := fetchDocument(ctx, client, http.MethodGet, script, nil, nil)
		if err != nil {
			if firstFetchError == nil {
				firstFailedScript = script
				firstFetchError = err
			}
			continue
		}
		if id := serverActionID(doc.body); id != "" {
			return doc.url, id, doc.body, doc.fetchedAt, nil
		}
	}
	if firstFetchError != nil {
		return "", "", nil, "", fmt.Errorf("no fetchLeaderboardWithRows server action found; linked bundle %s failed: %w", firstFailedScript, firstFetchError)
	}
	return "", "", nil, "", fmt.Errorf("no fetchLeaderboardWithRows server action found in %d linked JavaScript bundles", len(scripts))
}

func serverActionID(script []byte) string {
	match := actionRE.FindSubmatch(script)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}

// parseTerminalActionResponse extracts the complete leaderboard object from one Next Flight frame.
func parseTerminalActionResponse(response []byte) (terminalActionLeaderboard, error) {
	for _, line := range bytes.Split(response, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		colon := bytes.IndexByte(line, ':')
		if colon <= 0 {
			continue
		}
		frameID := string(line[:colon])
		if _, err := strconv.ParseUint(frameID, 16, 64); err != nil {
			continue
		}
		payload := bytes.TrimSpace(line[colon+1:])
		var candidate map[string]json.RawMessage
		if err := json.Unmarshal(payload, &candidate); err != nil {
			continue
		}
		if _, ok := candidate["rows"]; !ok {
			continue
		}
		var leaderboard terminalActionLeaderboard
		if err := json.Unmarshal(payload, &leaderboard); err != nil {
			return terminalActionLeaderboard{}, fmt.Errorf("decode leaderboard frame %s: %w", frameID, err)
		}
		if leaderboard.Title == "" {
			return terminalActionLeaderboard{}, fmt.Errorf("leaderboard frame %s has no title", frameID)
		}
		if len(leaderboard.Rows) == 0 {
			return terminalActionLeaderboard{}, errors.New("Terminal-Bench response contains no rows")
		}
		return leaderboard, nil
	}
	return terminalActionLeaderboard{}, errors.New("no complete leaderboard object in server-action response")
}

type terminalActionLeaderboard struct {
	ID    string              `json:"id"`
	Title string              `json:"title"`
	Rows  []terminalActionRow `json:"rows"`
}

type terminalActionRow struct {
	ID       string                 `json:"id"`
	Status   string                 `json:"status"`
	Metadata terminalActionMetadata `json:"metadata"`
	Metrics  terminalActionMetrics  `json:"metrics"`
}

type terminalActionMetadata struct {
	ModelDisplay    *terminalLabel `json:"model_display"`
	AgentDisplay    *terminalLabel `json:"agent_display"`
	ReasoningEffort string         `json:"reasoning_effort"`
}

type terminalLabel struct {
	Label string `json:"label"`
}

type terminalActionMetrics struct {
	Accuracy     *float64 `json:"accuracy"`
	TotalCostUSD *float64 `json:"total_cost_usd"`
	NTrials      *int64   `json:"n_trials"`
	TotalTokens  *float64 `json:"total_tokens"`
}

func parseTerminalBenchmark(response []byte, sourceURL string) (benchmark, error) {
	leaderboard, err := parseTerminalActionResponse(response)
	if err != nil {
		return benchmark{}, err
	}
	points := make([]point, 0, len(leaderboard.Rows))
	for index, row := range leaderboard.Rows {
		if row.Status != "display" {
			continue
		}
		if row.Metadata.ModelDisplay == nil || strings.TrimSpace(row.Metadata.ModelDisplay.Label) == "" {
			return benchmark{}, fmt.Errorf("display row %d has no model_display.label", index)
		}
		if row.Metadata.AgentDisplay == nil || strings.TrimSpace(row.Metadata.AgentDisplay.Label) == "" {
			return benchmark{}, fmt.Errorf("display row %d has no agent_display.label", index)
		}
		if row.Metrics.Accuracy == nil {
			return benchmark{}, fmt.Errorf("display row %d has no accuracy", index)
		}
		if row.Metrics.TotalCostUSD == nil {
			return benchmark{}, fmt.Errorf("display row %d has no total_cost_usd", index)
		}
		if row.Metrics.NTrials == nil {
			return benchmark{}, fmt.Errorf("display row %d has no n_trials", index)
		}
		if *row.Metrics.NTrials <= 0 {
			return benchmark{}, fmt.Errorf("display row %d has invalid n_trials %d; expected a positive count", index, *row.Metrics.NTrials)
		}
		tokens := float64(0)
		if row.Metrics.TotalTokens != nil {
			if *row.Metrics.TotalTokens < 0 {
				return benchmark{}, fmt.Errorf("display row %d has invalid total_tokens %g; expected a non-negative count", index, *row.Metrics.TotalTokens)
			}
			tokens = *row.Metrics.TotalTokens / float64(*row.Metrics.NTrials)
		}
		points = append(points, point{
			ID:      row.ID,
			Model:   row.Metadata.ModelDisplay.Label,
			Agent:   row.Metadata.AgentDisplay.Label,
			Effort:  row.Metadata.ReasoningEffort,
			Score:   *row.Metrics.Accuracy,
			APICost: *row.Metrics.TotalCostUSD / float64(*row.Metrics.NTrials),
			Tokens:  tokens,
		})
	}
	if len(points) == 0 {
		return benchmark{}, errors.New("Terminal-Bench response contains no display rows")
	}
	return benchmark{
		ID:         terminalBenchmarkID,
		Title:      leaderboard.Title,
		URL:        sourceURL,
		ScoreLabel: "Accuracy (%)",
		CostNote:   "API cost per trial",
		Points:     points,
	}, nil
}

type frontierPayload struct {
	Colors  map[string]string                               `json:"colors"`
	Harness map[string]string                               `json:"harness"`
	Data    map[string]map[string]map[string]frontierResult `json:"data"`
}

type frontierResult struct {
	NewScore *float64 `json:"new_score"`
	Cost     *float64 `json:"cost"`
}

func parseFrontierCode(data []byte, sourceURL string) (benchmark, error) {
	var payload frontierPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return benchmark{}, fmt.Errorf("decode FrontierCode JSON: %w", err)
	}
	models := make([]string, 0, len(payload.Data))
	for model := range payload.Data {
		models = append(models, model)
	}
	sort.Strings(models)
	points := make([]point, 0)
	for _, model := range models {
		efforts := make([]string, 0, len(payload.Data[model]))
		for effort := range payload.Data[model] {
			efforts = append(efforts, effort)
		}
		sort.Strings(efforts)
		for _, effort := range efforts {
			mainResult, ok := payload.Data[model][effort]["main"]
			if !ok {
				continue
			}
			if mainResult.NewScore == nil {
				return benchmark{}, fmt.Errorf("FrontierCode %s/%s main result has no new_score", model, effort)
			}
			if mainResult.Cost == nil {
				return benchmark{}, fmt.Errorf("FrontierCode %s/%s main result has no cost", model, effort)
			}
			points = append(points, point{
				ID:      frontierCodeBenchmarkID + ":" + model + ":" + effort,
				Model:   model,
				Agent:   payload.Harness[model],
				Effort:  effort,
				Score:   *mainResult.NewScore * 100,
				APICost: *mainResult.Cost,
				Color:   payload.Colors[model],
			})
		}
	}
	if len(points) == 0 {
		return benchmark{}, errors.New("FrontierCode data contains no main-subset results")
	}
	return benchmark{
		ID:         frontierCodeBenchmarkID,
		Title:      "FrontierCode 2.0",
		URL:        sourceURL,
		ScoreLabel: "New score (%)",
		CostNote:   "Mean rollout API cost",
		Points:     points,
	}, nil
}

func loadBenchmarks(dataDir string) ([]benchmark, []source, error) {
	readData := func(name string) ([]byte, error) {
		path := filepath.Join(dataDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s (run fetch first): %w", path, err)
		}
		return data, nil
	}
	terminalResponse, err := readData(terminalRSCFile)
	if err != nil {
		return nil, nil, err
	}
	frontierData, err := readData(frontierJSONFile)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := readData(sourcesMetadataFile)
	if err != nil {
		return nil, nil, err
	}
	var sources []source
	if err := json.Unmarshal(metadata, &sources); err != nil {
		return nil, nil, fmt.Errorf("decode %s: %w", filepath.Join(dataDir, sourcesMetadataFile), err)
	}
	terminalURL, ok := sourceURLForFile(sources, terminalHTMLFile)
	if !ok {
		return nil, nil, fmt.Errorf("%s has no entry for %s", sourcesMetadataFile, terminalHTMLFile)
	}
	terminal, err := parseTerminalBenchmark(terminalResponse, terminalURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", terminalRSCFile, err)
	}
	frontier, err := parseFrontierCode(frontierData, frontierCodePageURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", frontierJSONFile, err)
	}
	return []benchmark{terminal, frontier}, sources, nil
}

func sourceURLForFile(sources []source, file string) (string, bool) {
	for _, item := range sources {
		if item.File == file {
			return item.URL, true
		}
	}
	return "", false
}
