# KAO Design Document

Version: v1.0

Status: Accepted

---

# 1. Overview

KAO (花押) は GitHub + Kubernetes + Argo CD + Terraform 環境において、変更管理統制 (Change Management Controls) を継続的に評価するための CLI である。

KAO はソフトウェア変更およびインフラ変更について、以下を証跡として生成・評価する。

- 誰が変更したか
- 誰が承認したか
- 何が本番へ反映されたか
- その変更が適切な統制のもとで実施されたか

| 責務 | 内容 |
|---|---|
| 1 | 証跡を収集する |
| 2 | 証跡を評価する |
| 3 | 評価結果を出力する |

---

# 2. Relationship with HADO

| | HADO | KAO |
|---|---|---|
| 目的 | 変更を本番へ出してよいか判断する | 本番へ出た変更が適切な統制のもとで実施されたか確認する |
| 責務 | Product / System Quality, Pre-release Validation | Process / Governance Quality, Change Management Validation |

---

# 3. Goals and Non-Goals

## Goals

| ID | 内容 |
|---|---|
| G1 | Repository 変更統制を評価する |
| G2 | コード変更統制を評価する |
| G3 | Application Deployment 変更統制を評価する |
| G4 | Infrastructure Deployment 変更統制を評価する |
| G5 | Production Change Traceability を保証する |
| G6 | 監査可能な証跡を生成する |
| G7 | 低運用コストで継続運用可能である |

## Non-Goals

| ID | 内容 |
|---|---|
| NG1 | リアルタイム検知 |
| NG2 | SOC2 取得支援 |
| NG3 | Vanta 代替 |
| NG4 | インシデント管理 |
| NG5 | SLO 評価 |
| NG6 | Datadog 設定評価 |
| NG7 | IAM 評価 |
| NG8 | 通知機能 |
| NG9 | Dashboard 機能 |
| NG10 | Approval Workflow |
| NG11 | Web UI |

---

# 4. Design Principles

| Principle | 内容 |
|---|---|
| Lightweight | CLI のみ提供する |
| Stateless | 状態を持たない |
| Storage Agnostic | KAO 自身はデータを保持しない。証跡・評価結果の永続化は `storage` backend（§9, §17）で差し替える |
| Execution Agnostic | 実行基盤を持たない |
| GitHub Native | GitHub を前提とする |
| Argo CD Native | Argo CD を前提とする |
| Terraform Native | インフラ変更管理に Terraform を利用していることを前提とする |
| Evidence First | イベントではなく証跡を扱う |
| Evaluate Only | 証跡収集と評価のみを責務とする |
| Provider Agnostic Commands | コマンド名は統制ドメインの汎用名とし、接続先は Configuration で指定する |
| Low Operational Cost | Webhook や常駐サーバーを必要としない |
| Policy as Configuration | Configuration ファイルが統制ポリシーの説明資料として機能する（§9 参照） |
| Secrets via Environment | 認証情報は Configuration に書かず、環境変数経由でのみ渡す（§9 Secrets） |

---

# 5. Supported Providers (v1)

v1 MVP の外部連携範囲。CircleCI は v1.1 で追加する（§20）。

| Provider | Evidence | 主な収集データ | v1 |
|---|---|---|---|
| GitHub | `repo_control`, `code_change` | Branch Protection, CODEOWNERS, Teams, PR, Reviews, GitHub Actions | ○ |
| Argo CD | `app_deployment` | Application Deployment History, Source Repository | ○ |
| Terraform | `infra_deployment` | GitHub Actions 上の apply 実行履歴 | ○ |
| CircleCI | `code_change` (CI) | Workflow Result, Job URL | v1.1 |

---

# 6. Architecture

```mermaid
flowchart TD
    subgraph providers["Providers (evidence.*.provider)"]
        GH[GitHub]
        AC[Argo CD]
        TF[Terraform]
    end

    subgraph kao["KAO"]
        RUN["kao run daily"]
        COLLECT["kao collect *"]
        EVAL["kao evaluate"]
    end

    subgraph storage["Storage (storage.backend)"]
        DB[(Postgres)]
    end

    GH --> COLLECT
    AC --> COLLECT
    TF --> COLLECT

    RUN --> COLLECT
    COLLECT --> DB
    RUN --> EVAL
    DB --> EVAL
    EVAL --> DB
```

証跡の on-the-wire 形式は JSONL。本番では `storage` backend への読み書きが基本パターンである。

---

# 7. Execution Model

KAO は状態を持たない。実行基盤は GitHub Actions、Kubernetes CronJob、Cloud Run Job、ローカル実行を想定する。

## Schedule

全 evidence を **毎日 00:00 UTC の単一ジョブ** で収集・評価する。Cron からは `kao run daily` を実行する（§8）。

