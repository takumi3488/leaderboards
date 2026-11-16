package main

// Scores are benchmark percentages; API and offer prices are USD per million tokens.
type point struct {
	ID    string `json:"id"`
	Model string `json:"model"`
	Agent string `json:"agent"`
	// ModelKey is canonicalModel(Model); the page shares model selection across benchmarks by it.
	ModelKey string  `json:"modelKey"`
	Effort   string  `json:"effort"`
	Score    float64 `json:"score"`
	// APICost and Tokens preserve source metrics; rollout costs use no-cache estimates.
	APICost        float64  `json:"apiCost"`
	Tokens         *float64 `json:"tokens,omitempty"`
	APIUSDPerMTok  *float64 `json:"apiUSDPerMTok"`
	APIRolloutCost *float64 `json:"apiRolloutCost"`
	Color          string   `json:"color,omitempty"`
	Offers         []offer  `json:"offers"`
}

type offer struct {
	Plan        string   `json:"plan"`
	USDPerMTok  *float64 `json:"usdPerMTok"`
	RolloutCost *float64 `json:"rolloutCost"`
	Confidence  string   `json:"confidence"`
}

type subscription struct {
	Name       string  `json:"name"`
	MonthlyUSD float64 `json:"monthlyUSD"`
}

type pricingPoint struct {
	Billing               string     `json:"billing"`
	Plan                  string     `json:"plan"`
	PlanEN                *string    `json:"plan_en"`
	Vendor                string     `json:"vendor"`
	Model                 string     `json:"model"`
	PriceUSD              float64    `json:"price_usd"`
	ListBlendedUSDPerMTok float64    `json:"list_blended_usd_per_mtok"`
	ListPrice             *listPrice `json:"list_price"`
	RealUSDPerMTok        float64    `json:"real_usd_per_mtok"`
	Unmetered             bool       `json:"unmetered"`
	Confidence            string     `json:"confidence"`
}

type listPrice struct {
	Input    float64 `json:"input"`
	Output   float64 `json:"output"`
	Currency string  `json:"currency"`
}

type pricingData struct {
	GeneratedAt string         `json:"generatedAt"`
	Points      []pricingPoint `json:"points"`
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
	DataDir     string
	TerminalURL string
	FrontierURL string
	ScienceURL  string
	PricingURL  string
}

const (
	defaultTerminalURL = "https://hub.harborframework.com/datasets/terminal-bench/terminal-bench/5?tab=leaderboard&leaderboard=4-1-0"
	defaultFrontierURL = "https://cognition.com/data/frontier-code-2.0/data.json"
	defaultScienceURL  = "https://www.terminal-bench-science.ai/api/leaderboard?package=terminal-bench-science%2Fterminal-bench-science&name=v0-1-eval"
	defaultPricingURL  = "https://raw.githubusercontent.com/Fei2411/real-api-pricing/main/derived/points.json"
)

type benchmark struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	ScoreLabel string  `json:"scoreLabel"`
	CostNote   string  `json:"costNote"`
	Points     []point `json:"points"`
}
