---
name: recipe-table-health-check
description: Comprehensive health check for a specific database table. Evaluate size, dead tuples, vacuum status, index efficiency, bloat, and missing FK indexes. Use when the user wants to assess the health of a table or investigate degraded performance on a specific table.
---

# Recipe: Table Health Check

Use this recipe when the user asks about the health of a specific table, wants to know if maintenance is needed, or when you notice performance degradation on a particular table. This is a systematic check that covers storage, vacuuming, indexes, and bloat.

## Prerequisites

- A working connection (verify with `dq connection test -c <connection> --output json`)
- The table name to check
- Schema context (run `dq discover -c <connection> --output json` if you haven't already)

## Steps

### 1. Get table size and row count

Start with the basics -- how big is this table?

```bash
dq postgres -c <connection> "SELECT c.relname AS table_name, c.reltuples::bigint AS estimated_row_count, pg_size_pretty(pg_total_relation_size(c.oid)) AS total_size, pg_size_pretty(pg_relation_size(c.oid)) AS table_size, pg_size_pretty(pg_total_relation_size(c.oid) - pg_relation_size(c.oid)) AS index_and_toast_size, pg_total_relation_size(c.oid) AS total_size_bytes FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid WHERE c.relname = '<table>' AND n.nspname = 'public' AND c.relkind = 'r'" --output json
```

**MySQL equivalent**:

```bash
dq mysql -c <connection> "SELECT TABLE_NAME, TABLE_ROWS, ROUND(DATA_LENGTH / 1024 / 1024, 2) AS data_size_mb, ROUND(INDEX_LENGTH / 1024 / 1024, 2) AS index_size_mb, ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2) AS total_size_mb, DATA_FREE AS free_space_bytes FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '<table>'" --output json
```

Note the ratio of index size to table size. If indexes are larger than the table data, that may indicate too many indexes or bloated indexes.

### 2. Check dead tuple ratio

Dead tuples are rows that have been deleted or updated but not yet cleaned up by VACUUM. A high dead tuple ratio degrades query performance because the database must skip over dead rows during scans.

```bash
dq postgres -c <connection> "SELECT relname, n_live_tup, n_dead_tup, CASE WHEN n_live_tup > 0 THEN ROUND(n_dead_tup::numeric / n_live_tup * 100, 2) ELSE 0 END AS dead_tuple_pct, n_mod_since_analyze FROM pg_stat_user_tables WHERE relname = '<table>'" --output json
```

**Interpreting dead_tuple_pct**:

| Dead tuple % | Status | Action |
|-------------|--------|--------|
| < 5% | Healthy | No action needed |
| 5-10% | Moderate | Autovacuum may be lagging, check settings |
| 10-20% | Concerning | Consider manual VACUUM |
| > 20% | Critical | Run VACUUM immediately, investigate why autovacuum isn't keeping up |

**MySQL equivalent** -- MySQL/InnoDB handles this differently via its purge system. Check fragmentation instead:

```bash
dq mysql -c <connection> "SELECT TABLE_NAME, DATA_FREE, ROUND(DATA_FREE / (DATA_LENGTH + INDEX_LENGTH) * 100, 2) AS fragmentation_pct FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '<table>'" --output json
```

### 3. Check last vacuum and autovacuum times

```bash
dq postgres -c <connection> "SELECT relname, last_vacuum, last_autovacuum, last_analyze, last_autoanalyze, vacuum_count, autovacuum_count, analyze_count, autoanalyze_count FROM pg_stat_user_tables WHERE relname = '<table>'" --output json
```

**What to look for**:
- `last_autovacuum` is NULL: Autovacuum has never run on this table. Either the table was just created, or autovacuum is disabled/misconfigured.
- `last_autovacuum` is very old (> 1 day for an active table): Autovacuum may be falling behind.
- `autovacuum_count` is 0 but `vacuum_count` > 0: Someone is running manual vacuums but autovacuum isn't triggering.
- `last_analyze` / `last_autoanalyze` is old: Planner statistics may be stale, leading to bad query plans.

If autovacuum appears to be lagging, check the table-specific autovacuum settings:

```bash
dq postgres -c <connection> "SELECT relname, reloptions FROM pg_class WHERE relname = '<table>'" --output json
```

`reloptions` may contain overrides like `autovacuum_vacuum_threshold`, `autovacuum_vacuum_scale_factor`, etc.

### 4. Check index usage: sequential scan vs index scan ratio

```bash
dq postgres -c <connection> "SELECT relname, seq_scan, seq_tup_read, idx_scan, idx_tup_fetch, CASE WHEN (seq_scan + COALESCE(idx_scan, 0)) > 0 THEN ROUND(seq_scan::numeric / (seq_scan + COALESCE(idx_scan, 0)) * 100, 2) ELSE 0 END AS seq_scan_pct FROM pg_stat_user_tables WHERE relname = '<table>'" --output json
```

**Interpreting seq_scan_pct**:

| Seq scan % | Status | Action |
|-----------|--------|--------|
| < 10% | Excellent | Indexes are well-utilized |
| 10-30% | Acceptable | Some queries may benefit from additional indexes |
| 30-60% | Concerning | Investigate top queries hitting this table |
| > 60% | Poor | Critical indexing gaps, investigate immediately |

**Decision point**: If `seq_scan_pct` is high, this table is a candidate for the `recipe-slow-query-investigation` workflow to find which queries are causing the sequential scans.

### 5. List indexes and their sizes, flag problems

Get all indexes on the table with their sizes and usage stats:

```bash
dq postgres -c <connection> "SELECT i.indexrelname AS index_name, i.idx_scan AS times_used, i.idx_tup_read, i.idx_tup_fetch, pg_size_pretty(pg_relation_size(i.indexrelid)) AS index_size, pg_relation_size(i.indexrelid) AS index_size_bytes, ix.indisunique AS is_unique, ix.indisprimary AS is_primary, pg_get_indexdef(i.indexrelid) AS index_definition FROM pg_stat_user_indexes i JOIN pg_index ix ON i.indexrelid = ix.indexrelid WHERE i.relname = '<table>' ORDER BY pg_relation_size(i.indexrelid) DESC" --output json
```

**Flag these problems**:

#### Unused indexes (never scanned)

Indexes with `idx_scan = 0` since the last statistics reset. These cost write performance (every INSERT/UPDATE/DELETE must update all indexes) but provide no read benefit.

**Exception**: Do not flag primary key indexes, unique constraint indexes, or indexes on foreign key columns as "unused" -- they serve constraint enforcement even if they aren't used for scans.

```bash
# Check when stats were last reset
dq postgres -c <connection> "SELECT stats_reset FROM pg_stat_database WHERE datname = current_database()" --output json
```

If stats were recently reset, unused indexes may just not have been used yet.

#### Duplicate indexes

Two indexes on the same column(s) where one is a subset of the other:

```bash
dq postgres -c <connection> "WITH index_cols AS (SELECT indexrelid, indrelid, array_to_string(ARRAY(SELECT attname FROM pg_attribute WHERE attrelid = indexrelid ORDER BY attnum), ', ') AS columns FROM pg_index WHERE indrelid = '<table>'::regclass) SELECT a.indexrelid::regclass AS redundant_index, b.indexrelid::regclass AS covering_index, a.columns AS redundant_columns, b.columns AS covering_columns FROM index_cols a JOIN index_cols b ON a.indrelid = b.indrelid AND a.indexrelid != b.indexrelid AND a.columns = LEFT(b.columns, LENGTH(a.columns)) AND LENGTH(a.columns) < LENGTH(b.columns)" --output json
```

#### Oversized indexes

If an index is larger than the table itself, something may be wrong (bloated index, or the index includes many columns that duplicate the table data).

**MySQL equivalent**:

```bash
dq mysql -c <connection> "SHOW INDEX FROM <table>" --output json
```

### 6. Estimate table bloat

Table bloat occurs when dead space from deleted/updated rows accumulates faster than VACUUM can clean it up. This query estimates the bloat ratio:

```bash
dq postgres -c <connection> "SELECT current_database(), schemaname, tablename, pg_size_pretty(pg_relation_size(schemaname || '.' || tablename)) AS table_size, CASE WHEN avg_width > 0 AND reltuples > 0 THEN pg_size_pretty( (pg_relation_size(schemaname || '.' || tablename) - (reltuples * (avg_width + 24))::bigint)::bigint ) ELSE 'N/A' END AS estimated_bloat, CASE WHEN pg_relation_size(schemaname || '.' || tablename) > 0 AND avg_width > 0 AND reltuples > 0 THEN ROUND( ((pg_relation_size(schemaname || '.' || tablename) - (reltuples * (avg_width + 24))::bigint)::numeric / pg_relation_size(schemaname || '.' || tablename)) * 100, 2 ) ELSE 0 END AS bloat_pct FROM ( SELECT s.schemaname, s.relname AS tablename, c.reltuples, (SELECT AVG(avg_width) FROM pg_stats WHERE schemaname = s.schemaname AND tablename = s.relname) AS avg_width FROM pg_stat_user_tables s JOIN pg_class c ON s.relid = c.oid WHERE s.relname = '<table>' ) sub" --output json
```

**Note**: This is a rough estimate. For precise bloat measurement, consider using the `pgstattuple` extension if available:

```bash
dq postgres -c <connection> "SELECT * FROM pgstattuple('<table>')" --output json
```

**Interpreting bloat_pct**:

| Bloat % | Status | Action |
|---------|--------|--------|
| < 20% | Normal | Expected for active tables |
| 20-40% | Moderate | Monitor, consider VACUUM FULL during maintenance window |
| 40-60% | High | VACUUM FULL recommended |
| > 60% | Critical | VACUUM FULL urgently needed, or consider pg_repack |

**Warning**: `VACUUM FULL` takes an exclusive lock on the table and rewrites the entire table. It should only be run during maintenance windows. For online compaction, recommend `pg_repack` instead.

### 7. Check for missing indexes on foreign key columns

Foreign key columns without indexes cause slow joins and slow cascading deletes. This is one of the most common performance problems:

```bash
dq postgres -c <connection> "SELECT tc.constraint_name, kcu.column_name AS fk_column, ccu.table_name AS referenced_table, ccu.column_name AS referenced_column, EXISTS ( SELECT 1 FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) WHERE i.indrelid = tc.conrelid AND a.attname = kcu.column_name ) AS has_index FROM pg_constraint tc JOIN information_schema.key_column_usage kcu ON tc.conname = kcu.constraint_name AND tc.connamespace = (SELECT oid FROM pg_namespace WHERE nspname = kcu.constraint_schema) JOIN information_schema.constraint_column_usage ccu ON tc.conname = ccu.constraint_name WHERE tc.contype = 'f' AND tc.conrelid = '<table>'::regclass" --output json
```

For any FK column where `has_index` is false:

> **Missing FK index**: Column `<table>.<fk_column>` references `<referenced_table>.<referenced_column>` but has no index. This causes:
> - Slow JOINs between `<table>` and `<referenced_table>`
> - Slow cascading DELETEs when rows are deleted from `<referenced_table>`
>
> Recommended:
> ```sql
> CREATE INDEX idx_<table>_<fk_column> ON <table>(<fk_column>);
> ```

**MySQL note**: InnoDB automatically creates indexes on FK columns, so this check is primarily relevant for PostgreSQL.

### 8. Compile health report and annotate

Compile all findings into a structured health report:

> ## Table Health Report: `<table>`
>
> | Metric | Value | Status |
> |--------|-------|--------|
> | Row count | 2,500,000 | -- |
> | Total size | 1.2 GB | -- |
> | Dead tuple ratio | 3.2% | Healthy |
> | Last autovacuum | 2 hours ago | Healthy |
> | Seq scan ratio | 45% | Concerning |
> | Unused indexes | 2 (128 MB total) | Action needed |
> | Table bloat | ~18% | Normal |
> | Missing FK indexes | 1 (`customer_id`) | Action needed |
>
> ### Actions Recommended
> 1. **Create index on `customer_id`**: Missing FK index causing slow joins
> 2. **Drop unused indexes**: `idx_old_status`, `idx_temp_debug` (saving 128 MB and write overhead)
> 3. **Investigate seq scans**: 45% sequential scan ratio suggests missing indexes on query filter columns

Annotate the health check results:

```bash
dq annotate set -c <connection> --table <table> --note "Health check <date>: <row_count> rows, <total_size>. Dead tuples: <dead_pct>%. Seq scan ratio: <seq_pct>%. Bloat: <bloat_pct>%. Issues: <summary of issues found>."
```

For specific column issues:

```bash
dq annotate set -c <connection> --table <table> --column <fk_column> --note "FK to <referenced_table> but missing index. Recommend: CREATE INDEX idx_<table>_<fk_column> ON <table>(<fk_column>)."
```