| 順序 | 処理 | 収集対象 | 備考 |
|---|---|---|---|
| 1 | `repo_control` 収集 → storage | ジョブ実行時点のリポジトリ統制状態 | CODEOWNERS と Team Membership を含む |
| 2 | `code_change` 収集 → storage | **前日**に default branch へマージされた PR | `--date` で対象日を指定 |
| 3 | `app_deployment` 収集 → storage | **前日**の本番 Application デプロイ | 同上 |
| 4 | `infra_deployment` 収集 → storage | **前日**の本番 Infrastructure デプロイ | 同上 |
| 5 | `evaluate` ← storage → storage | 手順 1〜4 の証跡を入力として評価 | 結果を storage へ保存 |

`repo_control` を日次で記録する理由:

- スケジュールが 1 本化され運用が単純になる
- CM-008 の参照スナップショットとマージのタイムラグが最大約 24 時間に短縮される
- 同一ジョブ内で収集した `repo_control` を `code_change` 評価に使える

## Operational Assumptions

監査上の運用前提として、以下を組織が維持する。

1. **権限変更の安定性** — PR マージから次回日次ジョブ（最大約 24 時間）の間、当該リポジトリの Team Membership と CODEOWNERS は変更されない。
2. **日次スナップショットの使い方** — 同一日次ジョブで収集した `repo_control` を、そのジョブが評価する `code_change` の CM-008 判定に使用する（§15.6）。
3. **default branch 運用** — 本番リリースは default branch へのマージを経由する。

### 日付境界

`--date` および収集対象の判定は **UTC のカレンダー日**（`YYYY-MM-DD`）で行う。`merged_at` / `deployed_at` / `applied_at` の ISO 8601 タイムスタンプから UTC 日付を抽出し、`TARGET_DATE` と一致する証跡を収集する。

例: `merged_at = 2026-06-05T23:59:00Z` は `2026-06-05`、`2026-06-06T00:00:01Z` は `2026-06-06` として扱う。

Webhook による承認時点の Point-in-Time 再現は行わない（§19 Limitations）。

## 部分失敗とリポジトリ単位の再実行

収集・評価は **repository 単位で独立** して実行する。1 リポジトリの collect が失敗しても他リポジトリの処理は継続する。いずれかが失敗した場合、ジョブ全体の終了コードは非ゼロとする。

失敗したリポジトリのみ、同一 `--date` で再実行する。

```bash
kao run daily --config kao.yaml --date 2026-06-05 --repository trading-api
```

`--repository` を指定した場合、そのリポジトリに属する証跡のみを収集・評価する。storage への upsert は §17.2 の PK に基づき冪等であるため、再実行しても重複しない。

| evidence | `--repository` の絞り込み |
|---|---|
| `repo_control` / `code_change` | リポジトリ名が一致 |
| `app_deployment` | `repository` フィールドが一致 |
| `infra_deployment` | `repository` フィールドが一致 |

---

# 8. Command Reference

全コマンドは `--config <path>` で Configuration ファイルを指定する。省略時はカレントディレクトリの `kao.yaml` を参照する。

## kao run daily（推奨）

日次ジョブのエントリポイント。§7 Schedule の手順 1〜5 を順に実行する。

```bash
kao run daily --config kao.yaml --date 2026-06-05
```

| フラグ | 内容 |
|---|---|
| `--date YYYY-MM-DD` | 評価対象日（前日分の `code_change` / deployment）。省略時は実行日の前日（UTC） |
| `--repository <name>` | 対象リポジトリを限定（**繰り返し指定可**）。省略時は `scope` の全リポジトリ |

内部で `collect` → `storage` へ保存 → `storage` から読み出して `evaluate` → 結果を `storage` へ保存する。§7 の部分失敗時は `--repository` で失敗分のみ再実行する。

## I/O モデル

`collect` / `evaluate` は `--input` / `--output` でデータの行き先を指定する。**省略時は `storage`（`storage.backend`）**。

| 値 | 用途 |
|---|---|
| `storage`（デフォルト） | 本番。`storage` backend へ upsert / クエリ |
| `-` | stdout（JSONL 等）。デバッグ・パイプ用 |
| `<file>` | ファイルへの読み書き。ローカル検証用 |

JSONL は証跡の **シリアライズ形式** であり、本番運用で利用者が手動でファイルを組み立てることを想定しない。

### collect

`code_change` / `app_deployment` / `infra_deployment` は `--date YYYY-MM-DD` で対象日を指定できる。`repo_control` は実行時点の状態を収集するため `--date` を受け付けない。全 `collect` コマンドは `--repository <name>`（繰り返し指定可）で対象リポジトリを限定できる。

| 共通フラグ | 内容 |
|---|---|
| `--repository <name>` | 収集対象リポジトリを限定。省略時は `scope` の全リポジトリ |

| コマンド | デフォルト出力 |
|---|---|
| `kao collect repo-control` | `storage`（`repo_control` Evidence を upsert） |
| `kao collect code-change` | `storage` |
| `kao collect app-deployment` | `storage` |
| `kao collect infra-deployment` | `storage` |

