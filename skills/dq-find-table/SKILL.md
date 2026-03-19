---
name: dq-find-table
description: Locate a table when the user gives a vague or partial name. Search table names, then column names, compare candidates, annotate the result, and report the match with schema details.
---

# Recipe: Find Table — Locate a Table from a Vague Description

Use this recipe when the user says something like "find the orders table," "where are workspaces stored?", or "is there a table for invoices?" The goal is to find the right table even when the user's wording does not exactly match the schema, then persist the finding so you never repeat the search.

## Step 1 — Get all tables via discover

Start with the cached schema overview. This gives you every table name, column list, row count, and any existing annotations — without hitting the database again.

```bash
dq discover -c <connection> --output json
```

If the cache might be stale (e.g., the user mentions a table that was recently created), refresh it:

```bash
dq discover -c <connection> --refresh --output json
```

## Step 2 — Search table names for keyword matches

Extract the keyword from the user's request (e.g., "orders," "workspace," "invoice") and search for it in the table names from the discover output. Apply these matching strategies in order:

1. **Exact match**: table name equals the keyword (e.g., `orders`).
2. **Plural/singular variant**: if the keyword is `order`, also check `orders`; if `workspaces`, also check `workspace`.
3. **Prefix match**: table name starts with the keyword (e.g., `order_items`, `order_history`).
4. **Contains match**: table name contains the keyword anywhere (e.g., `customer_orders`, `archived_orders`).
5. **Common abbreviations**: `tx` for transactions, `inv` for invoices, `usr` for users, `org` for organizations, `ws` for workspaces.

**Decision point:**

- If you find **exactly one match**, proceed to Step 4.
- If you find **multiple matches**, proceed to Step 4 to compare them.
- If you find **zero matches**, proceed to Step 3.

## Step 3 — Search column names for the keyword

When no table name matches, the data might live as a column inside a broader table. Scan all columns across all tables for the keyword.

```bash
dq schema tables -c <connection> --output json
```

For example, if the user asks "where are workspaces stored?" and no `workspaces` table exists, look for a `workspace_id` or `workspace_name` column. The table containing that column is likely the answer.

**Decision point:**

- If you find columns matching the keyword, the table(s) containing them are your candidates. Proceed to Step 4.
- If you still find nothing, tell the user that no table or column matches their description. Suggest they rephrase or check whether the feature exists in this database. You can also try a broader search:

```bash
dq postgres -c <connection> "SELECT table_name, column_name FROM information_schema.columns WHERE column_name ILIKE '%<keyword>%' ORDER BY table_name" --output json
```

> **MySQL:** Use `LIKE` instead of `ILIKE` (MySQL is case-insensitive by default with most collations).
>
> **SQLite:** Use `LIKE` (case-insensitive for ASCII by default).

## Step 4 — Describe each candidate to compare

For every candidate table, get its full schema details:

```bash
dq schema describe -c <connection> --table <candidate_table> --output json
```

This returns columns (with types, PKs, nullability), indexes, constraints, foreign keys, and annotations.

**If there are multiple candidates**, compare them by:

| Factor | How to evaluate |
|---|---|
| Row count | The table with more rows is usually the primary one |
| Foreign keys | A table referenced by many others is more central |
| Column relevance | Does it have the columns you would expect for the user's concept? |
| Naming convention | `orders` is more likely the main table than `order_archive` |
| Annotations | Prior annotations may already clarify the table's role |

Present the comparison to the user if the choice is ambiguous. Let them confirm before proceeding.

## Step 5 — Annotate the finding

Once you have identified the correct table, annotate it so future sessions find it instantly without repeating this search.

```bash
dq annotate set -c <connection> --table <table> --note "<User-friendly description of what this table stores>"
```

If the user's keyword was different from the table name, include that mapping in the annotation:

```bash
dq annotate set -c <connection> --table subscription_plans --note "Product pricing plans. Users may refer to this as 'plans' or 'pricing'."
```

## Step 6 — Report the match

Present the result to the user with these details:

1. **Table name** and which schema it belongs to.
2. **Row count** — gives a sense of scale.
3. **Key columns** — primary key, foreign keys, and any columns that seem most relevant to the user's question.
4. **Column list** — full list with types, noting which are nullable and which are indexed.
5. **Relationships** — foreign keys pointing to or from this table.
6. **Annotations** — any existing notes (including PII flags).

Example summary format:

```
Found: public.subscription_plans (12,450 rows)

Primary key: id (INT8)
Foreign keys:
  - product_id -> products.id
  - currency_id -> currencies.id

Key columns:
  - name (VARCHAR) — plan display name
  - price_cents (INT4) — price in cents
  - interval (VARCHAR) — 'monthly' or 'yearly'
  - active (BOOL) — whether the plan is currently offered

Referenced by:
  - subscriptions.plan_id
  - invoice_line_items.plan_id
```

Keep it concise. The user asked "where is X?" — give them the answer, the context to understand it, and move on.
