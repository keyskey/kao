CREATE TABLE IF NOT EXISTS repo_control_evidence (
    repository   TEXT        NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL,
    payload      JSONB       NOT NULL,
    PRIMARY KEY (repository, collected_at)
);

CREATE TABLE IF NOT EXISTS code_change_evidence (
    repository TEXT NOT NULL,
    pr_number  INT  NOT NULL,
    payload    JSONB NOT NULL,
    PRIMARY KEY (repository, pr_number)
);

CREATE TABLE IF NOT EXISTS app_deployment_evidence (
    application  TEXT        NOT NULL,
    deployed_at  TIMESTAMPTZ NOT NULL,
    payload      JSONB       NOT NULL,
    PRIMARY KEY (application, deployed_at)
);

CREATE TABLE IF NOT EXISTS infra_deployment_evidence (
    repository  TEXT        NOT NULL,
    workspace   TEXT        NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL,
    payload     JSONB       NOT NULL,
    PRIMARY KEY (repository, workspace, applied_at)
);

CREATE TABLE IF NOT EXISTS control_evaluations (
    control_id    TEXT        NOT NULL,
    resource_type TEXT        NOT NULL,
    resource_id   TEXT        NOT NULL,
    evaluated_at  TIMESTAMPTZ NOT NULL,
    payload       JSONB       NOT NULL,
    PRIMARY KEY (control_id, resource_type, resource_id, evaluated_at)
);

CREATE INDEX IF NOT EXISTS idx_code_change_merged_at
    ON code_change_evidence ((payload->>'merged_at'));

CREATE INDEX IF NOT EXISTS idx_code_change_commit_sha
    ON code_change_evidence ((payload->>'commit_sha'));