```bash
kao collect code-change --date 2026-06-05              # → storage
kao collect code-change --date 2026-06-05 --output -   # → stdout JSONL
```

### evaluate

| フラグ | 内容 |
|---|---|
| `--date YYYY-MM-DD` | 評価対象日。storage から当該日の deployment 証跡と、必要な `code_change` / `repo_control` を読み出す |
| `--repository <name>` | 評価対象リポジトリを限定（繰り返し指定可） |
| `--input storage` | 入力を storage から取得（**デフォルト**） |
| `--output storage` | 結果を storage へ保存（**デフォルト**） |
| `--format jsonl` | `--output -` 時のシリアライズ形式（`json` / `jsonl` / `csv` / `markdown`） |

storage 入力時、CM-006/007 の結合に必要な過去の `code_change` は backend が `repository` + `commit_sha` および `evaluation.code_change.lookback_days` に基づき自動取得する。

### evaluate ファイル入力（副路）

storage を使わないローカル検証用。本番運用の推奨パスではない。

| フラグ | 内容 |
|---|---|
| `--input files` | 以下のファイルフラグを使用 |
| `--repo-control <file>` | `repo_control` Evidence JSONL |
| `--code-change <file>` | `code_change` Evidence JSONL（繰り返し指定可） |
| `--code-change-dir <dir>` | `code_change` JSONL ディレクトリ（`lookback_days` と併用） |
| `--app-deployment <file>` | `app_deployment` Evidence JSONL |
| `--infra-deployment <file>` | `infra_deployment` Evidence JSONL |

接続先ツールは `evidence.<domain>.provider` で指定する。評価ポリシーは `controls.<domain>` で指定する。永続化先は `storage.backend` で指定する。

---

# 9. Configuration Schema

Configuration は 5 つのトップレベルキーで構成する。

| Key | 責務 | 参照コマンド |
|---|---|---|
| `providers` | ツール接続設定 | `kao collect` |
| `scope` | 収集対象のデフォルト範囲 | `kao collect` |
| `evidence` | 証跡収集設定 | `kao collect` |
| `controls` | 統制評価ポリシー | `kao evaluate` |
| `storage` | 証跡・評価結果の永続化 backend | `kao collect`, `kao evaluate`, `kao run` |

`evidence` と `controls` は統制ドメイン名（`repo_control`, `code_change` など）を共通キーとして対応する。

## Example

```yaml
version: v1

storage:
  backend: postgres
  postgres:
    url_env: DATABASE_URL

providers:
  github:
    org: enechain
    token_env: GITHUB_TOKEN
  argocd:
    server_env: ARGOCD_SERVER
    token_env: ARGOCD_TOKEN

scope:
  defaults:
    repositories:
      include: [trading-api, settlement-api, matching-engine, platform-infra]
      exclude: [sandbox-*, archived-*]

evidence:
  repo_control:
    provider: github

  code_change:
    provider: github
    ci: [github_actions]

  app_deployment:
    provider: argocd
    scope:
      applications:
        include: [trading-api-prod, settlement-api-prod]
        exclude: ["*-staging"]

  infra_deployment:
    provider: terraform
    scope:
      repositories:
        include: [platform-infra]
      workspaces:
        include: [prod-gke, prod-network]
    apply_workflows: [terraform-apply.yml]
    apply_job_names: [terraform apply]

controls:
  repo_control:
    branch_protection:
      enabled: true
      required_reviews: 1
      require_codeowner_review: true
      allow_force_push: false
      allow_branch_deletion: false
    codeowners:
      required: true

  code_change:
    ci:
      require_pass: true
    approver:
      must_be_authorized: true

  app_deployment:
    traceability:
      required: true

  infra_deployment:
    traceability:
      required: true

evaluation:
  code_change:
    lookback_days: 90
```

## Structure

### Secrets

認証情報・接続文字列は **Configuration ファイルに平文で書かない**。`*_env` キーで環境変数名のみを宣言し、KAO は実行時にその環境変数を読む。

| 設定 | 環境変数の例 | 内容 |
|---|---|---|
| `providers.github.token_env` | `GITHUB_TOKEN` | GitHub PAT / App token |
| `providers.argocd.server_env` | `ARGOCD_SERVER` | Argo CD API URL |
| `providers.argocd.token_env` | `ARGOCD_TOKEN` | Argo CD 認証トークン |
| `storage.postgres.url_env` | `DATABASE_URL` | PostgreSQL 接続 URI（下記） |

環境変数は実行基盤のシークレット機構から注入する（Kubernetes Secret、GitHub Actions secrets、Cloud Run secrets 等）。KAO は取得した値をログ出力・証跡・評価結果に含めない。

#### storage.postgres

v1 では `url_env` のみをサポートする。接続 URI は環境変数に **1 本の文字列** として渡す。

