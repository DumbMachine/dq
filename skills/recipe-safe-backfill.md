---
name: recipe-safe-backfill
description: Batch update pattern for large data changes. Processes rows in controlled batches to avoid long locks, timeouts, and replication lag. Includes progress tracking and verification.
---

# Recipe: Safe Backfill

Use this recipe when you need to UPDATE or INSERT a large number of rows (thousands or more). Running a single large UPDATE can lock the table, stall replication, and time out. Instead, process in batches.

## Step 1: Estimate Total Rows

Get an accurate count of how many rows need the backfill. This sets expectations and enables progress tracking.

```bash
dq postgres -c <connection> \
  "SELECT COUNT(*) AS total_rows FROM <table> WHERE <backfill_condition>" \
  --output json
```

Also check the total table size for context:

```bash
dq postgres -c <connection> \
  "SELECT COUNT(*) AS table_total,
          pg_size_pretty(pg_total_relation_size('<table>')) AS table_size
   FROM <table>" \
  --output json
```

If `total_rows` exceeds 100,000, batching is strongly recommended. If it exceeds 1,000,000, consider running during a maintenance window.

## Step 2: Preview the Change with --dry-run

Test the UPDATE on a small sample to confirm it does what you expect:

```bash
dq postgres -c <connection> \
  "UPDATE <table> SET <column> = <new_value>
   WHERE <backfill_condition> AND id IN (
     SELECT id FROM <table> WHERE <backfill_condition> ORDER BY id LIMIT 5
   )" --dry-run --output json
```

## Step 3: Show Before/After for a Sample

Select a few rows before and preview what they will look like after:

```bash
# Before
dq postgres -c <connection> \
  "SELECT id, <column>, <other_relevant_columns> FROM <table>
   WHERE <backfill_condition> ORDER BY id LIMIT 5" \
  --output json
```

```bash
# Simulated after (use a CASE or the new value expression)
dq postgres -c <connection> \
  "SELECT id, <new_value> AS <column>_new, <column> AS <column>_old
   FROM <table>
   WHERE <backfill_condition> ORDER BY id LIMIT 5" \
  --output json
```

Present both results side-by-side to the user. Confirm the transformation is correct before proceeding.

## Step 4: Execute in Batches

Use the `id > last_processed_id` pattern to page through rows without OFFSET (which gets slower on each iteration).

**Batch size guidance:**
- Start with 1,000 rows per batch
- If each batch takes more than 5 seconds, reduce the batch size
- If replication lag appears, pause and wait for it to catch up

Run each batch:

```bash
# Batch 1: start from 0 (or the minimum id)
dq postgres -c <connection> \
  "UPDATE <table> SET <column> = <new_value>
   WHERE <backfill_condition> AND id > 0
   ORDER BY id LIMIT 1000" \
  --output json
```

```bash
# Find the last id processed in this batch
dq postgres -c <connection> \
  "SELECT id FROM <table>
   WHERE <column> = <new_value>
   ORDER BY id DESC LIMIT 1" \
  --output json
```

```bash
# Batch 2: continue from last_id
dq postgres -c <connection> \
  "UPDATE <table> SET <column> = <new_value>
   WHERE <backfill_condition> AND id > <last_id>
   ORDER BY id LIMIT 1000" \
  --output json
```

Repeat until `affected_rows` returns 0.

**PostgreSQL batch pattern using a CTE (preferred — avoids ORDER BY in UPDATE):**

```bash
dq postgres -c <connection> \
  "WITH batch AS (
     SELECT id FROM <table>
     WHERE <backfill_condition> AND id > <last_id>
     ORDER BY id LIMIT 1000
   )
   UPDATE <table> SET <column> = <new_value>
   FROM batch WHERE <table>.id = batch.id
   RETURNING <table>.id" \
  --output json
```

The `RETURNING` clause gives you the processed IDs directly. Use the maximum returned ID as the next `last_id`.

## Step 5: Report Progress After Each Batch

After each batch, calculate and report progress:

```bash
dq postgres -c <connection> \
  "SELECT COUNT(*) AS completed FROM <table> WHERE <completion_condition>" \
  --output json
```

Report format: "Batch N complete: X of Y rows updated (Z%)"

If any batch returns 0 `affected_rows` before reaching the total, the backfill is done.

## Step 6: Verify Final State

After all batches complete, confirm the backfill is fully applied:

```bash
# Count remaining rows that still need the backfill
dq postgres -c <connection> \
  "SELECT COUNT(*) AS remaining FROM <table> WHERE <backfill_condition>" \
  --output json
```

`remaining` should be 0. If not, investigate the missed rows.

```bash
# Spot-check a sample of updated rows
dq postgres -c <connection> \
  "SELECT id, <column>, <other_relevant_columns> FROM <table>
   WHERE <completion_condition>
   ORDER BY id DESC LIMIT 10" \
  --output json
```

Annotate the table with the backfill status:

```bash
dq annotate set -c <connection> --table <table> \
  --note "Backfill of <column> completed on <date>. All rows now have <new_value>."
```

## Warnings

- **Table locks:** Large UPDATEs acquire row-level locks in PostgreSQL. If another process needs the same rows, it will block. Keep batch sizes small enough that each batch completes in under 5 seconds.
- **WAL / binlog growth:** Each batch generates write-ahead log entries. On high-traffic databases, large backfills can cause WAL accumulation. Monitor disk usage.
- **Replication lag:** If running on a primary with replicas, check replication lag between batches. Pause if lag exceeds your acceptable threshold.
- **Deadlocks:** If the backfill conflicts with application writes, use `ORDER BY id` to ensure consistent lock ordering and reduce deadlock risk.
- **Triggers:** If the table has UPDATE triggers, each batch will fire them. Factor this into your timing and batch size estimates.

## MySQL Differences

- MySQL supports `UPDATE ... ORDER BY id LIMIT 1000` directly (no CTE needed).
- Check replication lag: `SHOW SLAVE STATUS` or `SELECT * FROM sys.replication_lag`.
- Use `dq mysql -c <connection> "<sql>"` instead of `dq postgres`.

## SQLite Differences

- SQLite locks the entire database on writes (no row-level locking). Batching prevents long lock holds.
- No replication concerns, but long write locks block all readers in WAL mode.
- Use `dq sqlite -c <connection> "<sql>"` instead of `dq postgres`.
