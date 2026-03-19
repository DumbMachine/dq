---
name: dq-trace-relationships
description: Follow foreign key chains from a starting table to map the data model. Describe each table's FKs, traverse referenced tables up to a configurable depth, build a relationship map, annotate findings, and present the result.
---

# Recipe: Trace Relationships — Map the Data Model from a Starting Table

Use this recipe when the user wants to understand how a table connects to the rest of the database. The goal is to follow foreign key chains outward from a starting table, build a clear relationship map, and persist what you learn.

## Step 1 — Describe the starting table to get its foreign keys

Get the full schema details for the table the user is interested in. This returns columns, indexes, constraints, and — critically — foreign key definitions.

```bash
dq schema describe -c <connection> --table <starting_table> --output json
```

From the output, extract:

- **Outgoing FKs** (this table references another): e.g., `orders.user_id -> users.id`
- **Incoming FKs** (another table references this one): e.g., `order_items.order_id -> orders.id`

If the table has **no foreign keys**, it may still have implicit relationships (columns named `*_id` without formal FK constraints). In that case, look for columns ending in `_id` and check whether a matching table exists:

```bash
dq postgres -c <connection> "SELECT column_name FROM information_schema.columns WHERE table_name = '<starting_table>' AND column_name LIKE '%_id'" --output json
```

> **MySQL:** Same query works. **SQLite:** Use `PRAGMA table_info('<starting_table>')` and filter for `_id` columns.

## Step 2 — Describe each directly referenced table

For every table found via outgoing FKs in Step 1, get its full schema:

```bash
dq schema describe -c <connection> --table <referenced_table> --output json
```

Record:

- The table name and its row count
- Its own outgoing FKs (you will need these for Step 3)
- Its primary key and key columns
- Any existing annotations

Do the same for tables with incoming FKs (tables that reference the starting table). These are the "children" in the relationship.

**Decision point:**

- If there are **more than 10 directly related tables**, prioritize: describe the ones the user seems most interested in first, and list the rest by name. Ask the user if they want to expand any of them.
- If there are **zero related tables** (no FKs in either direction), tell the user and suggest checking for implicit relationships or looking at the discover output for tables with similar naming patterns.

## Step 3 — Traverse one more level deep

For each table described in Step 2, follow its outgoing FKs one more level. This gives you a two-hop relationship map from the starting table.

```bash
dq schema describe -c <connection> --table <second_level_table> --output json
```

**Depth is configurable:** The default is 2 levels (starting table + 1 hop + 1 more hop). If the user asks to go deeper, repeat this step. If the graph is very wide (many tables at each level), ask the user which branches to follow rather than expanding everything.

**Cycle detection:** If a table at level 2 points back to the starting table or to a table already visited, note the cycle but do not traverse it again. Mention it in the final map (e.g., "orders -> users -> orders (cycle)").

## Step 4 — Build the relationship summary

Organize the collected data into a relationship map. Use this structure:

```
<starting_table>
  |-- <fk_column> -> <referenced_table>.<referenced_column>  (outgoing)
  |     |-- <fk_column> -> <second_level_table>.<column>      (level 2)
  |     |-- <fk_column> -> <second_level_table>.<column>      (level 2)
  |-- <fk_column> -> <referenced_table>.<referenced_column>  (outgoing)
  |
  |<- <child_table>.<fk_column>                               (incoming)
  |<- <child_table>.<fk_column>                               (incoming)
```

Example for an `orders` table:

```
orders
  |-- user_id -> users.id
  |     |-- users.org_id -> organizations.id
  |     |-- users.plan_id -> subscription_plans.id
  |-- shipping_address_id -> addresses.id
  |-- coupon_id -> coupons.id
  |
  |<- order_items.order_id          (14 columns, ~2.3M rows)
  |<- payments.order_id             (8 columns, ~1.1M rows)
  |<- shipments.order_id            (12 columns, ~800K rows)
  |<- order_status_history.order_id (6 columns, ~5.2M rows)
```

Include row counts and column counts for each related table so the user can gauge their scale.

## Step 5 — Annotate the starting table with relationship context

Persist the relationship map as an annotation on the starting table. This saves future sessions from re-traversing the graph.

```bash
dq annotate set -c <connection> --table <starting_table> --note "Relationships: user_id->users, shipping_address_id->addresses, coupon_id->coupons. Referenced by: order_items, payments, shipments, order_status_history."
```

If any of the related tables lack annotations, annotate them too:

```bash
dq annotate set -c <connection> --table order_items --note "Line items for orders. FK: order_id->orders, product_id->products."
dq annotate set -c <connection> --table payments --note "Payment records. FK: order_id->orders, payment_method_id->payment_methods."
```

Keep annotation text concise — it will be included in every future `dq discover` call.

## Step 6 — Present the relationship map to the user

Give the user:

1. **The relationship tree** from Step 4, formatted for readability.
2. **Key observations:**
   - Which table is the "center of gravity" (most FKs pointing to/from it)?
   - Are there any tables with no incoming FKs (potential root/entry-point tables)?
   - Are there junction tables indicating many-to-many relationships (e.g., `user_roles` linking `users` and `roles`)?
   - Are there any cycles in the graph?
   - Are there any `_id` columns without formal FK constraints (potential missing constraints)?
3. **Suggested next steps:**
   - "Run a JOIN query across `orders` and `order_items` to see line-item details."
   - "The `order_status_history` table has 5.2M rows — use indexed filters when querying."
   - "Consider tracing from `users` next to see the full user-centric data model."

If the user asked about a specific relationship (e.g., "how are orders connected to products?"), highlight that specific path in the tree and provide a sample JOIN query:

```bash
dq postgres -c <connection> "SELECT o.id AS order_id, oi.quantity, p.name AS product_name FROM orders o JOIN order_items oi ON o.id = oi.order_id JOIN products p ON oi.product_id = p.id LIMIT 10" --output json
```