```bash
export DATABASE_URL='postgres://kao:********@db.example.com:5432/kao?sslmode=require'
```

- ユーザー名・パスワード・ホスト・DB 名はすべて URI に含める
- Configuration には `url_env: DATABASE_URL` のみ記載し、パスワードを分割設定しない（v1）
- 本番では `sslmode=require`（または組織標準の TLS 設定）を URI に含める

ローカル開発で `filesystem` backend を使う場合、`DATABASE_URL` は不要。

### providers

ツール接続設定。`evidence.<domain>.provider` が `providers.<name>` を参照する。認証は §9 Secrets に従い `*_env` 経由とする。

### scope

`scope.defaults.repositories` は `repo_control` / `code_change` のデフォルト。`evidence.<domain>.scope` で override する。

| Evidence | Scope キー | defaults 継承 |
|---|---|---|
| `repo_control` | `repositories` | する |
| `code_change` | `repositories` | する |
| `app_deployment` | `applications` | しない |
| `infra_deployment` | `repositories`, `workspaces` | `repositories` のみ |

`include` / `exclude` のワイルドカード（`*`）は glob マッチとする。

### evidence

証跡収集設定。`provider`、`scope`（任意）、ドメイン固有の収集設定を持つ。

`code_change.ci` は収集対象 CI provider の一覧。v1 は `github_actions` のみ（`providers.github` を利用）。`circleci` は v1.1（§20）。

`infra_deployment.apply_workflows` / `apply_job_names` は CI 上で Terraform apply とみなす workflow / job の識別子。

### controls

統制評価ポリシー（Policy as Configuration）。§15 Control Framework の期待値を宣言する。閾値のない統制もポリシー宣言として記載する。省略時は §9.1 のデフォルト値を用いる。

### storage

証跡と評価結果の永続化 backend。`providers` と同様に `backend` で実装を選択する。postgres の認証は §9 Secrets に従う。

| backend | v1 | 用途 |
|---|---|---|
| `postgres` | サポート | **本番推奨**。§17 スキーマに upsert。CM-006/007 は SQL で結合 |
| `filesystem` | サポート | ローカル開発。日付別 JSONL をディレクトリに蓄積 |
| `s3` | 将来 | Object Storage への JSONL 保存（§20） |

```yaml
# 本番
storage:
  backend: postgres
  postgres:
    url_env: DATABASE_URL

# ローカル開発
storage:
  backend: filesystem
  filesystem:
    path: ./evidence
```

### evaluation

`kao evaluate` の結合・検索動作を制御する。`--input storage` 時は backend がクエリを実行する。

| キー | 説明 | Default |
|---|---|---|
| `evaluation.code_change.lookback_days` | CM-006/007 結合時に遡って参照する `code_change` の最大日数 | `90` |

## Provider Mapping (v1)

| Evidence | Command | Provider |
|---|---|---|
| `repo_control` | `kao collect repo-control` | `github` |
| `code_change` | `kao collect code-change` | `github` |
| `app_deployment` | `kao collect app-deployment` | `argocd` |
| `infra_deployment` | `kao collect infra-deployment` | `terraform` |

## Control Policy Mapping (v1)

| Control | Configuration | Evidence / Evaluation |
|---|---|---|
| CM-001 | `controls.repo_control.branch_protection.enabled` | `branch_protection.enabled` |
| CM-002 | `controls.repo_control.branch_protection.required_reviews` | `repo_control`: 設定値。`code_change`: 承認件数（§15.2） |
| CM-003 | `controls.repo_control.branch_protection.require_codeowner_review` | `branch_protection.require_codeowner_review` |
| CM-004 | `controls.repo_control.branch_protection.allow_force_push` | `branch_protection.allow_force_push` |
| CM-005 | `controls.code_change.ci.require_pass` | `ci.passed` |
| CM-006 | `controls.app_deployment.traceability.required` | Required Chain（§15.5） |
| CM-007 | `controls.infra_deployment.traceability.required` | Required Chain（§15.5） |
| CM-008 | `controls.code_change.approver.must_be_authorized` | Approver ∈ resolved users（§15.6） |
| CM-009 | `controls.repo_control.branch_protection.allow_branch_deletion` | `branch_protection.allow_branch_deletion` |
| CM-010 | `controls.repo_control.codeowners.required` | `codeowners.exists` |

### 9.1 controls デフォルト値

| Path | Default |
|---|---|
| `controls.repo_control.branch_protection.enabled` | `true` |
| `controls.repo_control.branch_protection.required_reviews` | `1` |
| `controls.repo_control.branch_protection.require_codeowner_review` | `true` |
| `controls.repo_control.branch_protection.allow_force_push` | `false` |
| `controls.repo_control.branch_protection.allow_branch_deletion` | `false` |
| `controls.repo_control.codeowners.required` | `true` |
| `controls.code_change.ci.require_pass` | `true` |
| `controls.code_change.approver.must_be_authorized` | `true` |
| `controls.app_deployment.traceability.required` | `true` |
| `controls.infra_deployment.traceability.required` | `true` |
| `evaluation.code_change.lookback_days` | `90` |

