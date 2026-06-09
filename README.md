# KAO (花押)

GitHub + Kubernetes + Argo CD + Terraform 環境において、変更管理統制 (Change Management Controls) を継続的に評価する CLI。

花押は文書の真正性を証明する署名。KAO は本番に反映された変更について、誰が・誰が承認し・何がデプロイされ・適切な統制のもとで行われたかを証跡として収集・評価する。

## HADO との関係

| | HADO | KAO |
|---|---|---|
| タイミング | リリース前 | リリース後 |
| 問い | 変更を出してよいか | 変更は適切な統制のもとで出されたか |

## 責務

1. 証跡を収集する
2. 証跡を評価する
3. 評価結果を出力する

## v1 で扱う証跡

| 証跡 | 収集元 (v1) |
|---|---|
| `repo_control` | GitHub（Branch Protection, CODEOWNERS 等） |
| `code_change` | GitHub（default branch へマージされた PR） |
| `app_deployment` | Argo CD |
| `infra_deployment` | Terraform（GitHub Actions 上の apply） |

CM-001〜010 により統制を評価する。詳細は [設計ドキュメント](docs/design-doc.md) を参照。

## 運用

毎日 00:00 UTC に `kao run daily` を実行する。証跡と評価結果は PostgreSQL（推奨）または filesystem backend に保存する。

```bash
kao run daily --config kao.yaml --date 2026-06-05
```

認証情報は Configuration に平文で書かず、環境変数（`GITHUB_TOKEN`, `ARGOCD_SERVER`, `ARGOCD_TOKEN`, `DATABASE_URL` 等）から注入する。`go.mod` の `argoproj/argo-cd/v2` は本番 Argo CD サーバーのバージョンに合わせて pin する。

## 設計ドキュメント

- [docs/design-doc.md](docs/design-doc.md) — v1.0 (Accepted)

## ローカル開発

```bash
# ビルド
make build

# PostgreSQL 起動とスキーマ適用
docker compose up -d
make migrate

# 環境変数
export GITHUB_TOKEN=ghp_...
export DATABASE_URL='postgres://kao:kao@localhost:5432/kao?sslmode=disable'

# 日次ジョブ（GitHub 証跡のみ、MVP）
./bin/kao run daily --config kao.local.yaml --date 2026-06-05

# 個別コマンド
./bin/kao collect repo-control --config kao.local.yaml --output -
./bin/kao collect code-change --config kao.local.yaml --date 2026-06-05 --output -
./bin/kao evaluate --config kao.local.yaml --date 2026-06-05 --output -
```

ファイル入力での評価（storage 不要）:

```bash
./bin/kao evaluate --input files \
  --repo-control testdata/repo_control.jsonl \
  --code-change testdata/code_change.jsonl \
  --output -
```

## ステータス

MVP 実装中。`repo_control` / `code_change` の GitHub 収集・評価・PostgreSQL/filesystem 保存に対応。
