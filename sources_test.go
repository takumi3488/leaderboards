package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCapturedBenchmarkSources(t *testing.T) {
	terminalResponse, err := os.ReadFile(filepath.Join("testdata", terminalRSCFile))
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := parseTerminalBenchmark(terminalResponse, "")
	if err != nil {
		t.Fatal(err)
	}
	opus := findPoint(t, terminal.Points, "Opus 5.5", "max")
	assertNear(t, opus.Score, 65.76, 0.000001)
	assertNear(t, opus.APICost, 4718.65/330, 0.000001)
	zeroCost := findPoint(t, terminal.Points, "Muse Spark 1.3", "xhigh")
	if zeroCost.APICost != 0 {
		t.Fatalf("zero API cost was not preserved: per-trial=%v", zeroCost.APICost)
	}

	frontierData, err := os.ReadFile(filepath.Join("testdata", frontierJSONFile))
	if err != nil {
		t.Fatal(err)
	}
	frontier, err := parseFrontierCode(frontierData, "")
	if err != nil {
		t.Fatal(err)
	}
	frontierOpus := findPoint(t, frontier.Points, "Claude Opus 5.5", "low")
	assertNear(t, frontierOpus.Score, 14.72, 0.000001)
	assertNear(t, frontierOpus.APICost, 0.746, 0.000001)
	findPointByModel(t, frontier.Points, "Claude Opus 4.8")
}

func TestTerminalBenchmarkRejectsMissingAndZeroNumbers(t *testing.T) {
	cases := []struct {
		name    string
		metrics string
		wantErr string
	}{
		{"missing accuracy", `{"accuracy_ci95_half_width":1,"total_cost_usd":0,"n_trials":2}`, "accuracy"},
		{"missing total cost", `{"accuracy":50,"accuracy_ci95_half_width":1,"n_trials":2}`, "total_cost_usd"},
		{"null trials", `{"accuracy":50,"accuracy_ci95_half_width":1,"total_cost_usd":0,"n_trials":null}`, "n_trials"},
		{"zero trials", `{"accuracy":50,"accuracy_ci95_half_width":1,"total_cost_usd":0,"n_trials":0}`, "positive count"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			response := syntheticTerminalResponse(test.metrics)
			_, err := parseTerminalBenchmark(response, "https://example.test/terminal")
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("got error %v, want text %q", err, test.wantErr)
			}
		})
	}
}

func TestTerminalBenchmarkSkipsNonDisplayRowsAndAllowsZeroCost(t *testing.T) {
	response := []byte(`1:{"title":"Sample","rows":[` +
		`{"id":"hidden","status":"pending","metrics":{}},` +
		`{"id":"free","status":"display","metadata":{"model_display":{"label":"Model"},"agent_display":{"label":"Agent"},"reasoning_effort":"low"},"metrics":{"accuracy":12.5,"accuracy_ci95_half_width":2,"total_cost_usd":0,"total_tokens":1000,"n_trials":2}}]}`)
	parsed, err := parseTerminalBenchmark(response, "https://example.test/terminal")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Points) != 1 || parsed.Points[0].ID != "free" || parsed.Points[0].Score != 12.5 || parsed.Points[0].APICost != 0 || parsed.Points[0].Tokens != 500 {
		t.Fatalf("unexpected parsed visible row: %+v", parsed.Points)
	}
}

func TestFrontierCodeRequiresScoreAndCost(t *testing.T) {
	cases := []struct {
		name string
		data string
		want string
	}{
		{"missing score", `{"data":{"Model":{"high":{"main":{"cost":1}}}}}`, "new_score"},
		{"null cost", `{"data":{"Model":{"high":{"main":{"new_score":0.2,"cost":null}}}}}`, "cost"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseFrontierCode([]byte(test.data), frontierCodePageURL)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got error %v, want text %q", err, test.want)
			}
		})
	}
}