---

# 10. Evidence Types

KAO は 4 種類の証跡のみを扱う。`type` フィールド・ドメイン名・コマンド名はすべて同じ識別子を用いる。

| type / domain | コマンド | 説明 | 適用 CM |
|---|---|---|---|
| `repo_control` | `kao collect repo-control` | リポジトリ統制状態の日次スナップショット | CM-001〜004, CM-009, CM-010, CM-008（参照） |
| `code_change` | `kao collect code-change` | default branch へマージされた PR の変更証跡 | CM-002, CM-005, CM-008, CM-006/007（参照） |
| `app_deployment` | `kao collect app-deployment` | 本番 Application デプロイ証跡 | CM-006 |
| `infra_deployment` | `kao collect infra-deployment` | 本番 Infrastructure デプロイ証跡 | CM-007 |

全 Evidence JSON は `schema_version: "v1"` を含む。

---

# 11. repo_control Evidence

リポジトリ統制状態の日次スナップショット。Branch Protection、CODEOWNERS、Team Membership 解決結果を含む。

日次ジョブの先頭で収集する。`collected_at` がスナップショットの基準時刻。CM-008 は同一日次ジョブで収集した `repo_control` を参照する（§15.6）。

```json
{
  "schema_version": "v1",
  "type": "repo_control",
  "provider": "github",
  "repository": "trading-api",
  "default_branch": "main",
  "branch_protection": {
    "enabled": true,
    "required_reviews": 1,
    "require_codeowner_review": true,
    "allow_force_push": false,
    "allow_branch_deletion": false
  },
  "codeowners": {
    "exists": true,
    "path": ".github/CODEOWNERS",
    "rules": [
      {
        "pattern": "*",
        "owners": [
          {
            "type": "team",
            "name": "@enechain/sre",
            "resolved_users": ["alice", "bob", "carol"]
          }
        ]
      }
    ]
  },
  "collected_at": "2026-06-06T00:00:00Z"
}
```

---

# 12. code_change Evidence

default branch へマージされた PR の変更証跡。対象は `--date` で指定した日（デフォルト: 前日 UTC）に `merged_at` を持つ PR に限定する。

| フィールド | 説明 |
|---|---|
| `commit_sha` | マージコミットの完全 SHA（40 文字） |
| `merged_by` | マージ操作を実行したユーザー |
| `approvals` | 承認レビューの一覧（`reviewer`, `state`, `submitted_at`） |
| `approval_count` | `state: APPROVED` の重複除き `reviewer` 数。§15.2 CM-002 の判定に使用 |
| `ci.runs` | PR へのコミット push ごとに起動した CI workflow run の一覧 |
| `ci.passed` | §15.3 CM-005 の判定結果 |

`approvals` は `state: APPROVED` のレビューのみを含む。同一 `reviewer` の複数承認は `approval_count` で 1 件として数える。CM-008 の評価対象は `approvals[].reviewer` とする（`merged_by` とは別）。

```json
{
  "schema_version": "v1",
  "type": "code_change",
  "provider": "github",
  "repository": "trading-api",
  "default_branch": "main",
  "pr_number": 123,
  "url": "https://github.com/org/repo/pull/123",
  "author": "alice",
  "merged_by": "bob",
  "approvals": [
    {
      "reviewer": "carol",
      "state": "APPROVED",
      "submitted_at": "2026-06-05T14:30:00Z"
    }
  ],
  "approval_count": 1,
  "commit_sha": "abc123def4567890abc123def4567890abc12345",
  "ci": {
    "passed": true,
    "runs": [
      {
        "provider": "github_actions",
        "workflow": "CI",
        "run_id": "12345678",
        "status": "success",
        "commit_sha": "abc123def4567890abc123def4567890abc12345",
        "url": "https://github.com/org/repo/actions/runs/12345678"
      }
    ]
  },
  "merged_at": "2026-06-05T16:00:00Z"
}
```

---

# 13. app_deployment Evidence

本番 Application のデプロイ証跡。Argo CD Application のデプロイ履歴から収集する。

| フィールド | 説明 |
|---|---|
| `repository` | Argo CD Application の `spec.source` から取得した Git リポジトリ名 |
| `revision` | デプロイされた Git コミットの完全 SHA |
| `environment` | `evidence.app_deployment.scope` で指定した本番環境（Application 名または label で判定） |

```json
{
  "schema_version": "v1",
  "type": "app_deployment",
  "provider": "argocd",
  "application": "trading-api-prod",
  "repository": "trading-api",
  "environment": "production",
  "revision": "abc123def4567890abc123def4567890abc12345",
  "sync_initiated_by": "alice",
  "status": "succeeded",
  "deployed_at": "2026-06-05T16:05:00Z"
}
```

