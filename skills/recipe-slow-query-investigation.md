---
name: recipe-slow-query-investigation
description: Find and diagnose slow queries on a database. Identify active long-running queries, analyze their plans, check table statistics, find unused indexes, and suggest optimizations. Use when the user reports slow performance or wants to optimize their database.
---

# Recipe: Slow Query Investigation

Use this recipe when the user reports slow database performance, wants to find bottlenecks, or asks "why is my query slow?" This is a systematic investigation from symptoms to root causes to fixes.

## Prerequisites

- A working connection with sufficient privileges to read `pg_stat_*` views (verify with `dq connection test -c <connection> --output json`)
- Schema context (run `dq discover -c <connection> --output json` if you haven't already)

## Steps

### 1. Find active slow queries

Query `pg_stat_activity` to find currently running queries sorted by duration:

```bash
dq postgres -c <connection> "SELECT pid, usename, state, wait_event_type, wait_event, NOW() - query_start AS duration, LEFT(query, 200) AS query_preview FROM pg_stat_activity WHERE state != 'idle' AND pid != pg_backend_pid() ORDER BY query_start ASC" --output json
```

**What to look for**:
- Queries running for more than a few seconds in an OLTP workload
- `wait_event_type` of `Lock` indicates the query is blocked by another transaction
- Multiple queries with similar `query_preview` may indicate a hot path in the application

**Decision point**: If the user is investigating a specific query they already have (not a live performance issue), skip to step 2 with that query.

To see queries that are waiting on locks:

```bash
dq postgres -c <connection> "SELECT blocked.pid AS blocked_pid, blocked.query AS blocked_query, blocking.pid AS blocking_pid, blocking.query AS blocking_query, NOW() - blocked.query_start AS blocked_duration FROM pg_stat_activity blocked JOIN pg_locks bl ON blocked.pid = bl.pid JOIN pg_locks kl ON bl.locktype = kl.locktype AND bl.database IS NOT DISTINCT FROM kl.database AND bl.relation IS NOT DISTINCT FROM kl.relation AND bl.page IS NOT DISTINCT FROM kl.page AND bl.tuple IS NOT DISTINCT FROM kl.tuple AND bl.virtualxid IS NOT DISTINCT FROM kl.virtualxid AND bl.transactionid IS NOT DISTINCT FROM kl.transactionid AND bl.classid IS NOT DISTINCT FROM kl.classid AND bl.objid IS NOT DISTINCT FROM kl.objid AND bl.objsubid IS NOT DISTINCT FROM kl.objsubid AND bl.pid != kl.pid JOIN pg_stat_activity blocking ON kl.pid = blocking.pid WHERE NOT bl.granted ORDER BY blocked_duration DESC" --output json
```

**MySQL equivalent** -- use `SHOW PROCESSLIST`:

```bash
dq mysql -c <connection> "SELECT ID, USER, HOST, DB, COMMAND, TIME, STATE, LEFT(INFO, 200) AS query_preview FROM information_schema.PROCESSLIST WHERE COMMAND != 'Sleep' ORDER BY TIME DESC" --output json
```

### 2. Analyze query plans for slow queries

For each slow query identified, run EXPLAIN ANALYZE to get the actual execution plan with real timing. **Important**: Run this on a safe copy of the query, not the original running query. If the query is a mutation, use EXPLAIN only (without ANALYZE) or run EXPLAIN ANALYZE inside a transaction that you roll back.

For SELECT queries:

```bash
dq postgres -c <connection> "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) <slow_query>" --output json --timeout 120s
```

For mutation queries (safe analysis without committing):

```bash
dq postgres -c <connection> "EXPLAIN <mutation_query>" --explain --output json
```

#### Reading the EXPLAIN ANALYZE output

The output is a tree of plan nodes. For each node, compare:

- **`Actual Rows`** vs **`Plan Rows`** (estimated): A large discrepancy means the planner has bad statistics. Consider running `ANALYZE <table>`.
- **`Actual Time`**: Time in milliseconds. The node with the highest exclusive time is your bottleneck.
- **`Buffers: shared hit`** vs **`shared read`**: `shared read` means disk I/O (slow). `shared hit` means cache hit (fast). High `shared read` indicates the working set doesn't fit in `shared_buffers`.

### 3. Identify plan problems

Look for these common performance killers in the plan:

#### Sequential scans on large tables

```
Seq Scan on orders  (cost=0.00..125432.00 rows=2000000 width=128)
  Filter: (status = 'pending')
  Rows Removed by Filter: 1990000
```

This reads 2M rows and discards 1.99M -- a clear sign that an index on `status` is needed.

#### Nested loops without index on inner side

```
Nested Loop  (cost=0.00..1250000.00 rows=50000 width=64)
  -> Seq Scan on customers  (cost=0.00..500.00 rows=500 width=32)
  -> Seq Scan on orders  (cost=0.00..2500.00 rows=100 width=32)
        Filter: (orders.customer_id = customers.id)
```

For each of 500 customers, it scans all of orders. An index on `orders.customer_id` turns this into 500 index lookups.

#### Sort spilling to disk

```
Sort  (cost=250000.00..255000.00 rows=2000000 width=64)
  Sort Key: created_at
  Sort Method: external merge  Disk: 128000kB
```

`external merge` means the sort exceeded `work_mem` and spilled to disk. Either increase `work_mem` or add an index on `created_at` to avoid the sort.

#### Hash Join with large build side

If the build side of a Hash Join is very large, it may spill to disk. Check `Batches` in the output -- more than 1 means disk spilling.

### 4. Check table statistics

For each table involved in slow queries, check `pg_stat_user_tables` for access patterns:

```bash
dq postgres -c <connection> "SELECT schemaname, relname, seq_scan, seq_tup_read, idx_scan, idx_tup_fetch, n_live_tup, n_dead_tup, last_vacuum, last_autovacuum, last_analyze, last_autoanalyze FROM pg_stat_user_tables WHERE relname IN ('<table1>', '<table2>') ORDER BY seq_scan DESC" --output json
```

**What to look for**:

| Metric | Concern | Action |
|--------|---------|--------|
| `seq_scan` >> `idx_scan` | Table is mostly scanned sequentially | Add indexes on commonly filtered columns |
| `n_dead_tup` high relative to `n_live_tup` | Table needs vacuuming | Check vacuum settings, consider manual VACUUM |
| `last_analyze` is NULL or very old | Planner statistics are stale | Run `ANALYZE <table>` |
| `seq_tup_read` very high | Many rows read via sequential scans | Indexes needed |

**MySQL equivalent**:

```bash
dq mysql -c <connection> "SHOW TABLE STATUS LIKE '<table>'" --output json
```

### 5. Check index usage

Find unused or underused indexes (they cost write performance but provide no read benefit):

```bash
dq postgres -c <connection> "SELECT schemaname, relname, indexrelname, idx_scan, idx_tup_read, idx_tup_fetch, pg_size_pretty(pg_relation_size(indexrelid)) AS index_size FROM pg_stat_user_indexes WHERE relname IN ('<table1>', '<table2>') ORDER BY idx_scan ASC" --output json
```

**What to look for**:
- `idx_scan = 0`: Index has never been used since stats were reset. Consider dropping it unless it's for a unique constraint or is needed for rare but critical queries.
- Large `index_size` with low `idx_scan`: Wasting disk space and slowing down writes.

Check for duplicate or overlapping indexes:

```bash
dq postgres -c <connection> "SELECT a.indexrelid::regclass AS index_a, b.indexrelid::regclass AS index_b, a.indrelid::regclass AS table_name FROM pg_index a JOIN pg_index b ON a.indrelid = b.indrelid AND a.indexrelid != b.indexrelid AND a.indkey::text = LEFT(b.indkey::text, LENGTH(a.indkey::text)) WHERE a.indrelid IN ('<table>'::regclass)" --output json
```

This finds indexes where one is a prefix of another (e.g., index on `(a)` is redundant if an index on `(a, b)` exists).

### 6. Suggest optimizations

Based on findings from steps 2-5, compile specific recommendations:

#### Index creation

When a sequential scan is filtering heavily, suggest an index:

```bash
# Recommend to the user:
# CREATE INDEX idx_<table>_<column> ON <table>(<column>);
```

For queries with multiple WHERE conditions, suggest a composite index with the most selective column first:

```sql
-- For: WHERE status = 'active' AND created_at > '2026-01-01'
CREATE INDEX idx_orders_status_created ON orders(status, created_at);
```

For queries that only need a few columns, suggest a covering index:

```sql
-- For: SELECT id, email FROM users WHERE status = 'active'
CREATE INDEX idx_users_status_covering ON users(status) INCLUDE (id, email);
```

#### Query rewrites

- Replace `SELECT *` with specific columns
- Add `LIMIT` to unbounded queries
- Replace `NOT IN (subquery)` with `NOT EXISTS` (better plan)
- Replace `DISTINCT` with `GROUP BY` when appropriate
- Move expensive function calls out of WHERE clauses

#### Configuration suggestions

- If sorts spill to disk: suggest increasing `work_mem`
- If shared reads are high: suggest increasing `shared_buffers`
- If stats are stale: suggest running `ANALYZE` on affected tables

### 7. Annotate findings

Persist your findings so future investigations have context:

```bash
dq annotate set -c <connection> --table <table> --note "Slow query investigation <date>: seq_scan ratio high (seq=45000, idx=200). Missing index on <column>. Recommended: CREATE INDEX idx_<table>_<column> ON <table>(<column>)."
```

```bash
dq annotate set -c <connection> --table <table> --column <column> --note "Frequently used in WHERE clause but not indexed. Causes seq scans."
```

If you recommend an index and the user creates it, update the annotation:

```bash
dq annotate set -c <connection> --table <table> --note "Index idx_<table>_<column> created on <date>. Monitor idx_scan to confirm it's being used."
```