func TestNextFlightLeaderboardAndServerActionDiscovery(t *testing.T) {
	firstFragment, err := json.Marshal(`76:["$","$L79",null,{"leaderboards":[{"id":"leaderboard-uuid","name":"4-1-0","title":"Terminal `)
	if err != nil {
		t.Fatal(err)
	}
	secondFragment, err := json.Marshal(`Bench"}]}]`)
	if err != nil {
		t.Fatal(err)
	}
	html := []byte(`<script>self.__next_f.push([1,` + string(firstFragment) + `])</script><script>self.__next_f.push([1,` + string(secondFragment) + `])</script>`)
	leaderboard, err := findLeaderboardReference(html, "4-1-0")
	if err != nil {
		t.Fatal(err)
	}
	if leaderboard.ID != "leaderboard-uuid" || leaderboard.Title != "Terminal Bench" {
		t.Fatalf("unexpected leaderboard: %+v", leaderboard)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_next/static/immutable/chunks/base.js":
			_, _ = w.Write([]byte("no action here"))
		case "/_next/static/chunks/action.js":
			_, _ = w.Write([]byte(`(0,tw.createServerReference)("40775de86105e3d601d31c1040d2235ad70cc4a593",tw.callServer,void 0,tw.findSourceMapURL,"fetchLeaderboardWithRows")`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	pageURL, err := url.Parse(server.URL + "/dataset")
	if err != nil {
		t.Fatal(err)
	}
	scriptHTML := []byte(`<script src="/_next/static/immutable/chunks/base.js"></script><script src="/_next/static/chunks/action.js"></script>`)
	bundleURL, actionID, body, _, err := fetchServerActionBundle(context.Background(), server.Client(), scriptHTML, pageURL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(bundleURL, "/_next/static/chunks/action.js") {
		t.Fatalf("got action bundle URL %q", bundleURL)
	}
	if actionID != "40775de86105e3d601d31c1040d2235ad70cc4a593" || !strings.Contains(string(body), "fetchLeaderboardWithRows") {
		t.Fatalf("action bundle was not identified: id=%q body=%q", actionID, body)
	}
}

func TestServerActionIDAcceptsDirectCallForm(t *testing.T) {
	const want = "0123456789abcdef0123456789abcdef01234567"
	script := []byte(`createServerReference("` + want + `",callServer,void 0,findSourceMapURL,"fetchLeaderboardWithRows")`)
	if got := serverActionID(script); got != want {
		t.Fatalf("got action ID %q, want %q", got, want)
	}
}

func TestTerminalActionResponseFrameBoundaries(t *testing.T) {
	response := []byte("0:{\"a\":\"$@1\"}\n1:{\"unrelated\":true}\n2:{broken frame}\na:{\"id\":\"board-id\",\"title\":\"Board title\",\"rows\":[]}\n")
	leaderboard, err := parseTerminalActionResponse(response)
	if err == nil || !strings.Contains(err.Error(), "no rows") {
		t.Fatalf("got leaderboard=%+v error=%v, want empty leaderboard frame error", leaderboard, err)
	}

	valid := []byte("0:{\"a\":\"$@1\"}\n1:{\"unrelated\":true}\n2:{broken frame}\na:{\"id\":\"board-id\",\"title\":\"Board title\",\"rows\":[{}]}\n")
	leaderboard, err = parseTerminalActionResponse(valid)
	if err != nil {
		t.Fatal(err)
	}
	if leaderboard.Title != "Board title" || len(leaderboard.Rows) != 1 {
		t.Fatalf("wrong leaderboard frame selected: %+v", leaderboard)
	}
	if _, err := parseTerminalActionResponse([]byte("1:{\"title\":\"Board\",\"rows\":[{}]}trailing")); err == nil {
		t.Fatal("accepted trailing data beyond the response frame")
	}
}

func TestFetchDocumentReportsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	_, err := fetchDocument(context.Background(), server.Client(), http.MethodGet, server.URL, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 503 Service Unavailable") {
		t.Fatalf("got error %v, want HTTP 503 detail", err)
	}
}

func syntheticTerminalResponse(metrics string) []byte {
	return []byte(`1:{"title":"Synthetic Terminal Bench","rows":[{"id":"row-id","status":"display","metadata":{"model_display":{"label":"Model"},"agent_display":{"label":"Agent"},"reasoning_effort":"high"},"metrics":` + metrics + `}]}`)
}

func findPoint(t *testing.T, points []point, model, effort string) point {
	t.Helper()
	for _, candidate := range points {
		if candidate.Model == model && candidate.Effort == effort {
			return candidate
		}
	}
	t.Fatalf("no point for model=%q effort=%q", model, effort)
	return point{}
}

func findPointByModel(t *testing.T, points []point, model string) point {
	t.Helper()
	for _, candidate := range points {
		if candidate.Model == model {
			return candidate
		}
	}
	t.Fatalf("no point for model=%q", model)
	return point{}
}

func assertNear(t *testing.T, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("got %g, want %g (tolerance %g)", got, want, tolerance)
	}
}
func TestFetchRejectsUnusablePricingBeforeWritingFiles(t *testing.T) {
	terminalResponse, err := os.ReadFile(filepath.Join("testdata", terminalRSCFile))
	if err != nil {
		t.Fatal(err)
	}
	frontierResponse, err := os.ReadFile(filepath.Join("testdata", frontierJSONFile))
	if err != nil {
		t.Fatal(err)
	}
	page := []byte(`<script>self.__next_f.push([1,"{\"leaderboards\":[{\"id\":\"leaderboard-uuid\",\"name\":\"4-1-0\",\"title\":\"Board\"}]}"])</script><script src="/action.js"></script>`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/terminal":
			if r.Method == http.MethodPost {
				_, _ = w.Write(terminalResponse)
			} else {
				_, _ = w.Write(page)
			}
		case "/action.js":
			_, _ = w.Write([]byte(`createServerReference("40775de86105e3d601d31c1040d2235ad70cc4a593",callServer,void 0,findSourceMapURL,"fetchLeaderboardWithRows")`))
		case "/frontier":
			_, _ = w.Write(frontierResponse)
		case "/pricing":
			_, _ = w.Write([]byte(`{"generatedAt":"2026-11-10","points":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	dataDir := filepath.Join(t.TempDir(), "data")
	err = fetchSources(context.Background(), fetchOptions{
		DataDir: dataDir, TerminalURL: server.URL + "/terminal?leaderboard=4-1-0",
		FrontierURL: server.URL + "/frontier", PricingURL: server.URL + "/pricing",
	})
	if err == nil || !strings.Contains(err.Error(), "no usable subscription") {
		t.Fatalf("got error %v, want unusable pricing validation error", err)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("fetch wrote into data directory before pricing validation: stat error %v", err)
	}
}