---

# 14. infra_deployment Evidence

GitHub Actions 上で実行された Terraform apply の証跡（v1）。`evidence.infra_deployment.apply_workflows` または `apply_job_names` に一致する成功した workflow run を収集する。

### 収集手順

1. `scope.repositories` の各リポジトリについて、対象日に完了した CI run を列挙する
2. workflow 名 / job 名が `apply_workflows` / `apply_job_names` に一致する run を apply 実行とみなす
3. マッチした apply job ごとに commit SHA、実行者、URL、workspace を記録する。workspace は GHA matrix job 名から取得する（共有 workflow で apply job 名が固定でも、GitHub が matrix 次元を `(dim1, dim2, ...)` 形式で suffix する）。`workspace` は先頭次元とし、KAO は `workspace_job_name_pattern`（デフォルト: `terraform apply \(([^,)]+)`）で抽出する

```json
{
  "schema_version": "v1",
  "type": "infra_deployment",
  "provider": "terraform",
  "repository": "platform-infra",
  "workspace": "prod-gke",
  "environment": "production",
  "commit_sha": "def4567890abcdef4567890abcdef4567890ab",
  "execution": {
    "ci_provider": "github_actions",
    "workflow": "terraform-apply.yml",
    "run_id": "87654321",
    "status": "succeeded",
    "triggered_by": "alice",
    "url": "https://github.com/org/repo/actions/runs/87654321"
  },
  "applied_at": "2026-06-05T16:05:00Z"
}
```

将来 OpenTofu / Pulumi 等をサポートする場合も `type` とコマンドは変更せず、`evidence.infra_deployment.provider` の追加で拡張する。

---

# 15. Control Framework

## 15.1 Control 一覧

| ID | 名称 | Severity | Configuration | Evaluation |
|---|---|---|---|---|
| CM-001 | Default Branch Must Be Protected | HIGH | `controls.repo_control.branch_protection.enabled` | `branch_protection.enabled == configured_value` |
| CM-002 | Review Required | HIGH | `controls.repo_control.branch_protection.required_reviews` | §15.2（設定 + 承認件数） |
| CM-003 | CODEOWNER Review Required | HIGH | `controls.repo_control.branch_protection.require_codeowner_review` | `branch_protection.require_codeowner_review == configured_value` |
| CM-004 | Force Push Disabled | HIGH | `controls.repo_control.branch_protection.allow_force_push` | `branch_protection.allow_force_push == configured_value` |
| CM-005 | CI Must Pass Before Merge | HIGH | `controls.code_change.ci.require_pass` | `ci.passed == configured_value`（§15.3） |
| CM-006 | Application Deployment Traceable | CRITICAL | `controls.app_deployment.traceability.required` | Required Chain（§15.5） |
| CM-007 | Infrastructure Deployment Traceable | CRITICAL | `controls.infra_deployment.traceability.required` | Required Chain（§15.5） |
| CM-008 | Approver Must Be Authorized | CRITICAL | `controls.code_change.approver.must_be_authorized` | §15.6 |
| CM-009 | Branch Deletion Disabled | HIGH | `controls.repo_control.branch_protection.allow_branch_deletion` | `branch_protection.allow_branch_deletion == configured_value` |
| CM-010 | CODEOWNERS File Required | HIGH | `controls.repo_control.codeowners.required` | `codeowners.exists == configured_value` |

## 15.2 CM-002: Review Required

CM-002 は **リポジトリ設定** と **PR 実績** の両方を評価する。閾値は `controls.repo_control.branch_protection.required_reviews` で共通とする。

### repo_control（設定）

```text
branch_protection.required_reviews >= configured_value
```

`resource_type: repo_control`（§15.4）

### code_change（承認件数）

```text
approval_count >= configured_value
```

`approval_count` は `approvals` のうち `state: APPROVED` である `reviewer` の重複除き件数。収集時に Evidence へ記録する。

`resource_type: code_change`, `resource_id: {repository}#{pr_number}`

## 15.3 CM-005: CI 判定

`evidence.code_change.ci` に列挙した各 provider について、**マージコミット（`commit_sha`）に対して実行され成功した CI workflow run** がすべて `status: success` であれば `ci.passed = true` とする。

- 個別のテストステップ名や job 名の設定は不要
- 中間コミットで失敗した run は評価対象外。マージコミット時点で成功していればよい
- 設定した provider について、マージコミット向けの run が 1 件でも失敗または欠落していれば `ci.passed = false`

## 15.4 repo_control 評価の意味

CM-001, CM-003, CM-004, CM-009, CM-010 および CM-002 の repo_control 部分は **日次 `repo_control` スナップショットに対する継続的コンプライアンス評価** である。マージ時点のリポジトリ状態を再現するものではない。

| 項目 | 値 |
|---|---|
| `resource_type` | `repo_control` |
| `resource_id` | `{repository}@{collected_at の UTC 日付}`（例: `trading-api@2026-06-06`） |

