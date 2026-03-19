---
name: dq-query-impact-analysis
description: Analyze a SQL query for safety before execution. Evaluate query plan, blast radius, cascading effects, missing indexes, and compile a risk assessment. Use when the user asks "is this query safe?" or before running any unfamiliar mutation.
---

# Recipe: Query Impact Analysis

Use this recipe when the user asks "is this query safe to run?" or when you need to evaluate a query before execution. **Do NOT execute the query** -- only analyze it.

This is a defensive workflow. The goal is to give the user a clear risk assessment (safe / caution / dangerous) with specific, actionable reasons.

## Prerequisites

- A working connection (verify with `dq connection list --output json`)
- The query to analyze (provided by the user)
- Schema context (run `dq discover -c <connection> --output json` if you haven't already)

## Steps

### 1. Classify the query

Before any analysis, classify the query type:

- **SELECT**: Read-only, lower risk but can still cause performance issues
- **INSERT**: Additive, moderate risk (constraint violations, bloat)
- **UPDATE**: Mutation, higher risk (wrong WHERE clause = data corruption)
- **DELETE**: Destructive, highest risk (data loss, cascading deletes)
- **DDL** (ALTER, DROP, CREATE): Schema change, very high risk
- **TRUNCATE**: Destructive, maximum risk (no WHERE clause possible)

**Decision point**: If the query is DDL (especially `DROP` or `TRUNCATE`), immediately flag as **dangerous** and recommend extreme caution. DDL operations are often not transactional.

### 2. Get the query plan with EXPLAIN

Use the `--explain` flag to get the query plan without executing the query:

```bash
dq postgres -c <connection> "<query>" --explain --output json
```

This prepends `EXPLAIN` to the query and returns the plan.

#### PostgreSQL EXPLAIN output parsing

The plan is a tree of nodes. Key things to look for:

| Node type | What it means | Concern level |
|-----------|---------------|---------------|
| `Seq Scan` | Full table scan, reads every row | High on large tables (> 10k rows) |
| `Index Scan` | Uses an index, reads matching rows | Low |
| `Index Only Scan` | Covered by index, no table access | Very low |
| `Bitmap Index Scan` + `Bitmap Heap Scan` | Index narrows down, then fetches | Low-Medium |
| `Nested Loop` | For each row in outer, scan inner | High if inner has no index |
| `Hash Join` / `Merge Join` | Efficient join methods | Low |
| `Sort` | In-memory or disk sort | Medium if large `rows` estimate |

Key fields in each node:
- **`rows`**: Estimated number of rows processed at this step
- **`width`**: Average row width in bytes
- **`cost`**: Relative cost (startup..total). Higher = more expensive
- **`Filter`**: A filter applied after the scan (rows read but discarded)

**Red flags in EXPLAIN output**:
- `Seq Scan` on a table with > 10,000 estimated rows
- `Filter` removing a large percentage of rows (indicates a missing index)
- `Sort` with `Sort Method: external merge` (spilling to disk)
- Very high `cost` values (> 100,000)
- `Nested Loop` with a `Seq Scan` on the inner side

#### MySQL differences

MySQL uses a different EXPLAIN format. Use `EXPLAIN FORMAT=JSON` for structured output:

```bash
dq mysql -c <connection> "EXPLAIN FORMAT=JSON <query>" --output json
```

Key fields in MySQL EXPLAIN:
- `access_type`: `ALL` = full table scan (bad), `ref`/`range`/`const` = indexed (good)
- `rows_examined_per_scan`: rows read per scan
- `filtered`: percentage of rows that pass the WHERE clause (low = inefficient)
- `using_filesort`: `true` = may be slow on large result sets
- `using_temporary_table`: `true` = temp table created, can be slow

#### SQLite differences

SQLite uses `EXPLAIN QUERY PLAN`:

```bash
dq sqlite -c <connection> "EXPLAIN QUERY PLAN <query>" --output json
```

Key things to look for:
- `SCAN TABLE` = full table scan (equivalent of Seq Scan)
- `SEARCH TABLE ... USING INDEX` = indexed access (good)
- `USE TEMP B-TREE` = sorting without an index

### 3. If it's a mutation (INSERT/UPDATE/DELETE): assess blast radius

#### 3a. Count affected rows

Use `SELECT COUNT(*)` with the same WHERE clause as the mutation to get the affected row count with zero side effects:

```bash
dq postgres -c <connection> "SELECT COUNT(*) AS affected_rows FROM <table> WHERE <same_where_clause>" --output json
```

This returns the exact number of rows that would be modified.

#### 3b. Calculate blast radius

Get the total row count of the target table to contextualize the affected rows:

```bash
dq postgres -c <connection> "SELECT COUNT(*) AS total_rows FROM <table>" --output json
```

Compute the blast radius:
- `affected_rows / total_rows * 100` = percentage of table affected
- **< 1%**: Low blast radius
- **1-10%**: Moderate -- double-check the WHERE clause
- **10-50%**: High -- confirm with the user
- **> 50%**: Very high -- almost certainly needs review
- **100%**: Maximum -- equivalent to TRUNCATE for DELETE, or bulk update

Present this clearly: "This DELETE affects **12,000 of 2,000,000 rows** (0.6%) -- low blast radius."

#### 3c. Check for cascading foreign keys

Cascading deletes and updates can silently affect rows in other tables. Check for `ON DELETE CASCADE` and `ON UPDATE CASCADE` constraints:

```bash
dq postgres -c <connection> "SELECT tc.table_name AS source_table, kcu.column_name AS source_column, ccu.table_name AS target_table, ccu.column_name AS target_column, rc.delete_rule, rc.update_rule FROM information_schema.referential_constraints rc JOIN information_schema.table_constraints tc ON rc.constraint_name = tc.constraint_name JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name JOIN information_schema.constraint_column_usage ccu ON rc.unique_constraint_name = ccu.constraint_name WHERE ccu.table_name = '<table>' AND (rc.delete_rule = 'CASCADE' OR rc.update_rule = 'CASCADE')" --output json
```

**If cascading FKs exist**, warn the user: "Deleting from `orders` will CASCADE delete rows from `order_items` and `shipments`."

For each cascading FK, estimate the cascading impact:

```bash
dq postgres -c <connection> "SELECT COUNT(*) AS cascading_rows FROM <child_table> WHERE <fk_column> IN (SELECT <pk_column> FROM <table> WHERE <original_where_clause>)" --output json
```

#### 3d. Check for triggers

Triggers can cause unexpected side effects (logging, auditing, denormalization, or even further mutations):

```bash
dq postgres -c <connection> "SELECT trigger_name, event_manipulation, action_timing, action_statement FROM information_schema.triggers WHERE event_object_table = '<table>' ORDER BY action_timing, event_manipulation" --output json
```

**MySQL equivalent**:

```bash
dq mysql -c <connection> "SELECT TRIGGER_NAME, EVENT_MANIPULATION, ACTION_TIMING, ACTION_STATEMENT FROM information_schema.TRIGGERS WHERE EVENT_OBJECT_TABLE = '<table>'" --output json
```

If triggers exist, summarize what they do and warn about side effects.

### 4. If it's a SELECT: assess performance risk

#### 4a. Check estimated rows from EXPLAIN

From the EXPLAIN output (step 2), find the top-level `rows` estimate. This is how many rows the query is expected to return.

#### 4b. Warn if unbounded

If the query has no `LIMIT` clause and estimated rows > 10,000:

> **Warning**: This query is estimated to return **~250,000 rows** with no LIMIT. This will:
> - Consume significant memory
> - Take a long time to transfer
> - Potentially overflow your context window
>
> Recommend: Add `--limit 100` or add a `LIMIT` clause to the SQL.

#### 4c. Suggest output optimizations

If the result set is wide (many columns) but the user likely only needs a few:

> Recommend: Use `--fields id,name,status` to select only the columns you need.

### 5. Check for missing indexes

Look at the columns used in WHERE, JOIN ON, and ORDER BY clauses. For each column, check if an index exists:

```bash
dq schema indexes -c <connection> --table <table> --output json
```

Cross-reference the index columns with the columns in the query's filter and join conditions. If a column used in a WHERE or JOIN clause is not indexed, flag it:

> **Missing index**: Column `orders.customer_id` is used in a JOIN but has no index. This forces a sequential scan on `orders`. Consider creating an index:
> ```sql
> CREATE INDEX idx_orders_customer_id ON orders(customer_id);
> ```

For compound WHERE clauses (e.g., `WHERE status = 'active' AND created_at > '2026-01-01'`), check if a composite index would help:

> **Potential optimization**: A composite index on `(status, created_at)` would cover both filter conditions.

### 6. Compile risk assessment

Based on all findings, assign an overall risk level:

#### Safe (green)
- SELECT with indexes used, reasonable row estimate, has LIMIT
- Mutation affecting < 1% of rows, no cascading FKs, no triggers, indexes used

#### Caution (yellow)
- Sequential scan on a medium table (10k-100k rows)
- Mutation affecting 1-10% of rows
- Cascading FKs exist but affect a small number of rows
- Triggers exist but are benign (audit logging)
- Missing indexes on filter columns

#### Dangerous (red)
- Sequential scan on a large table (> 100k rows)
- Mutation affecting > 10% of rows
- DELETE/UPDATE with no WHERE clause
- Cascading FKs that will affect many rows in other tables
- Triggers that perform further mutations
- DDL on a production table (ALTER TABLE on large table = table lock)
- Any TRUNCATE or DROP

### 7. Present recommendations

Compile actionable recommendations based on the findings. Prioritize by impact:

1. **Critical** (must fix before running):
   - Add a WHERE clause to unbounded DELETE/UPDATE
   - Fix incorrect join conditions
   - Use `SELECT COUNT(*)` to estimate affected rows before mutating

2. **Recommended** (should fix for safety/performance):
   - Add missing indexes before running the query
   - Add LIMIT to unbounded SELECTs
   - Use `--fields` to reduce output width
   - Add `--timeout 30s` to prevent runaway queries

3. **Optional** (nice to have):
   - Use `--output json` for structured output
   - Consider query rewrites for better performance

Example recommendation summary:

> ## Risk Assessment: CAUTION
>
> **Query**: `DELETE FROM orders WHERE status = 'cancelled' AND created_at < '2025-01-01'`
>
> | Factor | Finding | Risk |
> |--------|---------|------|
> | Blast radius | 12,000 of 2M rows (0.6%) | Low |
> | Cascading FKs | `order_items` ON DELETE CASCADE (~36,000 rows) | Medium |
> | Triggers | `audit_log_trigger` (INSERT into audit_log) | Low |
> | Index usage | Index on `(status, created_at)` will be used | Low |
> | Sequential scan | No | Low |
>
> **Recommendations**:
> 1. Count affected rows first with `SELECT COUNT(*)`
> 2. Be aware that ~36,000 rows in `order_items` will also be deleted via CASCADE
> 3. The `audit_log_trigger` will fire for each deleted row -- expect ~12,000 audit entries
>
> ```bash
> # Step 1: Count affected rows
> dq postgres -c <connection> "SELECT COUNT(*) AS affected_rows FROM orders WHERE status = 'cancelled' AND created_at < '2025-01-01'" --output json
>
> # Step 2: Execute (only after user confirms)
> dq postgres -c <connection> "DELETE FROM orders WHERE status = 'cancelled' AND created_at < '2025-01-01'" --output json
> ```
