# leaderboards

Terminal-Bench / FrontierCode のスコアと試行平均料金を比較する Go CLI。
サブスクリプションの利用実績から推計した実質価格にも切り替えられる。

公開版: https://takumi3488.github.io/leaderboards/

## 実行

Go 1.27.3 以降。

```sh
go run . fetch
go run . build
open leaderboard.html
```

- `fetch`: 原本と取得 URL・日時を `data/` に保存。
- `build`: 保存済みファイルだけから単一 HTML を生成。データ取得の通信なし。
- HTML のデータ・CSS・操作コードは埋め込み。グラフ描画の ECharts のみ CDN を利用するため、閲覧時に通信が必要。
- 出力先や取得 URL は変更可能。`go run . fetch -h` / `go run . build -h` 参照。

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

- Terminal-Bench / FrontierCode、API / サブスク、対数 / 線形軸を切り替え。
- サブスクごとに ON/OFF。初期状態は全 ON。同じモデルに対応する有効なプランのうち、実質価格が最安のものを採用。
- 「換算に使うプラン」を開いて設定変更。「モデル」で表示する曲線と一覧表を絞り込める。
- プラン対応はモデル名の完全一致。大文字小文字・区切り文字・プロバイダ接頭辞と Claude の表示名のみ正規化。別バージョンや `preview` 等の別名には拡張しない。
- 有効プランに完全一致の記載がなければ API 料金に fallback。全プラン OFF の場合も同様。

```text
サブスク実質価格 = 試行平均API料金 × 月額料金 / 月平均API換算価値
```

月額・API 換算価値は gist の概要表、対応モデルは各プランのモデル表から抽出する。
同じ文書内の説明文と表が異なる場合、表を採用。モデル別 API 換算値ではなく、
複数モデルで共有されるプラン全体の比率を利用する。

- Terminal-Bench: `total_cost_usd / n_trials`。スコアは `accuracy`。
- FrontierCode: `cost` は rollout 平均 USD。スコアは `new_score × 100`。
- FrontierCode は公開 JSON の `main` 結果を全件利用。履歴モデルも含む。
- ベンチマーク間のスコアは直接比較しない。モデル・ハーネス単位で effort を線で接続する。
- 公開データが料金 0 を報告する場合、0 のまま保持。対数軸では描画できないため件数を注記し、表と線形軸には残す。

実質価格は個人の利用実績に基づく推計。定額で同じ試行数を購入できる保証ではない。
共有クォータ、利用モデル構成、キャッシュ、未完了の測定期間、ハーネスやプランの
利用条件によって結果が変わる。モデル表への記載も、現在の提供・アクセス可否の保証ではない。

## 原本とオフライン検証

```sh
go test ./...
```

抽出・料金計算のテストは `testdata/` の固定原本を利用。`fetch` や外部サービスへの
接続なしで実行できる。取得済み `data/` があれば HTML 生成もオフライン実行可能。

取得先:

- [Terminal-Bench 4.1](https://hub.harborframework.com/datasets/terminal-bench/terminal-bench/5?tab=leaderboard&leaderboard=4-1-0):
  ページ HTML から leaderboard UUID、JS chunk から `fetchLeaderboardWithRows` の
  action ID を動的に発見して取得。UUID・action ID の固定値には依存しない。
- [FrontierCode](https://cognition.com/frontiercode):
  [公開 JSON](https://cognition.com/data/frontier-code-2.0/data.json)。
- [AI subscription value summary](https://gist.githubusercontent.com/ngardiner/d577b3c9f3ca283937879bae8460102b/raw/ai_subscription_value_summary.md):
  revision なしの最新 Markdown。固定 revision を使う場合は `-subscription-url` 指定。

`data/` の保存内容:

| ファイル | 内容 |
| --- | --- |
| `terminal-bench.html` | Terminal-Bench ページ原本 |
| `terminal-bench-action.js` | 発見した server action を含む JS |
| `terminal-bench.rsc` | server action 応答原本。JSON を抽出して読み込む |
| `frontiercode.json` | FrontierCode 公開 JSON 原本 |
| `ai_subscription_value_summary.md` | gist の Markdown 原文 |
| `sources.json` | 取得 URL・ファイル名・取得日時 |

Terminal-Bench の server action は公開ページの内部インターフェース。
サイト構造が変わった場合、取得処理の更新が必要になる。