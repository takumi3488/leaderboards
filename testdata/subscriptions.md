*Last updated: 2026-11-09*

## How Data is Collected

All inference across all AI providers is routed through [LiteLLM](https://github.com/BerriAI/litellm) with spend logging enabled. This captures token usage and calculates API-equivalent cost for every request, attributed to the correct subscription. Running everything through a single proxy makes the data directly comparable across providers regardless of their native APIs or billing models.

To ensure the comparison is fair, the goal for each subscription is to consume the full allocation available under the plan - working through rate limits (5-hour windows, weekly caps, monthly token budgets) until the subscription allocation is consumed or the measurement period completes.

## Subscription Performance Overview

| Subscription | Months | Avg Cost (USD) | Avg Tokens In | Avg Tokens Out | Avg API Value (USD) | Avg Tokens/$1 |
|--------------|--------|----------------|---------------|----------------|---------------------|----------------|
| Claude Pro | 4 | $20.00 | 1,781.6M | 9.3M | $972.32 | 89.5M |
| Claude Max | 1 | $200.00 | 52,284.7M | 171.6M | $17,557.85 | 262.3M |
| Minimax | 2 | $21.00 | 4,102.7M | 26.7M | $1,052.98 | 196.6M |
| Google AI Pro | 7 | $19.99 | 2,036.7M | 6.2M | $1,248.86 | 102.2M |
| Kiro | 6 | $20.00 | 210.1M | 0.3M | $546.92 | 10.5M |
| Kiro Power | 1 | $200.00 | 2,506.5M | 5.5M | $3,122.33 | 12.6M |
| ChatGPT Plus | 2 | $20.00 | 2,300.5M | 8.0M | $1,523.45 | 115.4M |
| OpenCode Go | 1 | $10.00 | 297.1M | 1.0M | $144.50 | 29.8M |
| Alibaba Cloud | 1 | $8.00 | 167.1M | 0.5M | $67.88 | 20.9M |
| Poe | 4 | $4.99 | 24.2M | 1.6M | $7.09 | 5.2M |

Click a subscription below to see the per-model breakdown across tracked periods.

<details>
<summary><strong>Claude Pro - per-model averages (4 periods)</strong></summary>

| Model | Periods | Avg Input Tokens | Avg Output Tokens | Avg API Value |
|-------|---------|-----------------|------------------|---------------|
| claude-sonnet-4-6 | 4 | 1,402.2M | 7.8M | $662.70 |
| claude-fable-5 | 2 | 244.6M | 0.9M | $350.03 |
| claude-opus-4-8 | 2 | 302.7M | 0.7M | $207.54 |
| claude-sonnet-5 | 1 | 194.3M | 0.4M | $53.16 |
| claude-haiku-4-5 | 4 | 57.0M | 0.6M | $17.54 |

</details>

<details>
<summary><strong>Claude Max - per-model totals (1 period)</strong></summary>

| Model | Periods | Input Tokens | Output Tokens | API Value |
|-------|---------|--------------|---------------|-----------|
| claude-opus-5-5 | 1 | 12,488.6M | 50.2M | $5,110.98 |
| claude-sonnet-5 | 1 | 6,517.9M | 3.7M | $1,655.55 |
| anthropic/claude-opus-4-8 | 1 | 2,195.8M | 7.7M | $1,502.43 |
| anthropic/claude-opus-5-5 | 1 | 357.9M | 1.0M | $256.29 |
| claude-fable-5-1 | 1 | 799.6M | 5.1M | $888.03 |
| claude-sonnet-5-5 | 1 | 920.8M | 3.8M | $268.46 |
| claude-haiku-5-5 | 1 | 8,097.3M | 45.6M | $180.81 |
| claude-opus-5 | 1 | 252.4M | 0.6M | $165.16 |

</details>

<details>
<summary><strong>Google AI Pro - per-model averages (7 periods)</strong></summary>

| Model | Periods | Avg Input Tokens | Avg Output Tokens | Avg API Value |
|-------|---------|-----------------|------------------|---------------|
| gemini-3-flash-preview | 5 | 895.4M | 4.9M | $458.26 |
| gemini-3.1-pro-preview | 5 | 99.8M | 0.3M | $202.74 |
| gemini-3.7-flash | 1 | 2,823.0M | 3.8M | $1,423.05 |
| gemini-3.6-flash | 2 | 536.4M | 1.0M | $271.30 |
| claude-sonnet-4-6 | 1 | 58.8M | 0.4M | $182.18 |
| gemini-3-pro-preview | 1 | 13.0M | 0.2M | $28.40 |
| gemini-3.1-pro-low | 1 | 9.3M | 0.03M | $19.00 |
| gemini-2.5-pro | 2 | 0.7M | 0.1M | $1.54 |
| gemini-2.5-flash | 2 | 3.9M | 0.6M | $2.66 |

</details>

<details>
<summary><strong>Kiro - per-model averages (6 periods)</strong></summary>

| Model | Periods | Avg Input Tokens | Avg Output Tokens | Avg API Value |
|-------|---------|-----------------|------------------|---------------|
| claude-sonnet-4-6 | 5 | 178.9M | 0.1M | $538.44 |
| claude-sonnet-4-5 | 1 | 106.7M | 0.3M | $324.54 |
| claude-sonnet-5 | 2 | 101.3M | 0.4M | $86.86 |
| claude-opus-5 | 1 | 12.8M | 0.1M | $73.54 |
| deepseek-3.2 | 1 | 21.2M | 0.04M | $2.14 |

</details>

<details>
<summary><strong>Kiro Power - per-model totals (1 period)</strong></summary>

| Model | Periods | Input Tokens | Output Tokens |
|-------|---------|--------------|---------------|
| claude-sonnet-5 | 1 | 2,341.5M | 5.1M |
| claude-opus-5 | 1 | 55.9M | 0.2M |
| claude-haiku-4.5 | 1 | 0.2M | 0.001M |
| claude-sonnet-4-6 | 1 | 0.01M | 0.00002M |

</details>

<details>
<summary><strong>OpenCode Go - per-model averages (1 period)</strong></summary>

| Model | Periods | Avg Input Tokens | Avg Output Tokens | Avg API Value |
|-------|---------|-----------------|------------------|---------------|
| qwen3.7-plus | 1 | 264.1M | 0.8M | $106.87 |
| qwen3.7-max | 1 | 10.7M | 0.04M | $27.04 |
| qwen3.6-plus | 1 | 0.1M | 0.1M | $0.40 |

</details>

<details>
<summary><strong>ChatGPT Plus - per-model totals (2 periods, latest in progress)</strong></summary>

| Model | Periods | Input Tokens | Output Tokens |
|-------|---------|--------------|---------------|
| gpt-6-luna | 1 | 2,760.6M | 9.3M |
| gpt-6-sol | 1 | 625.7M | 1.0M |
| gpt-6.1-sol | 1 | 572.9M | 1.2M |
| gpt-5.6-terra | 2 | 403.4M | 1.7M |
| gpt-5.6-luna | 2 | 97.2M | 0.3M |
| gpt-5.6-sol | 1 | 69.0M | 0.5M |
| gpt-5.4-mini | 1 | 44.1M | 1.8M |
| gpt-5.4 | 1 | 28.1M | 0.2M |
| chatgpt/gpt-5.4-mini | 1 | 0.06M | 0.0001M |
| gpt-6-astra | 1 | 0.00164M | 0.00001M |

</details>

<details>
<summary><strong>Alibaba Cloud - per-model totals (1 period)</strong></summary>

| Model | Periods | Input Tokens | Output Tokens |
|-------|---------|--------------|---------------|
| qwen3.7-plus | 1 | 167.1M | 0.5M |
| deepseek-v4-pro | 1 | 0.01M | 0.0001M |
| glm-5.2 | 1 | 0.0001M | 0.0003M |
| qwen3.6-flash | 1 | 0.00005M | 0.0007M |
| qwen3.7-max | 1 | 0.00005M | 0.001M |

</details>

<details>
<summary><strong>Minimax - per-model totals (2 periods, latest in progress)</strong></summary>

| Model | Periods | Input Tokens | Output Tokens |
|-------|---------|--------------|---------------|
| MiniMax-M3 | 2 | 4,404.5M | 12.5M |
| MiniMax-M3.1-Flash | 1 | 2,039.8M | 13.2M |
| MiniMax-M2.7 | 1 | 787.7M | 65.1M |
| MiniMax-M3.1-Flash-Preview | 1 | 707.6M | 4.3M |
| MiniMax-M2.7-highspeed | 1 | 0.00013M | 0.00015M |

</details>

<details>
<summary><strong>Poe - per-model totals (4 periods, latest in progress)</strong></summary>

| Model | Periods | Input Tokens | Output Tokens |
|-------|---------|--------------|---------------|
| gpt-oss-120b-t | 3 | 66.6M | 0.2M |
| qwen3-vl-235b-a22b-i | 3 | 19.6M | 6.0M |
| llama-3.3-70b-t | 2 | 6.0M | 0.01958M |
| gpt-4o-mini | 2 | 0.5M | 0.00304M |
| qwen3.7-plus | 2 | 0.4M | 0.02M |
| minimax-m3-t | 1 | 0.4M | 0.0224M |
| llama-3.3-70b | 1 | 0.4M | 0.00134M |
| gemini-2.5-pro | 2 | 0.2M | 0.11M |
| o3-mini-high | 1 | 0.05M | 0.0033M |
| gemma-4-31b | 3 | 0.046M | 0.00246M |
| qwen3-235b-a22b-n | 1 | 0.01861M | 0.0003M |
| deepseek-r1-n | 1 | 0.01207M | 0.00216M |
| solar-pro-2 | 1 | 0.01044M | 0.00029M |
| gemini-2.5-flash | 1 | 0.00406M | 0.00512M |
| phi-4-di | 1 | 0.00357M | 0.00056M |

</details>

## Methodology

**API Value** represents the equivalent cost if the same token volumes were purchased at standard API pricing (per-token rates), not the actual subscription cost paid.

**Avg Tokens/$1** measures token efficiency: how many tokens are delivered per dollar of subscription cost. Higher values indicate better utilization of the subscription's token capacity. Calculated as (Avg Tokens In + Avg Tokens Out) / Nominal Monthly Cost.

**Avg Cost** reflects the provider's nominal public monthly subscription fee. If a provider changes its public price, the Avg Cost column reflects the weighted average across periods at each price point.

**Months** counts tracked periods with recorded non-zero token volume. Partial periods are included at their accumulated values as of the last update.

Metrics use **Synthesized Zero-Caching** methodology, which calculates gross token volumes without accounting for prompt caching discounts. This provides a consistent baseline for comparing raw token delivery across different subscription models. There are two exceptions:

- **Claude (Pro and Max)** API Value is **cache-aware from measured data**. Anthropic reports cache reads and writes on every request, so those tokens are priced at each model's own cache rates. Agentic sessions run 95-98% cache reads, so pricing them at the full input rate overstated Claude's value by 4-7x; Claude figures before 2026-10-10 were calculated that way and have been revised down. Token columns are unaffected.
- **Kiro (Pro and Power)** API Value applies a fixed 0.375x caching factor, because the Kiro gateway does not report cache usage.

## Key Observations

- **Claude Pro** averages **$972 of cache-aware API value per month (49x its $20 price)**, driven by aggressive use of 5-hour rate-limit windows with claude-sonnet-4-6 as the primary model. This replaces the earlier $5,772 figure, which priced cached input at the full rate (see Methodology).
- **Claude Max** (x20 tier) is in its first tracked period: 36.53B tokens and $11,467.72 of cache-aware API value so far (57.3x its $200 price).
- **Minimax** delivered the highest token throughput and purchasing power of any single period: **4.48B gross tokens** and **$1,409.53 API-equivalent value** at **223.8M tokens/$1**. A second tracked period on the renamed Go tier ($22) is in progress: 3.78B tokens and $696.42 of API-equivalent value so far (31.7x its price), split between MiniMax-M3 and MiniMax-M3.1-Flash, the latter valued at its published $0.15/$0.60 per 1M rate.
- **Google AI Pro** averages 2.04B input tokens per tracked month at 102.2M tokens/$1 across 7 recorded periods. The most recent tracked period (#15) delivered 3.79B gross tokens / $2,157.40 API value / 189.5M tokens/$1. Period #14 (first post-Gemini-CLI month) delivered 2.19B gross tokens / $1,288.45 API value / 109.8M tokens/$1, showing that Antigravity CLI traffic on Claude Sonnet 4.6 and other models sustained strong utilization after the Gemini CLI discontinuation.
- **OpenCode Go** showed strong utilization at 29.8M tokens/$1 on open-source coding models, with qwen3.7-plus driving the majority of throughput.
- **Kiro** averaged 10.5M tokens/$1 across 6 paid periods; Kiro #6 closed with 159.5M tokens and $241.56 in API-equivalent value before transitioning to the higher Kiro Power tier.
- **Kiro Power** is a higher-cost tier ($200/mo, 10,000 credits/month) succeeding Kiro Pro. Its tracked period delivered 2.51B tokens and $3,122.33 in API-equivalent value at 12.6M tokens/$1, driven primarily by claude-sonnet-5.
- **ChatGPT Plus** delivered 606.4M tokens in its first tracked month at 30.3M tokens/$1, with gpt-5.6-terra leading throughput. The second tracked period is in progress: 4.01B tokens and $2,742.74 of API-equivalent value so far (137.1x its $20 price). Most of that volume is the low-cost gpt-6-luna, so token throughput has risen much faster than value; gpt-6-sol remains the largest value contributor.
- **Alibaba Cloud** delivered 167.6M tokens at 20.9M tokens/$1 and $67.88 API-equivalent value.
- **Poe** has four tracked periods (the latest in progress). Combined: 103.0M gross tokens and $28.37 cumulative API value, with gpt-oss-120b-t and qwen3-vl-235b-a22b-i carrying the majority of throughput.

## FAQ

**Q: Are you using a specific prompt, pattern, or repetitive workload to inflate the numbers?**

No. This is strictly real usage from active projects. The methodology is to deliberately undersize inference requirements for a given month - spreading work across several subscriptions - and then consume each subscription to its limits (5-hour windows, weekly caps, monthly token budgets, wherever applicable) until it is exhausted or the measurement period completes. The only subscription where full utilization wasn't achieved was the first Claude Pro period, which has been retained in the dataset as it hasn't materially affected the overall Claude statistics.

**Q: How do rate limit resets or temporary capacity boosts factor in?**

Where a provider offers a temporary capacity boost or reset as part of the subscription - such as Anthropic's temporary 50% weekly limit increase - that is included in the data because it represents genuine value delivered by the plan. Usage delivered outside the measured subscription periods is excluded.

**Q: Are the numbers for complete measurement periods only?**

No. The aggregates can include partial measurement periods. Those figures are current as of the document's last-updated date.

**Q: How often is this updated?**

At a minimum weekly, and more frequently when significant usage is accumulating.

## Related Data

The entries below are other publicly available data points on AI subscription value and token efficiency. I do not endorse or vouch for any of these — methodologies, definitions, and subscription conditions vary. I believe this data is most useful in aggregate, so I maintain an automated search for similar work and list what I find here.

*List last updated by automated search: 2026-11-01*

### Confirmed

**[What a User Story Actually Costs in a Dark Code Factory (fxmartin)](https://gist.github.com/fxmartin/979da2a47fbbbac6d72d238073e23491)**
Detailed per-token cost accounting of an automated SDLC pipeline on Claude Max. 595.7M tokens across 193 story-builds; median $9.56 per delivered story. Includes raw CSV data and Python extraction script. One-off dataset (2026-07-24).

**[Claude Code quota consumption mechanics — tokenmaxxing (septimlabs-code)](https://gist.github.com/septimlabs-code/819f49cb8f03fc17d4c4cb44536276ab)**
Technical breakdown of the hidden cache_creation_input_tokens ledger and how request-count vs. token-count billing mismatches burn enterprise quota. Includes bash snippet for session log analysis. One-off post (May 2026).

### Also Found

*These results were returned by automated search and have not been manually reviewed.*

**[AI Token Usage Tracking — Copilot, OpenAI, Claude, Cursor](https://cloudmonitor.ai/finops-domains/understand-cloud-usage-and-cost-domain/ai-token-usage-tracking-capability/)**

**[Honest comparison after 4 months running Claude Pro ...](https://www.reddit.com/r/AI_Agents/comments/1tg953c/honest_comparison_after_4_months_running_claude/)**

**[Ran ChatGPT Plus and Claude Pro side by side for 30 days, here's what I ...](https://www.reddit.com/r/ChatGPT/comments/1smgpy6/ran_chatgpt_plus_and_claude_pro_side_by_side_for/)**

**[ChatGPT vs Gemini vs Claude Pro: 2M Token Gap [2026]](https://tech-insider.org/chatgpt-vs-gemini-vs-claude-pro-2026/)**

**[I Built a Claude Code Cost Tracker - Was Max Worth It?](https://www.ksred.com/i-built-a-cost-tracker-for-claude-code-to-see-if-my-subscription-was-worth-it/)**

**[Claude Code Cost & Token Usage Tracker, Free and Local](https://codeburn.app/claude-code-cost-tracker)**

**[Claude is 10x more expensive per token than any other ...](https://www.reddit.com/r/ClaudeCode/comments/1t1rkd1/claude_is_10x_more_expensive_per_token_than_any/)**

**[Z.ai vs Claude Pro vs ChatGPT Plus: 2026 Value Comparison (Real ...](https://docs.bswen.com/blog/2026-03-11-ai-subscription-value-comparison/)**
