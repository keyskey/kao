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

## Install

`go install` でインストール:

```bash
go install github.com/keyskey/kao/cmd/kao@latest
```

CI やチーム運用ではバージョンを pin する:

```bash
go install github.com/keyskey/kao/cmd/kao@v0.1.0
```

## Binary Distribution

GitHub Releases からプリビルドバイナリを配布する。対応アーカイブ:

- darwin-arm64
- darwin-amd64
- linux-arm64
- linux-amd64

最新版をインストール:

```bash
curl -fsSL "https://raw.githubusercontent.com/keyskey/kao/main/scripts/install.sh" | sh
```

特定バージョンをインストール:

```bash
VERSION=v0.1.0 curl -fsSL "https://raw.githubusercontent.com/keyskey/kao/main/scripts/install.sh" | sh
```

カスタムインストール先:

```bash
INSTALL_DIR="$HOME/.local/bin" curl -fsSL "https://raw.githubusercontent.com/keyskey/kao/main/scripts/install.sh" | sh
```

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

## Versioning Policy

KAO は [Semantic Versioning](https://semver.org/) に従う。

pre-1.0 フェーズでは `0.x` でバージョニングし、統制評価ロジックを実運用で検証する。

- `PATCH` (`0.1.0` -> `0.1.1`): バグ修正のみ
- `MINOR` (`0.1.1` -> `0.2.0`): 新機能・挙動変更・破壊的変更を含む
- `MAJOR` (`1.x`): インターフェースと出力が安定した後に使用

`0.x` のマイナーバージョン間ではインターフェースや出力形式が変わる可能性がある。タグは `vX.Y.Z` 形式（例: `v0.1.0`）。

## Release Flow

`v*` タグの push で GoReleaser が GitHub Release を自動作成する。リリースノートはタグ間の git コミットから生成される。

次の SemVer タグを作成:

```bash
make tag-patch
make tag-minor
make tag-major
# or
make tag-next TYPE=patch
make tag-next TYPE=minor
make tag-next TYPE=major
```

初回リリース例:

```bash
make tag-minor
git push origin v0.1.0
```

push 後:

- GitHub Actions workflow `release` が自動実行される
- GitHub Release が作成される
- OS/arch アーカイブと `checksums.txt` がアップロードされる

リリースノートをきれいに保つため、コミットメッセージには `feat:`, `fix:`, `refactor:`, `perf:` などのプレフィックスを推奨する。

## ステータス

v0.1.0 リリース。`repo_control` / `code_change` / `app_deployment` / `infra_deployment` の収集・評価・PostgreSQL/filesystem 保存に対応（MVP）。
