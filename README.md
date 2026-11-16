# leaderboards

Terminal-Bench / FrontierCode / Terminal-Bench Science のスコアと合計100万トークンあたりの API・サブスク単価を比較する Go CLI。

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

OGP/Twitter プレビューには英語固定の `assets/og-image.png`（編集用 SVG: `assets/og-image.svg`）を使い、workflow が `_site/og-image.png` にコピーして GitHub Pages へデプロイする。URL パラメーターごとの画像は生成しない。

## 表示と単価計算

- Terminal-Bench / FrontierCode / Terminal-Bench Science、API / サブスク、料金基準、対数 / 線形軸を切り替え。
- 料金基準は「100万トークン単価」と「実測量換算（1試行）」から独立して選択。初期値は100万トークン単価。既定表示は英語で、URL に `?la=ja` を付けると日本語表示。日時は英語で UTC、日本語で JST 表示。
- ベンチマーク・支払方式・料金基準・軸・プラン・モデルの選択は URL に保存され、再読み込みや共有リンクで復元される。ブラウザーの戻る／進むでも選択状態を復元。
- URL パラメーターは `benchmark`（ベンチマーク ID）、`cost-mode`（`api` / `subscription`）、`cost-basis`（`unit` / `rollout`、省略時は `unit`）、`axis-scale`（`log` / `linear`）、複数指定可能な `disabled-plan`（プラン名）、`model-on` / `model-off`（正規化済みモデルキー）。`la=ja` も併用可能。
- プランは月額の安い順で表示し、各プランを個別に ON/OFF。同じモデルに有効なプランが複数あれば、料金基準にかかわらず既知のキャッシュなし推定単価が最も安いプランを採用する。初期状態は全 ON。
- サブスクモードで有効プランに一致しないモデルは初期状態で非選択。手動選択時は API 推定値を表示する。一致するプランの換算値が不明でもプラン一致として扱い、表は `—`、グラフは除外する。換算不能な一致プランを API 推定値で代替しない。
- モデル照合は料金データとの正規化後の完全一致。大文字小文字、区切り文字、プロバイダ接頭辞と Claude の表示名だけを正規化し、料金照合で別バージョン等は推定しない。
- 表示対象は全ベンチマークの読み込み結果を通じ、同一モデル系列・variant の数値バージョンで最新のものだけ（ドット区切りの世代番号は数値比較）。サイズを表す数値 suffix は別 variant として扱う。たとえば GPT-6.1 Sol があれば GPT-6 Sol / GPT-5.6 Sol は除外する。Sol / Astra / Luna や Claude Opus / Sonnet など異なる系列は別扱い。

料金表示には「100万トークン単価」と「実測量換算（1試行）」がある。API / サブスクは支払方式であり、料金基準とは独立。表示する API・サブスク料金は、いずれもキャッシュなし料金の推定であり、原本の実測 API 費用そのものではない。

- API 100万トークン単価は USD で、入力価格を5、出力価格を1として正規化した合計6トークンの平均 `(5 × input + output) ÷ 6`。5:1 は、従来の比較mixにおける新規入力と出力の比率 2.5:0.5 を、cache-read 分を除いて正規化した比較上の仮定であり、ベンチマーク全体の実測token比ではない。cached input / cache-read と cache-write の料金は使わない。
- 通常の入力・出力 component は、有効な metered 行、`api_prices.go` の手動公式 Input/Output 価格、その他ソース行の有効 component の順に採用する。同じモデルで有効な価格が複数ある場合は既存の規則どおり安い価格を使う。metered の `real_usd` や旧 `list_blended` は、新しい API 表示単価には使わない。
- `list_price` の通貨が CNY の通常入力・出力価格は、`6.7787 CNY/USD` で USD に換算する。レートは [real-api-pricing の conventions.json](https://raw.githubusercontent.com/Fei2411/real-api-pricing/main/data/conventions.json) にある `usdPerCny` の値（適用日 2026-09-04、メタデータ更新 2026-11-10）。
- サブスク100万トークン単価は、原本行の `real_usd_per_mtok ÷ 旧API参照単価 × 新しいキャッシュなしAPI単価` で推定する。これは原本におけるサブスクと旧APIの料金比を保つ方法。月枠を使い切ること、およびこの料金比（割引率）が変わらないことを仮定しており、実際の月間支出やプランの利用量を示すものではない。新単価または旧参照単価がなければ `—`。一致プランは維持し、API 推定値に置き換えない。
- 実測量換算（1試行）では、報告された試行token数があれば、API / サブスクそれぞれの新しい推定単価を `tokens ÷ 1,000,000` 倍する。token数が未報告の場合、原本の実測 `APICost` は費用表示には使わず、`APICost ÷ 旧API参照単価` で相当token量を推定してから新しい API / サブスク推定単価を掛ける。旧API参照単価はサブスク料金比とこのtoken量推定の較正専用で、新しい API 表示単価には使わない。必要な単価・データがなければ `—` とし、一致プランを API 推定費用で代替しない。FrontierCode はtoken数を報告しないため、原本 API 費用からの相当token量推定を使う。
- 0 は有効な費用として `$0.00` 表示する。欠損・換算不能は `—`。表には残し、グラフでは欠損を除外する。対数軸では0以下、線形軸では負の費用をグラフから除外する。0をグラフに表示するには線形軸を選ぶ。

ベンチマーク原本の `cost` / `total_cost_usd` や `costNote` の rollout 実測情報は抽出メタデータとして保持する。原本の実測 API 費用もメタデータに保持するが、費用表示には直接使わず、token数未報告時の相当量推定だけに使う。

- Terminal-Bench: スコアは `accuracy`。
- FrontierCode: スコアは `new_score × 100`。公開 JSON の `main` 結果を全件読み込み、同一系列・variant の旧バージョンは表示対象から除外する。
- Terminal-Bench Science: 全分野の集計結果を利用。スコアは百分率の `accuracy` をそのまま表示する。
- ベンチマーク間のスコアは直接比較しない。モデル・ハーネス単位で effort を線で接続する。

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
  [`derived/points.json`](https://raw.githubusercontent.com/Fei2411/real-api-pricing/main/derived/points.json)。新API単価には `list_price` の通常入力・出力 component を使い、CNY のUSD換算は上記 `conventions.json` のレートを使う。変更時は `-pricing-url` を指定。
  5:1 比率の比較mixの由来は [上流 README](https://raw.githubusercontent.com/Fei2411/real-api-pricing/main/README.md) の従来mix。cache-read 料金を除いた新規入力:出力の比率を正規化して使う。
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
