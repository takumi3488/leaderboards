package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRetainLatestModels(t *testing.T) {
	cases := []struct {
		name       string
		benchmarks []benchmark
		want       [][]string
	}{
		{
			name: "latest version is global and keeps every point in stable order",
			benchmarks: []benchmark{
				modelVersionTestBenchmark("first",
					modelVersionTestPoint("GPT-5.6 Sol", "codex", "low"),
					modelVersionTestPoint("GPT-6.1 Sol", "codex", "max"),
					modelVersionTestPoint("GPT-6 Sol", "codex", "high"),
					modelVersionTestPoint("GPT-6.1 Sol", "claude-code", "low"),
				),
				modelVersionTestBenchmark("second",
					modelVersionTestPoint("GPT-6 Sol", "codex", "max"),
					modelVersionTestPoint("GPT-6.1 Sol", "codex", "low"),
				),
			},
			want: [][]string{
				{"GPT-6.1 Sol|codex|max", "GPT-6.1 Sol|claude-code|low"},
				{"GPT-6.1 Sol|codex|low"},
			},
		},
		{
			name: "version components compare numerically",
			benchmarks: []benchmark{
				modelVersionTestBenchmark("decimal",
					modelVersionTestPoint("GPT-6.2 Sol", "agent", "low"),
					modelVersionTestPoint("GPT-6.10 Sol", "agent", "high"),
				),
				modelVersionTestBenchmark("major-minor",
					modelVersionTestPoint("GPT-5.99 Astra", "agent", "low"),
					modelVersionTestPoint("GPT-6 Astra", "agent", "high"),
				),
				modelVersionTestBenchmark("claude-decimal",
					modelVersionTestPoint("Claude Opus 6.2", "agent", "low"),
					modelVersionTestPoint("Claude Opus 6.10", "agent", "high"),
				),
			},
			want: [][]string{
				{"GPT-6.10 Sol|agent|high"},
				{"GPT-6 Astra|agent|high"},
				{"Claude Opus 6.10|agent|high"},
			},
		},
		{
			name: "dotted generation and numeric size variants",
			benchmarks: []benchmark{modelVersionTestBenchmark("qwen",
				modelVersionTestPoint("Qwen2.5-0.5B", "agent", "low"),
				modelVersionTestPoint("Qwen2.5-1.5B", "agent", "high"),
				modelVersionTestPoint("Qwen2.6-0.5B", "agent", "max"),
			)},
			want: [][]string{{
				"Qwen2.5-1.5B|agent|high",
				"Qwen2.6-0.5B|agent|max",
			}},
		},
		{
			name: "trailing zero is numerically equal",
			benchmarks: []benchmark{
				modelVersionTestBenchmark("first",
					modelVersionTestPoint("GPT-6.1 Sol", "codex", "low"),
					modelVersionTestPoint("GPT-6.1.0 Sol", "claude-code", "high"),
				),
				modelVersionTestBenchmark("second",
					modelVersionTestPoint("GPT-6.1 Sol", "codex", "max"),
				),
			},
			want: [][]string{
				{"GPT-6.1 Sol|codex|low", "GPT-6.1.0 Sol|claude-code|high"},
				{"GPT-6.1 Sol|codex|max"},
			},
		},
		{
			name: "families and suffix variants stay independent",
			benchmarks: []benchmark{modelVersionTestBenchmark("variants",
				modelVersionTestPoint("GPT-5.5", "agent", "low"),
				modelVersionTestPoint("GPT-6", "agent", "high"),
				modelVersionTestPoint("GPT-5 Astra", "agent", "low"),
				modelVersionTestPoint("GPT-6 Astra", "agent", "high"),
				modelVersionTestPoint("GPT-5 Luna", "agent", "low"),
				modelVersionTestPoint("GPT-6 Luna", "agent", "high"),
				modelVersionTestPoint("GPT-5 Sol", "agent", "low"),
				modelVersionTestPoint("GPT-6.1 Sol", "agent", "high"),
				modelVersionTestPoint("Claude Opus 4.8", "agent", "low"),
				modelVersionTestPoint("Claude Opus 5.5", "agent", "high"),
				modelVersionTestPoint("Claude Sonnet 4.8", "agent", "low"),
				modelVersionTestPoint("Claude Sonnet 5.5", "agent", "high"),
				modelVersionTestPoint("Claude Fable 5.1", "agent", "low"),
				modelVersionTestPoint("Claude Fable 5.2", "agent", "high"),
				modelVersionTestPoint("Claude Haiku 4.8", "agent", "low"),
				modelVersionTestPoint("Claude Haiku 5.5", "agent", "high"),
				modelVersionTestPoint("GPT-5 Flash", "agent", "low"),
				modelVersionTestPoint("GPT-6 Flash", "agent", "high"),
				modelVersionTestPoint("GPT-5 Pro", "agent", "low"),
				modelVersionTestPoint("GPT-6 Pro", "agent", "high"),
				modelVersionTestPoint("GPT-5 preview", "agent", "low"),
				modelVersionTestPoint("GPT-6 preview", "agent", "high"),
				modelVersionTestPoint("GPT-5 mini", "agent", "low"),
				modelVersionTestPoint("GPT-6 mini", "agent", "high"),
			)},
			want: [][]string{{
				"GPT-6|agent|high",
				"GPT-6 Astra|agent|high",
				"GPT-6 Luna|agent|high",
				"GPT-6.1 Sol|agent|high",
				"Claude Opus 5.5|agent|high",
				"Claude Sonnet 5.5|agent|high",
				"Claude Fable 5.2|agent|high",
				"Claude Haiku 5.5|agent|high",
				"GPT-6 Flash|agent|high",
				"GPT-6 Pro|agent|high",
				"GPT-6 preview|agent|high",
				"GPT-6 mini|agent|high",
			}},
		},
		{
			name: "provider and Claude labels normalize before comparison",
			benchmarks: []benchmark{modelVersionTestBenchmark("normalized",
				modelVersionTestPoint("openai/GPT_6.2-Sol", "agent", "low"),
				modelVersionTestPoint("GPT 6.10 Sol", "agent", "high"),
				modelVersionTestPoint("Anthropic/Opus_4.8", "agent", "low"),
				modelVersionTestPoint("Claude Opus 5.5", "agent", "high"),
			)},
			want: [][]string{{
				"GPT 6.10 Sol|agent|high",
				"Claude Opus 5.5|agent|high",
			}},
		},
		{
			name: "attached version prefixes and numeric suffixes",
			benchmarks: []benchmark{modelVersionTestBenchmark("attached",
				modelVersionTestPoint("DeepSeek V4.1 Flash 1017", "agent", "low"),
				modelVersionTestPoint("DeepSeek V4.2 Flash 1017", "agent", "high"),
				modelVersionTestPoint("DeepSeek V4.1 Flash", "agent", "low"),
				modelVersionTestPoint("Kimi K3.1 Pro", "agent", "low"),
				modelVersionTestPoint("Kimi K4 Pro", "agent", "high"),
			)},
			want: [][]string{{
				"DeepSeek V4.2 Flash 1017|agent|high",
				"DeepSeek V4.1 Flash|agent|low",
				"Kimi K4 Pro|agent|high",
			}},
		},
		{
			name: "unversioned and unknown labels are preserved",
			benchmarks: []benchmark{modelVersionTestBenchmark("unversioned",
				modelVersionTestPoint("GPT Sol", "agent", "low"),
				modelVersionTestPoint("GPT-6 Sol", "agent", "high"),
				modelVersionTestPoint("Opus", "agent", "low"),
				modelVersionTestPoint("Claude Opus 5.5", "agent", "high"),
				modelVersionTestPoint("Unlisted Model", "agent", "max"),
			)},
			want: [][]string{{
				"GPT Sol|agent|low",
				"GPT-6 Sol|agent|high",
				"Opus|agent|low",
				"Claude Opus 5.5|agent|high",
				"Unlisted Model|agent|max",
			}},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			original := append([]benchmark(nil), test.benchmarks...)
			retainLatestModels(test.benchmarks)
			if len(test.benchmarks) != len(test.want) {
				t.Fatalf("got %d benchmarks, want %d", len(test.benchmarks), len(test.want))
			}
			for index, benchmark := range test.benchmarks {
				before := original[index]
				if benchmark.ID != before.ID || benchmark.Title != before.Title || benchmark.URL != before.URL ||
					benchmark.ScoreLabel != before.ScoreLabel || benchmark.CostNote != before.CostNote {
					t.Errorf("benchmark %d metadata changed: got %+v, want %+v", index, benchmark, before)
				}
				if got := modelVersionTestSignatures(benchmark.Points); !equalStrings(got, test.want[index]) {
					t.Errorf("benchmark %d points = %v, want %v", index, got, test.want[index])
				}
			}
		})
	}
}

