---
name: recipe-find-missing-indexes
description: Identify columns that should have indexes but don't. Checks sequential scan frequency, unindexed foreign keys, and query patterns to generate CREATE INDEX recommendations.
---

# Recipe: Find Missing Indexes

Use this recipe to audit a database for missing indexes. Unindexed columns cause sequential scans, slow queries, and unnecessary I/O. This recipe checks three sources: table scan statistics, foreign key columns, and query patterns.

## Step 1: Find Tables with High Sequential Scan Counts

Tables with many sequential scans relative to index scans are likely missing an index on a frequently queried column.

```bash
dq postgres -c <connection> \
  "SELECT schemaname, relname AS table_name,
          seq_scan, idx_scan,
          CASE WHEN (seq_scan + idx_scan) > 0
               THEN ROUND(100.0 * seq_scan / (seq_scan + idx_scan), 1)
               ELSE 0 END AS seq_scan_pct,
          seq_tup_read, n_live_tup AS estimated_rows
   FROM pg_stat_user_tables
   WHERE seq_scan > 0 AND n_live_tup > 1000
   ORDER BY seq_scan DESC
   LIMIT 20" \
  --output json
```

Focus on tables where:
- `seq_scan_pct` is above 50% (most reads are sequential)
- `estimated_rows` is large (sequential scans on big tables are expensive)
- `seq_tup_read` is significantly higher than `n_live_tup` (indicates repeated full scans)

## Step 2: Find Foreign Key Columns Without Indexes

Foreign key columns are frequently used in JOINs and WHERE clauses. Missing indexes on FK columns is one of the most common performance oversights.

```bash
dq postgres -c <connection> \
  "SELECT
     tc.table_schema, tc.table_name,
     kcu.column_name AS fk_column,
     ccu.table_name AS referenced_table,
     ccu.column_name AS referenced_column
   FROM information_schema.table_constraints tc
   JOIN information_schema.key_column_usage kcu
     ON tc.constraint_name = kcu.constraint_name
     AND tc.table_schema = kcu.table_schema
   JOIN information_schema.constraint_column_usage ccu
     ON tc.constraint_name = ccu.constraint_name
   WHERE tc.constraint_type = 'FOREIGN KEY'
     AND tc.table_schema NOT IN ('pg_catalog', 'information_schema')
   ORDER BY tc.table_name" \
  --output json
```

Now check which of these FK columns already have indexes:

```bash
dq postgres -c <connection> \
  "SELECT
     t.relname AS table_name,
     a.attname AS column_name,
     EXISTS (
       SELECT 1 FROM pg_index i
       JOIN pg_attribute ia ON ia.attrelid = i.indrelid AND ia.attnum = ANY(i.indkey)
       WHERE i.indrelid = t.oid AND ia.attname = a.attname
     ) AS has_index
   FROM pg_class t
   JOIN pg_attribute a ON a.attrelid = t.oid
   JOIN pg_namespace n ON t.relnamespace = n.oid
   WHERE a.attname IN (
     SELECT kcu.column_name
     FROM information_schema.table_constraints tc
     JOIN information_schema.key_column_usage kcu
       ON tc.constraint_name = kcu.constraint_name
     WHERE tc.constraint_type = 'FOREIGN KEY'
   )
   AND n.nspname NOT IN ('pg_catalog', 'information_schema')
   AND a.attnum > 0
   ORDER BY t.relname, a.attname" \
  --output json
```

Any row with `has_index = false` is a missing FK index.

## Step 3: Check Query Patterns via pg_stat_statements (If Available)

The `pg_stat_statements` extension tracks query execution statistics. If installed, it reveals which queries run most often and which are slowest — both good signals for where indexes are needed.

```bash
# Check if pg_stat_statements is available
dq postgres -c <connection> \
  "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements') AS available" \
  --output json
```

If available:

```bash
dq postgres -c <connection> \
  "SELECT query, calls, mean_exec_time, total_exec_time,
          rows AS avg_rows_returned
   FROM pg_stat_statements
   WHERE mean_exec_time > 100
   ORDER BY total_exec_time DESC
   LIMIT 20" \
  --output json
```

Look at the WHERE and JOIN clauses in the top queries. Columns appearing in these clauses on tables identified in Step 1 are strong index candidates.

## Step 4: Cross-Reference Existing Indexes

Before generating recommendations, get the full list of existing indexes to avoid suggesting duplicates or redundant indexes (an index on `(a, b)` already covers queries filtering on `a` alone).

