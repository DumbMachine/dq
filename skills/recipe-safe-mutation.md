---
name: recipe-safe-mutation
description: The dry-run, confirm, execute pattern for any INSERT, UPDATE, or DELETE. Prevents accidental data loss by previewing impact before committing changes.
---

# Recipe: Safe Mutation

Every mutation (INSERT, UPDATE, DELETE) must follow this pattern: explain, dry-run, preview, confirm, execute, verify. Never skip steps.

## Step 1: Examine the Query Plan

Before touching data, understand HOW the database will execute the query. This catches missing indexes, full table scans, and unexpected join behavior.

```bash
dq postgres -c <connection> "<mutation_sql>" --explain --output json
```

Look for sequential scans on large tables — they signal the WHERE clause may not be selective enough.

## Step 2: Dry-Run to See Affected Row Count

The `--dry-run` flag wraps the mutation in a transaction and rolls it back, reporting the number of rows that would be affected without making any changes.

```bash
dq postgres -c <connection> "DELETE FROM sessions WHERE expires_at < NOW()" --dry-run --output json
```

The response includes `affected_rows`. If this number is unexpectedly high or zero, stop and investigate.

## Step 3: If UPDATE — Preview Rows Before the Change

For UPDATEs, show a sample of the rows that will be modified so the user can verify the WHERE clause targets the right data.

```bash
# Use the same WHERE clause as the UPDATE, but as a SELECT
dq postgres -c <connection> \
  "SELECT id, status, updated_at FROM users WHERE status = 'trial' AND created_at < '2025-01-01'" \
  --limit 10 --output json
```

Also get the total count:

```bash
dq postgres -c <connection> \
  "SELECT COUNT(*) AS total FROM users WHERE status = 'trial' AND created_at < '2025-01-01'" \
  --output json
```

## Step 4: If DELETE — Check for CASCADE Foreign Keys

Deletes can silently cascade to child tables. Check what foreign keys reference this table before proceeding.

```bash
# Get constraints on the target table
dq schema constraints -c <connection> --table <table> --output json
```

```bash
# Find all FKs that reference this table (PostgreSQL)
dq postgres -c <connection> \
  "SELECT tc.table_name AS child_table, kcu.column_name AS child_column, rc.delete_rule
   FROM information_schema.referential_constraints rc
   JOIN information_schema.table_constraints tc ON rc.constraint_name = tc.constraint_name
   JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
   WHERE rc.unique_constraint_name IN (
     SELECT constraint_name FROM information_schema.table_constraints
     WHERE table_name = '<table>' AND constraint_type IN ('PRIMARY KEY', 'UNIQUE')
   )" --output json
```

If `delete_rule` is `CASCADE`, count the child rows that would also be deleted:

```bash
dq postgres -c <connection> \
  "SELECT COUNT(*) AS cascade_count FROM <child_table>
   WHERE <child_column> IN (
     SELECT id FROM <table> WHERE <delete_where_clause>
   )" --output json
```

## Step 5: Present the Impact Summary

Before asking for confirmation, present a clear summary. Format it like this:

> **Mutation:** UPDATE `users` SET `status = 'expired'`
> **Rows affected:** 342 out of 50,000 total rows (0.7%)
> **Cascading effects:** None
> **Reversibility:** Can revert by setting status back to 'trial' for these rows

If the mutation affects more than 10% of the table, warn explicitly.

## Step 6: Wait for User Confirmation

Ask the user to confirm before executing. Never auto-execute mutations. Present the exact SQL that will run.

- If the user says no, stop. Do not execute.
- If the user wants to modify the query, go back to Step 1.

## Step 7: Execute the Mutation

Run the same SQL without `--dry-run`:

```bash
dq postgres -c <connection> \
  "UPDATE users SET status = 'expired' WHERE status = 'trial' AND created_at < '2025-01-01'" \
  --output json
```

## Step 8: Verify the Result

Query the affected rows to confirm the change took effect:

```bash
# Confirm the update landed
dq postgres -c <connection> \
  "SELECT id, status, updated_at FROM users WHERE status = 'expired' ORDER BY updated_at DESC" \
  --limit 10 --output json
```

```bash
# Confirm the count matches expectations
dq postgres -c <connection> \
  "SELECT COUNT(*) AS updated_count FROM users WHERE status = 'expired' AND updated_at >= NOW() - INTERVAL '1 minute'" \
  --output json
```

If the count does not match the dry-run prediction, investigate immediately.

## MySQL Differences

- `--explain` produces MySQL EXPLAIN format (no `EXPLAIN ANALYZE` before MySQL 8.0.18).
- CASCADE behavior: check `information_schema.REFERENTIAL_CONSTRAINTS` with `DELETE_RULE` column.
- Use `dq mysql -c <connection> "<sql>"` instead of `dq postgres`.

## SQLite Differences

- Foreign key enforcement must be enabled per-connection (`PRAGMA foreign_keys = ON`).
- No `--explain` support for mutation plans — use `EXPLAIN QUERY PLAN` inline.
- Use `dq sqlite -c <connection> "<sql>"` instead of `dq postgres`.
