# leaderboards

Terminal-Bench / FrontierCode / Terminal-Bench Science のスコアと1回あたりの API・サブスク換算料金を比較する Go CLI。

公開版: https://takumi3488.github.io/leaderboards/

## 実行

Go 1.27.3 以降。

```sh
go run . fetch
go run . build
open leaderboard.html
```

- `fetch`: 原本と取得 URL・日時を `data/` に保存。各原本を検証してから保存する。
- `build`: 保存済みファイルだけから単一 HTML を生成。データ取得の通信なし。
- HTML のデータ・CSS・操作コードは埋め込み。グラフ描画の ECharts のみ CDN を利用するため、閲覧時に通信が必要。
- 出力先や取得 URL は変更可能。`go run . fetch -h` / `go run . build -h` 参照。
- Science 追加前の `data/` を利用する場合は、`fetch` を再実行して原本を揃える。

```sh
go run . fetch -data-dir snapshots
go run . build -data-dir snapshots -output comparison.html
```

## 自動更新

`.github/workflows/update.yml` が毎日 06:00 JST、`main` への push 時、手動実行時に
`fetch` → `build` を実行し、GitHub Pages へ deploy する。取得・解析に失敗した場合は
deploy せず、前回公開版を維持する。原本はリポジトリにコミットしない。

リポジトリ設定の Pages の Source は「GitHub Actions」。

## 表示と料金計算

- Terminal-Bench / FrontierCode / Terminal-Bench Science、API / サブスク、対数 / 線形軸を切り替え。
- 画面は英語が既定。URL に `?la=ja` を付けると日本語表示。日時は英語で UTC、日本語で JST 表示。
- プランは料金の安い順で表示し、各プランを個別に ON/OFF。同じモデルに有効なプランが複数あれば換算額が最安のものを採用する。初期状態は全 ON。
- サブスク換算で有効プランに一致しないモデルは初期状態で非選択。手動選択時は API 料金で表示。
- モデル照合は正規化後の完全一致。大文字小文字、区切り文字、プロバイダ接頭辞と Claude の表示名だけを正規化し、別バージョン等は推定しない。

```text
Terminal-Bench / Terminal-Bench Science: トークン数 / rollout × プラン実質 $/MTok
FrontierCode: API rollout 料金 × プラン実質 $/MTok ÷ モデル API $/MTok
```

Terminal-Bench のトークン数は `total_tokens / n_trials`。FrontierCode は出力トークンだけを報告するため API 料金比で換算する。プランの `real_usd_per_mtok` は各行の `workload`（`standard` / `anthropic` / `low-cache` / `measured` raw samples）に従い、一律の混合率は適用しない。`metered` 行は「API list price (metered row)」で、実使用量の実測値ではない。API 参照価格は metered 行、`api_prices.go` の公式価格、非 Anthropic 行の `list_blended_usd_per_mtok` の順で補う。Claude の list blend は cache-write rate を含まないためフォールバックに使わない。手動 API 価格は公式価格の短コンテキスト tier を 97% cache read / 2.5% fresh input / 0.5% output で換算し、Claude fresh input は5分 cache write 価格を使う。

Terminal-Bench Science は `total_tokens / tasks` で1試行あたりのトークン数を求める。公開 JSON の `tasks` は全試行数（0.1 は70問 × 3回 = 210）であり、追加で3分の1にしない。

実質価格は元データに基づく推計で、プラン容量や今後の提供を保証しない。共有枠、モデル構成、利用期間により実際の値は異なる。

- Terminal-Bench: `total_cost_usd / n_trials`。スコアは `accuracy`。
- FrontierCode: `cost` は rollout 平均 USD。スコアは `new_score × 100`。
- FrontierCode は公開 JSON の `main` 結果を全件利用。履歴モデルも含む。
- Terminal-Bench Science: 全分野の集計結果を利用。API 料金は `total_cost_usd / tasks`、スコアは百分率の `accuracy` をそのまま表示。
- ベンチマーク間のスコアは直接比較しない。モデル・ハーネス単位で effort を線で接続する。
- 公開データが料金 0 を報告する場合、0 のまま保持。対数軸では描画できないため件数を注記し、表と線形軸には残す。

## 原本とオフライン検証

```sh
go test ./...
```

抽出・料金計算のテストは `testdata/` の小さな固定 fixture を利用する。`fetch` や外部サービスへの
接続なしで実行できる。取得済み `data/` があれば HTML 生成もオフライン実行可能。

取得先:

- [Terminal-Bench 4.1](https://hub.harborframework.com/datasets/terminal-bench/terminal-bench/5?tab=leaderboard&leaderboard=4-1-0):
  ページ HTML から leaderboard UUID、JS chunk から `fetchLeaderboardWithRows` の
  action ID を動的に発見して取得。
- [FrontierCode](https://cognition.com/frontiercode):
  [公開 JSON](https://cognition.com/data/frontier-code-2.0/data.json)。
- [Terminal-Bench Science 0.1](https://www.terminal-bench-science.ai/):
  [公開 JSON](https://www.terminal-bench-science.ai/api/leaderboard?package=terminal-bench-science%2Fterminal-bench-science&name=v0-1-eval)。変更時は `-science-url` を指定。
- [real-api-pricing](https://github.com/Fei2411/real-api-pricing) ([MIT](https://github.com/Fei2411/real-api-pricing/blob/main/LICENSE)):
  [`derived/points.json`](https://raw.githubusercontent.com/Fei2411/real-api-pricing/main/derived/points.json)。変更時は `-pricing-url` を指定。
- 公式 API 価格（2026-11-12 確認）: [OpenAI](https://developers.openai.com/api/docs/pricing)、[Anthropic](https://platform.claude.com/docs/en/about-claude/pricing)。ソースにないモデルの価格と workload 換算は `api_prices.go` を更新する。

`data/` の保存内容:

| ファイル | 内容 |
| --- | --- |
| `terminal-bench.html` | Terminal-Bench ページ原本 |
| `terminal-bench-action.js` | 発見した server action を含む JS |
| `terminal-bench.rsc` | server action 応答原本。JSON を抽出して読み込む |
| `frontiercode.json` | FrontierCode 公開 JSON 原本 |
| `terminal-bench-science.json` | Terminal-Bench Science 公開 JSON 原本 |
| `real-api-pricing.json` | real-api-pricing 公開 JSON 原本 |
| `sources.json` | 取得 URL・ファイル名・取得日時 |

Terminal-Bench の server action は公開ページの内部インターフェース。
サイト構造が変わった場合、取得処理の更新が必要になる。
