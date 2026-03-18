# dq Skills Index

## Core

The main skill covering all dq commands, flags, and patterns.

| Skill | Description |
|-------|-------------|
| [dq](dq.SKILL.md) | Agent-first database CLI: connections, queries, schema, annotations, discover. |

## Helpers

Quick-reference skills for common single operations.

| Skill | Description |
|-------|-------------|
| [dq-overview](dq-overview.md) | Agent orientation: quick start, key commands, output formats. |
| [dq-query](dq-query.md) | Query patterns: pagination, field filtering, dry-run, EXPLAIN, export. |
| [dq-troubleshooting](dq-troubleshooting.md) | Troubleshooting: connection issues, schema investigation, SQL diagnostics. |

## Recipes

Multi-step workflows with real commands. Each recipe is a complete playbook an agent can follow end-to-end.

### Discovery & Orientation

| Skill | Description |
|-------|-------------|
| [recipe-cold-start](recipe-cold-start.md) | First encounter with an unknown database: connect, discover, flag PII, annotate key tables. |
| [recipe-find-table](recipe-find-table.md) | Locate a table from a vague user description (fuzzy search, column search, annotate result). |
| [recipe-trace-relationships](recipe-trace-relationships.md) | Map foreign key chains from a table to understand the data model. |

### Data Exploration

| Skill | Description |
|-------|-------------|
| [recipe-data-profiling](recipe-data-profiling.md) | Profile a table: row count, null rates, distinct counts, min/max per column. |
| [recipe-sample-and-summarize](recipe-sample-and-summarize.md) | Get representative rows, summarize value distributions, flag anomalies. |

### Safety & Impact Analysis

| Skill | Description |
|-------|-------------|
| [recipe-query-impact-analysis](recipe-query-impact-analysis.md) | "Is this query safe?" EXPLAIN + dry-run + cascade check + risk assessment. No execution. |
| [recipe-safe-mutation](recipe-safe-mutation.md) | Dry-run, confirm, execute pattern for INSERT/UPDATE/DELETE. |
| [recipe-safe-backfill](recipe-safe-backfill.md) | Batch update pattern: preview, execute in chunks, track progress. |

### DBA & Performance

| Skill | Description |
|-------|-------------|
| [recipe-slow-query-investigation](recipe-slow-query-investigation.md) | Find slow queries via pg_stat_activity, EXPLAIN them, suggest fixes. |
| [recipe-table-health-check](recipe-table-health-check.md) | Table health: size, bloat, dead tuples, index usage, missing indexes. |
| [recipe-find-missing-indexes](recipe-find-missing-indexes.md) | Identify columns in WHERE/JOIN without indexes, generate CREATE INDEX suggestions. |
| [recipe-orphan-check](recipe-orphan-check.md) | Find rows with broken foreign key references (dangling FKs). |
