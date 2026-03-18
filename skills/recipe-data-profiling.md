---
name: recipe-data-profiling
description: Profile a table's data quality and distribution. Compute row counts, null rates, distinct counts, and min/max per column in a single query, flag anomalies, annotate findings, and present a summary table.
---

# Recipe: Data Profiling — Assess a Table's Data Quality and Distribution

Use this recipe when the user wants to understand a table's data: how complete are the columns, what are the value ranges, which columns have low cardinality or high null rates. The goal is to run a single efficient query, flag anything suspicious, annotate what you learn, and present a clean summary.

## Step 1 — Describe the table to get column names and types

Before profiling data, you need the exact column names and types. This also tells you which columns are numeric (safe for MIN/MAX arithmetic) versus text or boolean.

```bash
dq schema describe -c <connection> --table <table> --output json
```

From the output, categorize columns:

| Category | Types | Profiling approach |
|---|---|---|
| Numeric | `INT`, `INT2`, `INT4`, `INT8`, `FLOAT4`, `FLOAT8`, `NUMERIC`, `DECIMAL` | COUNT, COUNT DISTINCT, MIN, MAX, AVG |
| Text | `VARCHAR`, `TEXT`, `CHAR`, `BPCHAR` | COUNT, COUNT DISTINCT, MIN(LENGTH), MAX(LENGTH) |
| Temporal | `TIMESTAMP`, `TIMESTAMPTZ`, `DATE`, `TIME` | COUNT, COUNT DISTINCT, MIN, MAX |
| Boolean | `BOOL`, `BOOLEAN` | COUNT, COUNT DISTINCT (expect 2-3: true/false/null) |
| Binary/JSON | `BYTEA`, `JSON`, `JSONB`, `BLOB` | COUNT only (MIN/MAX not meaningful) |

## Step 2 — Run a single profiling SQL query

Build one query that computes all metrics for all columns in a single table scan. This is far more efficient than running per-column queries.

**PostgreSQL example** for a table with columns `id`, `email`, `status`, `created_at`, `deleted_at`:

```bash
dq postgres -c <connection> "SELECT
  COUNT(*) AS total_rows,
  COUNT(id) AS id_non_null,
  COUNT(DISTINCT id) AS id_distinct,
  MIN(id) AS id_min,
  MAX(id) AS id_max,
  COUNT(email) AS email_non_null,
  COUNT(DISTINCT email) AS email_distinct,
  MIN(LENGTH(email)) AS email_min_len,
  MAX(LENGTH(email)) AS email_max_len,
  COUNT(status) AS status_non_null,
  COUNT(DISTINCT status) AS status_distinct,
  MIN(status) AS status_min,
  MAX(status) AS status_max,
  COUNT(created_at) AS created_at_non_null,
  COUNT(DISTINCT created_at) AS created_at_distinct,
  MIN(created_at) AS created_at_min,
  MAX(created_at) AS created_at_max,
  COUNT(deleted_at) AS deleted_at_non_null,
  COUNT(DISTINCT deleted_at) AS deleted_at_distinct,
  MIN(deleted_at) AS deleted_at_min,
  MAX(deleted_at) AS deleted_at_max
FROM <table>" --output json
```

> **MySQL differences:**
> - Syntax is identical. `COUNT(DISTINCT col)` and `MIN`/`MAX` work the same way.
> - Use `CHAR_LENGTH(col)` instead of `LENGTH(col)` for multi-byte-safe string length.
>
> **SQLite differences:**
> - `COUNT(DISTINCT col)` works.
> - `LENGTH(col)` returns byte length for BLOBs but character length for TEXT.
> - No native `TIMESTAMPTZ` — dates are stored as TEXT or INTEGER. MIN/MAX still work on ISO-8601 text dates.

**For tables with many columns (>15):** Split into two or three queries to avoid excessively wide result sets. Group columns logically (e.g., identifiers, timestamps, business fields).

**For very large tables:** Add a sampling clause to avoid full table scans:

```bash
# PostgreSQL — use TABLESAMPLE for approximate profiling
dq postgres -c <connection> "SELECT COUNT(*) AS total_rows, ... FROM <table> TABLESAMPLE SYSTEM(1)" --output json
```

> **MySQL:** Use `ORDER BY RAND() LIMIT 100000` (slower but works). **SQLite:** No built-in sampling; use `WHERE rowid % 100 = 0` as an approximation.

