---
name: dq-orphan-check
description: Find rows with broken foreign key references. Detects orphaned records where the referenced parent row no longer exists, reports impact, and suggests remediation.
---

# Recipe: Orphan Check

Use this recipe to find rows that reference a parent row via a foreign key, but the parent no longer exists. Orphaned rows cause application errors, inconsistent reports, and data integrity issues. This is especially common in databases where FK constraints were added after data already existed, or where constraints use SET NULL / NO ACTION instead of CASCADE.

## Step 1: Get All Foreign Key Constraints

Start by identifying every FK relationship on the target table (or all tables if doing a full audit).

For a specific table:

```bash
dq schema describe -c <connection> --table <table> --output json
```

For all FK constraints in the database:

```bash
dq postgres -c <connection> \
  "SELECT
     tc.table_schema,
     tc.table_name AS child_table,
     kcu.column_name AS child_column,
     ccu.table_schema AS parent_schema,
     ccu.table_name AS parent_table,
     ccu.column_name AS parent_column,
     rc.delete_rule,
     rc.update_rule
   FROM information_schema.table_constraints tc
   JOIN information_schema.key_column_usage kcu
     ON tc.constraint_name = kcu.constraint_name
     AND tc.table_schema = kcu.table_schema
   JOIN information_schema.constraint_column_usage ccu
     ON tc.constraint_name = ccu.constraint_name
   JOIN information_schema.referential_constraints rc
     ON tc.constraint_name = rc.constraint_name
   WHERE tc.constraint_type = 'FOREIGN KEY'
     AND tc.table_schema NOT IN ('pg_catalog', 'information_schema')
   ORDER BY tc.table_name, kcu.column_name" \
  --output json
```

This gives you every child-parent relationship to check.

## Step 2: Find Orphaned Rows for Each FK

For each FK constraint, run a LEFT JOIN query to find child rows where the parent row does not exist:

```bash
dq postgres -c <connection> \
  "SELECT COUNT(*) AS orphan_count
   FROM <child_table> child
   LEFT JOIN <parent_table> parent
     ON child.<child_column> = parent.<parent_column>
   WHERE parent.<parent_column> IS NULL
     AND child.<child_column> IS NOT NULL" \
  --output json
```

The `AND child.<child_column> IS NOT NULL` clause excludes rows where the FK column is intentionally NULL (nullable FKs are valid).

If orphans exist, get a sample of the orphaned rows:

```bash
dq postgres -c <connection> \
  "SELECT child.id, child.<child_column> AS orphaned_fk_value,
          child.created_at
   FROM <child_table> child
   LEFT JOIN <parent_table> parent
     ON child.<child_column> = parent.<parent_column>
   WHERE parent.<parent_column> IS NULL
     AND child.<child_column> IS NOT NULL
   ORDER BY child.id
   LIMIT 10" \
  --output json
```

To see which parent IDs are referenced but missing:

```bash
dq postgres -c <connection> \
  "SELECT DISTINCT child.<child_column> AS missing_parent_id,
          COUNT(*) AS referencing_rows
   FROM <child_table> child
   LEFT JOIN <parent_table> parent
     ON child.<child_column> = parent.<parent_column>
   WHERE parent.<parent_column> IS NULL
     AND child.<child_column> IS NOT NULL
   GROUP BY child.<child_column>
   ORDER BY referencing_rows DESC
   LIMIT 20" \
  --output json
```

## Step 3: Report Findings

Present a summary for each FK relationship checked. Format:

> **Table:** `orders`
> **FK Column:** `customer_id` -> `customers.id`
> **Orphaned Rows:** 47 out of 125,000 (0.04%)
> **Missing Parent IDs:** 12 distinct customer IDs no longer exist
> **Sample orphaned values:** 1042, 1055, 1089, 2301, ...
> **Delete rule:** NO ACTION (orphans were not cleaned up when parents were deleted)