```bash
dq schema indexes -c <connection> --output json
```

For detailed index information including column order:

```bash
dq postgres -c <connection> \
  "SELECT
     t.relname AS table_name,
     i.relname AS index_name,
     array_to_string(ARRAY(
       SELECT a.attname
       FROM unnest(ix.indkey) WITH ORDINALITY AS k(attnum, ord)
       JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
       ORDER BY k.ord
     ), ', ') AS columns,
     ix.indisunique AS is_unique,
     ix.indisprimary AS is_primary
   FROM pg_index ix
   JOIN pg_class t ON t.oid = ix.indrelid
   JOIN pg_class i ON i.oid = ix.indexrelid
   JOIN pg_namespace n ON t.relnamespace = n.oid
   WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
   ORDER BY t.relname, i.relname" \
  --output json
```

A new index is redundant if:
- An existing index has the same leading columns in the same order
- A unique or primary key constraint already covers the column

## Step 5: Generate CREATE INDEX Suggestions

For each candidate column identified in Steps 1-3 that is not covered by an existing index from Step 4, generate a CREATE INDEX statement:

```bash
# Use CONCURRENTLY to avoid locking the table during index creation (PostgreSQL)
# Naming convention: idx_<table>_<column>
```

Example suggestions to present to the user:

```sql
-- FK column orders.customer_id has no index (used in JOINs, table has 2.3M rows)
CREATE INDEX CONCURRENTLY idx_orders_customer_id ON orders (customer_id);

-- High seq_scan count on events table, common WHERE clause on created_at
CREATE INDEX CONCURRENTLY idx_events_created_at ON events (created_at);

-- Composite index for frequent query pattern: WHERE status = X AND created_at > Y
CREATE INDEX CONCURRENTLY idx_orders_status_created_at ON orders (status, created_at);
```

Always use `CONCURRENTLY` on PostgreSQL to avoid blocking writes during creation.

## Step 6: Estimate Index Size Impact

Before creating indexes, estimate how much disk space they will consume:

```bash
dq postgres -c <connection> \
  "SELECT
     relname AS table_name,
     pg_size_pretty(pg_relation_size(oid)) AS table_size,
     pg_size_pretty(pg_indexes_size(oid)) AS current_indexes_size,
     n_live_tup AS row_count
   FROM pg_class
   WHERE relname IN ('<table1>', '<table2>')
   AND relkind = 'r'" \
  --output json
```

Rule of thumb: a B-tree index on a single integer column is roughly 20-30% of the table size. Text columns will be larger. Composite indexes are larger still.

Present the trade-off: "Adding this index will use approximately X MB of disk but will eliminate sequential scans on a table with Y million rows."

## Step 7: Annotate Tables with Recommendations

Persist the findings so future sessions remember the analysis:

```bash
dq annotate set -c <connection> --table orders \
  --note "Missing index on customer_id (FK). seq_scan count: 45,230. Recommend: CREATE INDEX CONCURRENTLY idx_orders_customer_id ON orders (customer_id)"

dq annotate set -c <connection> --table events \
  --note "High seq_scan ratio (92%). created_at column used in frequent WHERE clauses. Recommend: CREATE INDEX CONCURRENTLY idx_events_created_at ON events (created_at)"
```

## MySQL Equivalents

- **Table scan stats:** Use `sys.schema_tables_with_full_table_scans` or `SHOW TABLE STATUS`.
- **Index listing:** `SHOW INDEX FROM <table>` or `dq schema indexes -c <connection> --table <table>`.
- **Query patterns:** Enable the `performance_schema` and query `events_statements_summary_by_digest`.
- **Index creation:** Use `ALTER TABLE <table> ADD INDEX idx_name (column)` (no CONCURRENTLY equivalent; consider `pt-online-schema-change` for large tables).
- Use `dq mysql -c <connection> "<sql>"` instead of `dq postgres`.

## SQLite Equivalents

- **Missing indexes:** `EXPLAIN QUERY PLAN SELECT ...` shows `SCAN TABLE` for unindexed queries.
- **Index listing:** `PRAGMA index_list('<table>')` and `PRAGMA index_info('<index>')`.
- **Index creation:** `CREATE INDEX idx_name ON <table> (column)` (locks the database briefly).
- Use `dq sqlite -c <connection> "<sql>"` instead of `dq postgres`.