`code_change` に対する CM-002（承認件数）/ CM-005 / CM-008 は、当該 PR が収集された日次ジョブで評価する。CM-006/007 の結合で過去の `code_change` を参照する場合、結合に必要なフィールド（`author`, `approvals`, `approval_count`, `ci`）を読み取るのみで、CM-002/005/008 を再評価しない。

## 15.5 CM-006 / CM-007: トレーサビリティ結合

マージとデプロイに数日の差があっても正確に結合できるよう、**結合キーは日付ではなく `repository` + 完全 `commit_sha`** とする。`code_change` はマージ日とデプロイ日が異なっていても、SHA が一致すれば結合できる。

### code_change 証跡の参照

本番（`--input storage`）では backend が `repository` + `commit_sha` で過去の `code_change` を自動取得する。`postgres` backend では §17 のテーブルを SQL 結合する。

| 入力モード | 過去 `code_change` の供給方法 |
|---|---|
| `storage`（デフォルト） | backend が `lookback_days` 内でクエリ |
| `files`（副路） | `--code-change` 繰り返し、`--code-change-dir` + `lookback_days` |

日次 `collect` は前日分のみ storage へ upsert する。過去分の参照は evaluate 時に backend が行う。保持期間が `lookback_days` を超える古いデプロイは結合できないため、storage の保持ポリシーを `lookback_days` 以上に設定する。

### Join Keys

| 結合 | キー |
|---|---|
| app → code | `app_deployment.repository == code_change.repository` かつ `app_deployment.revision == code_change.commit_sha` |
| infra → code | `infra_deployment.repository == code_change.repository` かつ `infra_deployment.commit_sha == code_change.commit_sha` |

複数候補がある場合は `merged_at` が最も新しい `code_change` を採用する。

### CM-006 Required Chain

```text
app_deployment → code_change (sha+repo) → author → approvals → ci.runs[].url
```

1. `app_deployment` について join key で `code_change` を検索する
2. `code_change.author` が存在する
3. `code_change.approval_count` が CM-002 の `required_reviews` を満たす
4. `code_change.ci.runs` に `url` を持つ run が 1 件以上存在する

いずれかのステップで欠落した場合は `FAILED`。`reason` に欠落ノードを記録する。

### CM-007 Required Chain

```text
infra_deployment → code_change (sha+repo) → author → approvals → execution.url
```

CM-006 と同様。CI run URL は `infra_deployment.execution.url` を用いる。

## 15.6 CM-008: 承認者権限

### 参照スナップショット

`code_change` と **同一日次ジョブ** で収集した `repo_control` を参照する。`repository` が一致し、かつ `repo_control.collected_at` が `code_change.merged_at` より後の、同一ジョブ内のスナップショットを使用する（§7 Operational Assumptions）。

例: 6 月 6 日 00:00 UTC のジョブで `code_change`（`merged_at = 2026-06-05`）を評価する場合、同ジョブで収集した `repo_control`（`collected_at = 2026-06-06T00:00:00Z`）を参照する。

### 判定

`code_change.approvals` の各 `reviewer` について、参照 `repo_control` の全 CODEOWNERS rule の `resolved_users` の和集合に含まれることを要求する。

```text
∀ reviewer ∈ approvals: reviewer ∈ ⋃(rule.resolved_users)
```

ファイルパス単位の CODEOWNERS マッチングは v1 では行わない。リポジトリ単位の承認権限者集合による評価とする。

### CM-003 との関係

| Control | 観点 |
|---|---|
| CM-003 | GitHub branch protection 設定として `require_codeowner_review` が有効か（`repo_control` 証跡） |
| CM-008 | 実際の承認者が CODEOWNERS 上の承認権限者集合に含まれるか（`code_change` 証跡） |

CM-008 は「GitHub の CODEOWNER レビューが付いていたか」を直接検証しない。承認者が `resolved_users` の和集合に含まれることをもって、組織が定義した承認権限者による承認であったとみなす。CM-003 と併用することで、設定と実績の両面を評価する。

---

# 16. Evaluation Output

Evaluation は **Control × Resource** 単位で生成する。`schema_version: "v1"` を含む。

```json
{
  "schema_version": "v1",
  "control_id": "CM-008",
  "resource_type": "code_change",
  "resource_id": "trading-api#123",
  "status": "FAILED",
  "severity": "CRITICAL",
  "expected": "carol ∈ repo_control resolved_users",
  "actual": "carol ∉ resolved_users",
  "reason": "Approver is not included in CODEOWNERS resolved users",
  "evidence_refs": [
    { "type": "code_change", "resource_id": "trading-api#123" },
    { "type": "repo_control", "resource_id": "trading-api", "collected_at": "2026-06-06T00:00:00Z" }
  ],
  "evaluated_at": "2026-06-06T00:00:00Z"
}
```