## Step 3 — Calculate null rates

For each column, compute the null rate from the query results:

```
null_rate = 1 - (column_non_null / total_rows)
```

A null rate of 0.0 means fully populated. A null rate of 1.0 means entirely null.

Also compute cardinality ratio for context:

```
cardinality_ratio = column_distinct / total_rows
```

A ratio near 1.0 means nearly unique values (like an ID or email). A ratio near 0.0 means very few distinct values (like a status enum or boolean).

## Step 4 — Flag anomalies

Scan the computed metrics for patterns that deserve attention:

| Condition | Flag | Why it matters |
|---|---|---|
| Null rate **> 50%** | High null rate | Column may be optional, deprecated, or poorly populated |
| Null rate **= 100%** | Entirely null | Column may be unused or a migration artifact |
| Distinct count **= 1** | Single value | Column may be a constant, default, or dead field |
| Distinct count **< 10** on a non-boolean | Low cardinality | Likely an enum or status — check for valid values |
| Distinct count **= total_rows** | Fully unique | Candidate for a unique constraint or natural key |
| MIN **= MAX** on a numeric column | No variance | All values are the same — check if intentional |
| MIN date **is very old** (e.g., 1970-01-01) | Suspicious date | Possible epoch-zero defaults or bad data |
| MAX date **is in the future** | Future date | Possible data entry error or scheduled records |
| `*_id` column with **low distinct count** relative to referenced table | Skewed distribution | Most rows reference a small number of parent records |

For low-cardinality columns, fetch the actual distinct values to understand the enum:

```bash
dq postgres -c <connection> "SELECT <column>, COUNT(*) AS count FROM <table> GROUP BY <column> ORDER BY count DESC" --output json --limit 20
```

## Step 5 — Annotate notable findings

Persist the most important findings so future sessions have immediate context.

```bash
# Annotate columns with notable characteristics
dq annotate set -c <connection> --table <table> --column deleted_at --note "Soft-delete marker. 78% null (most records active). Non-null means deleted."
dq annotate set -c <connection> --table <table> --column status --note "Enum: active=45K, pending=2K, suspended=300, deactivated=150. Low cardinality."
dq annotate set -c <connection> --table <table> --column legacy_field --note "100% null. Likely deprecated — confirm with team before using."

# Annotate the table itself with a profiling summary
dq annotate set -c <connection> --table <table> --note "Profiled on <date>. ~<N> rows. Key: <pk>. Notable: deleted_at 78% null (soft deletes), status is 4-value enum."
```

Keep annotations factual and concise. They will appear in every `dq discover` and `dq schema describe` call.

## Step 6 — Present a summary table

Format the results as a table the user can scan quickly. Include all columns, sorted by relevance (flags first, then by position in the schema).

```
Table: public.users (47,200 rows)
Profiled: 2026-03-18

Column         | Type        | Non-Null | Null%  | Distinct | Min           | Max           | Flags
---------------|-------------|----------|--------|----------|---------------|---------------|------
id             | INT8        | 47,200   | 0.0%   | 47,200   | 1             | 49,831        | Unique
email          | VARCHAR     | 47,200   | 0.0%   | 47,198   | 5 chars       | 254 chars     | Near-unique
status         | VARCHAR     | 47,200   | 0.0%   | 4        | -             | -             | Low cardinality
created_at     | TIMESTAMPTZ | 47,200   | 0.0%   | 46,891   | 2022-01-15    | 2026-03-17    |
deleted_at     | TIMESTAMPTZ | 10,384   | 78.0%  | 9,200    | 2022-03-01    | 2026-03-16    | High null rate
legacy_field   | VARCHAR     | 0        | 100.0% | 0        | -             | -             | Entirely null
```

Below the table, highlight actionable findings:

1. **High null rate columns**: `deleted_at` (78%) — this is a soft-delete pattern, which is normal. `legacy_field` (100%) — likely deprecated.
2. **Low cardinality columns**: `status` has only 4 distinct values. Run a GROUP BY to see the distribution.
3. **Uniqueness**: `email` has 47,198 distinct values out of 47,200 rows — 2 duplicates exist. Investigate if a unique constraint is intended.
4. **Date ranges**: `created_at` spans 2022-01-15 to 2026-03-17, showing ~4 years of data.

End with suggested next steps based on what the profiling revealed.