func TestRetainLatestModelsFromFixturePointsAndSubscriptions(t *testing.T) {
	terminalData, err := os.ReadFile(filepath.Join("testdata", terminalRSCFile))
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := parseTerminalBenchmark(terminalData, "")
	if err != nil {
		t.Fatal(err)
	}
	frontierData, err := os.ReadFile(filepath.Join("testdata", frontierJSONFile))
	if err != nil {
		t.Fatal(err)
	}
	frontier, err := parseFrontierCode(frontierData, "")
	if err != nil {
		t.Fatal(err)
	}
	science, err := parseTerminalBenchScience(syntheticScienceResponse(`{"accuracy":68,"total_cost_usd":100,"total_tokens":1000,"tasks":1}`), "")
	if err != nil {
		t.Fatal(err)
	}
	oldModels := []string{
		"GPT-5.6 Sol", "GPT-6 Sol", "Claude Fable 5.1", "Claude Opus 4.8", "Claude Opus 5", "SWE-1.6", "SWE-1.7",
	}
	for _, model := range oldModels {
		if !modelVersionTestHasPoint(frontier.Points, model) {
			t.Fatalf("fixture no longer contains old candidate %q", model)
		}
	}

	benchmarks := []benchmark{terminal, frontier, science}
	retainLatestModels(benchmarks)
	applySubscriptions(benchmarks, readPricingFixture(t).Points)
	for _, model := range oldModels {
		if modelVersionTestReportHasModel(report{Benchmarks: benchmarks}, model) {
			t.Errorf("old model %q remains in benchmark candidates", model)
		}
	}
	for _, model := range []string{
		"GPT-5.5", "GPT-6 Astra", "GPT-6 Luna", "GPT-6.1 Sol",
		"Claude Fable 5.2", "Claude Opus 5.5", "Claude Sonnet 5.5", "Claude Haiku 5.5",
		"SWE-2", "SWE-2 Flash",
	} {
		if !modelVersionTestReportHasModel(report{Benchmarks: benchmarks}, model) {
			t.Errorf("distinct/latest model %q is missing from benchmark candidates", model)
		}
	}
}

func modelVersionTestBenchmark(id string, points ...point) benchmark {
	return benchmark{ID: "test-" + id, Title: "title-" + id, URL: "https://example.test/" + id, ScoreLabel: "score", CostNote: "cost", Points: points}
}

func modelVersionTestPoint(model, agent, effort string) point {
	return point{Model: model, Agent: agent, Effort: effort}
}

func modelVersionTestSignatures(points []point) []string {
	signatures := make([]string, len(points))
	for index, point := range points {
		signatures[index] = point.Model + "|" + point.Agent + "|" + point.Effort
	}
	return signatures
}

func modelVersionTestHasPoint(points []point, model string) bool {
	for _, point := range points {
		if point.Model == model {
			return true
		}
	}
	return false
}

func modelVersionTestReportHasModel(report report, model string) bool {
	for _, benchmark := range report.Benchmarks {
		if modelVersionTestHasPoint(benchmark.Points, model) {
			return true
		}
	}
	return false
}