| フィールド | 説明 |
|---|---|
| `expected` / `actual` | 監査再現のための期待値・実測値サマリ |
| `evidence_refs` | 判定に使用した証跡への参照 |

---

# 17. Storage

## 17.1 Backend インターフェース

KAO は実行ごとに stateless だが、証跡と評価結果は `storage` backend を経由して永続化する。backend は以下の操作を実装する。

| 操作 | 呼び出し元 | 説明 |
|---|---|---|
| `PutEvidence(type, records)` | `kao collect` | 証跡を upsert（§17.2 の PK で冪等） |
| `QueryEvidence(type, filter)` | `kao evaluate` | 対象日の証跡を取得 |
| `QueryCodeChangeForJoin(repos, shas, lookback)` | `kao evaluate` | CM-006/007 用。`repository` + `commit_sha` で検索 |
| `PutEvaluations(records)` | `kao evaluate` | 評価結果を保存 |

`filesystem` backend は JSONL ファイルへの読み書きで上記を実装する。`postgres` backend は §17.2 のテーブルを使用する。

## 17.2 スキーマ（postgres）

Evidence 1 件に対し Evaluation は N 件（Control 数分）の 1:N 関係。

```mermaid
flowchart TD
    RCE[repo_control_evidence]
    CCE[code_change_evidence]
    ADE[app_deployment_evidence]
    IDE[infra_deployment_evidence]
    CE[control_evaluations]

    RCE -.->|参照| CE
    CCE --> CE
    ADE --> CE
    IDE --> CE
```

| テーブル | 主キー例 |
|---|---|
| `repo_control_evidence` | `repository` + `collected_at` |
| `code_change_evidence` | `repository` + `pr_number` |
| `app_deployment_evidence` | `application` + `deployed_at` |
| `infra_deployment_evidence` | `repository` + `workspace` + `applied_at` |
| `control_evaluations` | `control_id` + `resource_type` + `resource_id` + `evaluated_at` |

各テーブルの `payload` 列に §10〜14 の JSON 証跡を格納する。KAO はマイグレーション SaaS を提供しない。利用者が §17.2 に基づきテーブルを用意する。

---

# 18. Example Execution

## 本番（推奨）

毎日 00:00 UTC の Cron。`storage.backend: postgres`。

```bash
kao run daily --config kao.yaml --date 2026-06-05
```

## 部分失敗時の再実行

`trading-api` のみ collect が失敗した場合、同一日付で当該リポジトリのみ再実行する。

```bash
kao run daily --config kao.yaml --date 2026-06-05 --repository trading-api
```

## ローカル開発（filesystem backend）

```bash
kao run daily --config kao.local.yaml --date 2026-06-05
```

`kao.local.yaml` は `storage.backend: filesystem` を指定する（§9）。

## デバッグ（stdout / ファイル副路）

個別コマンドを JSONL で確認する場合。

```bash
kao collect code-change --date 2026-06-05 --output - | head
kao evaluate --input files \
  --repo-control repo_control.jsonl \
  --code-change code_change.jsonl \
  --app-deployment app_deployment.jsonl \
  --infra-deployment infra_deployment.jsonl \
  --output - --format jsonl
```

---

# 19. Limitations

| 制約 | 内容 | 緩和策 |
|---|---|---|
| Webhook 不使用 | 承認・権限変更の瞬間を捉えない | 日次スナップショット + マージ後 24 時間以内の権限安定運用（§7） |
| CODEOWNERS マッチング | ファイルパス単位の評価はしない | リポジトリ単位の resolved_users 和集合で評価（§15.6） |
| マージ・デプロイの日数差 | 同日分の `code_change` だけでは結合できない | SHA 結合 + storage 上の過去 `code_change` と lookback（§15.5, §17） |
| CI 判定 | テストステップ名は評価しない | マージコミット向け workflow run の成功で判定（§15.3） |
| Terraform 収集 | state ファイルは直接読まない | CI run ベースの apply 検出（§14） |

---

# 20. Future Work

## Control Groups

Monitoring Governance（Datadog Monitor / SLO）、Security Governance（IAM / Audit Logs）、Reliability Governance（Runbook / Oncall / PRR）は独立した Control Group として将来追加する。

## Providers (v1.1)

| Provider | 内容 |
|---|---|
| CircleCI | `code_change.ci` への `circleci` 追加。`providers.circleci.token_env` で認証 |

## Storage Backends

| backend | 内容 |
|---|---|
| `s3` | Object Storage への JSONL 保存。`filesystem` と同様の list + read で結合 |
| カスタム driver | §17.1 インターフェースを実装した利用者独自 backend |

KAO v1 の責務は変更管理統制評価に限定する。ダッシュボード・通知・マイグレーション SaaS は Non-Goals（§3）のままとする。

---

# 21. Philosophy

HADO is the proof before release. KAO is the signature after release.

HADO verifies whether a change should be released. KAO verifies whether a change was released under proper governance.
