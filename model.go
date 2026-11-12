package main

// Costs are USD per rollout; scores are percentages within each benchmark.
type point struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	Agent string `json:"agent"`
	// ModelKey is canonicalModel(Model); the page shares model selection across benchmarks by it.
	ModelKey string  `json:"modelKey"`
	Effort   string  `json:"effort"`
	Score    float64 `json:"score"`
	APICost  float64 `json:"apiCost"`
	Color    string  `json:"color,omitempty"`
	Offers   []offer `json:"offers"`
}

type offer struct {
	Plan string  `json:"plan"`
	Cost float64 `json:"cost"`
}

type benchmark struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	ScoreLabel string  `json:"scoreLabel"`
	CostNote   string  `json:"costNote"`
	Points     []point `json:"points"`
}

type subscription struct {
	Name        string   `json:"name"`
	MonthlyUSD  float64  `json:"monthlyUSD"`
	APIValueUSD float64  `json:"apiValueUSD"`
	Models      []string `json:"-"`
}

type source struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	File      string `json:"file"`
	FetchedAt string `json:"fetchedAt"`
}

type report struct {
	GeneratedAt         string         `json:"generatedAt"`
	SubscriptionUpdated string         `json:"subscriptionUpdated"`
	Benchmarks          []benchmark    `json:"benchmarks"`
	Subscriptions       []subscription `json:"subscriptions"`
	Sources             []source       `json:"sources"`
}

type fetchOptions struct {
	DataDir         string
	TerminalURL     string
	FrontierURL     string
	SubscriptionURL string
	CodingPlansURL  string
}

const (
	defaultTerminalURL     = "https://hub.harborframework.com/datasets/terminal-bench/terminal-bench/5?tab=leaderboard&leaderboard=4-1-0"
	defaultFrontierURL     = "https://cognition.com/data/frontier-code-2.0/data.json"
	defaultSubscriptionURL = "https://gist.githubusercontent.com/ngardiner/d577b3c9f3ca283937879bae8460102b/raw/ai_subscription_value_summary.md"
	defaultCodingPlansURL  = "https://codingplans.app/"
)