If no orphans are found for a relationship, report it as clean:

> **Table:** `order_items`
> **FK Column:** `order_id` -> `orders.id`
> **Status:** Clean (0 orphans)

## Step 4: Suggest Remediation

Based on the findings, recommend one of these approaches:

**Option A: Delete orphaned rows** (when the child rows are meaningless without the parent)

```bash
# Count orphaned rows first
dq postgres -c <connection> \
  "SELECT COUNT(*) AS orphan_count FROM <child_table>
   WHERE <child_column> NOT IN (SELECT <parent_column> FROM <parent_table>)
     AND <child_column> IS NOT NULL" \
  --output json
```

Then follow the [dq-safe-mutation](../dq-safe-mutation/) recipe to execute.

**Option B: Re-create missing parent rows** (when the child data is valuable and needs a valid parent)

```bash
# Find the distinct missing parent IDs
dq postgres -c <connection> \
  "SELECT DISTINCT child.<child_column> AS missing_id
   FROM <child_table> child
   LEFT JOIN <parent_table> parent
     ON child.<child_column> = parent.<parent_column>
   WHERE parent.<parent_column> IS NULL
     AND child.<child_column> IS NOT NULL" \
  --output json
```

Then create placeholder parent rows for each missing ID with appropriate default values.

**Option C: Set orphaned FK columns to NULL** (when the FK is nullable and the relationship is optional)

```bash
dq postgres -c <connection> \
  "SELECT COUNT(*) AS affected_rows FROM <child_table>
   WHERE <child_column> NOT IN (SELECT <parent_column> FROM <parent_table>)
     AND <child_column> IS NOT NULL" \
  --output json
```

**Option D: Add or fix the FK constraint** (when no constraint exists yet)

If the table lacks a formal FK constraint, recommend adding one after cleaning up orphans:

```sql
-- After orphans are resolved:
ALTER TABLE <child_table>
  ADD CONSTRAINT fk_<child_table>_<child_column>
  FOREIGN KEY (<child_column>) REFERENCES <parent_table> (<parent_column>)
  ON DELETE CASCADE;  -- or SET NULL, RESTRICT, depending on requirements
```

## Step 5: Annotate Findings

Persist the results so future sessions know about the data quality state:

```bash
dq annotate set -c <connection> --table <child_table> \
  --column <child_column> \
  --note "Orphan check <date>: Found 47 orphaned rows referencing missing <parent_table> records. IDs: 1042, 1055, ... Status: pending cleanup."
```

After remediation:

```bash
dq annotate set -c <connection> --table <child_table> \
  --column <child_column> \
  --note "Orphan check <date>: Cleaned. 47 orphaned rows deleted. FK constraint verified."
```

## Full Database Audit

To check all FK relationships at once, iterate through the results from Step 1. For each row in the FK constraints list, run the orphan check from Step 2. Collect all results and present a single summary table:

| Child Table | FK Column | Parent Table | Orphan Count | Total Rows | Percentage |
|-------------|-----------|--------------|--------------|------------|------------|
| orders      | customer_id | customers  | 47           | 125,000    | 0.04%      |
| order_items | order_id  | orders       | 0            | 890,000    | 0%         |
| payments    | order_id  | orders       | 3            | 95,000     | 0.003%     |

## MySQL Differences

- FK constraints query: use the same `information_schema` tables (MySQL supports them).
- Orphan detection queries (LEFT JOIN pattern) work identically.
- Use `dq mysql -c <connection> "<sql>"` instead of `dq postgres`.

## SQLite Differences

- List FKs with `PRAGMA foreign_key_list('<table>')`.
- Check enforcement: `PRAGMA foreign_keys` (returns 0 if not enforced — orphans are very common in this case).
- SQLite has a built-in orphan check: `PRAGMA foreign_key_check('<table>')` returns all violations directly.
- Use `dq sqlite -c <connection> "<sql>"` instead of `dq postgres`.
