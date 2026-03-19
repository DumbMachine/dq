---
name: dq-cold-start
description: First encounter with an unknown database. Establish a connection, discover the full schema, identify large tables and PII columns, annotate key domain entities, and summarize findings to the user.
---

# Recipe: Cold Start — First Encounter with an Unknown Database

Use this recipe when the user points you at a database you have never seen before. The goal is to orient quickly, flag risks, persist what you learn, and give the user a useful summary — all before they ask their first real question.

## Step 1 — Check for existing connections

Before creating anything, see if a connection already exists.

```bash
dq connection list --output json
```

- If the output contains a connection that matches what the user described, skip to Step 3.
- If no connections exist (or none match), proceed to Step 2.

## Step 2 — Add a new connection

Ask the user for connection details: database type, host, port, database name, user, and password. Prefer `--store-in-keyring` (OS keychain) or `env:VAR_NAME` for passwords so credentials stay out of config files.

**PostgreSQL example:**

```bash
dq connection add <connection> \
  --type postgres \
  --host <host> \
  --port 5432 \
  --database <database> \
  --user <user> \
  --password "env:DB_PASSWORD" \
  --ssl-mode require
```

**MySQL example:**

```bash
dq connection add <connection> \
  --type mysql \
  --host <host> \
  --port 3306 \
  --database <database> \
  --user <user> \
  --password "env:DB_PASSWORD"
```

**SQLite example:**

```bash
dq connection add <connection> --type sqlite --path <path-to-file>
```

## Step 3 — Test the connection

Verify that the connection actually works before doing anything else. A failed test here saves time debugging later.

```bash
dq connection test <connection> --output json
```

- If the test fails with exit code 4 (auth failure), double-check credentials and whether the env var is set.
- If the test fails with exit code 1 (connection refused), verify host, port, and network access.
- Do not proceed until the test passes.

## Step 4 — Run discover to get the full schema overview

This is the single most important command. It returns schemas, tables, columns (with types, PKs, nullability), foreign keys, indexes, row counts, sizes, and any existing annotations — all in one call.

```bash
dq discover -c <connection> --output json
```

Parse the JSON output. You now have everything you need for Steps 5-7.

> **Note:** Results are cached after the first call. If the database may have changed since the last run, add `--refresh`.

## Step 5 — Identify large tables

Scan the discover output for tables with high row counts. Large tables affect query performance and are important to flag early.

**Decision points:**

- If a table has **> 1 million rows**, flag it as "large — use LIMIT and WHERE clauses, check for indexes before querying."
- If a table has **> 10 million rows**, flag it as "very large — avoid full-table scans, prefer indexed lookups, use `--timeout` on queries."
- If you want live row counts instead of cached estimates, run a targeted SQL query:

```bash
dq postgres -c <connection> "SELECT relname AS table_name, n_live_tup AS estimated_rows FROM pg_stat_user_tables ORDER BY n_live_tup DESC LIMIT 20" --output json
```

> **MySQL alternative:** `SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema = DATABASE() ORDER BY table_rows DESC LIMIT 20`
>
> **SQLite alternative:** There is no row-count metadata. Run `SELECT COUNT(*) FROM <table>` per table, but be cautious with large databases.

Annotate large tables so future sessions know immediately:

```bash
dq annotate set -c <connection> --table <large_table> --note "Large table (~<N>M rows). Use LIMIT and indexed lookups."
```

## Step 6 — Identify likely PII columns

Scan column names across all tables for patterns that suggest personally identifiable information. Look for:

| Pattern | Likely PII type |
|---|---|
| `email`, `e_mail`, `email_address` | Email address |
| `phone`, `phone_number`, `mobile`, `tel` | Phone number |
| `ssn`, `social_security`, `sin`, `national_id` | Government ID |
| `password`, `pass_hash`, `password_hash`, `passwd` | Credential |
| `first_name`, `last_name`, `full_name`, `name` (on a users/people table) | Personal name |
| `address`, `street`, `city`, `zip`, `postal` | Physical address |
| `dob`, `date_of_birth`, `birth_date`, `birthday` | Date of birth |
| `ip_address`, `ip`, `user_agent` | Network identity |
| `credit_card`, `card_number`, `pan`, `cvv` | Payment card |

For each match, annotate the column immediately:

```bash
dq annotate set -c <connection> --table <table> --column <column> --note "PII - <type>. Do not expose in logs or unfiltered output."
```

This annotation will appear in all future `dq discover` and `dq schema describe` calls, warning any agent (including yourself in a future session) to handle the column carefully.

## Step 7 — Annotate key domain tables

Look at table names for common domain patterns and annotate them with their likely purpose:

| Pattern | Likely role |
|---|---|
| `users`, `accounts`, `members` | Core identity table |
| `orders`, `purchases`, `transactions` | Commercial activity |
| `products`, `items`, `inventory` | Catalog |
| `sessions`, `tokens` | Auth / session management |
| `events`, `logs`, `audit_log` | Event stream / audit trail |
| `permissions`, `roles`, `acl` | Authorization |
| `migrations`, `schema_migrations` | Framework metadata (usually ignorable) |

```bash
dq annotate set -c <connection> --table users --note "Core user accounts table. Contains PII columns."
dq annotate set -c <connection> --table orders --note "Primary orders table. Linked to users via user_id FK."
```

Also annotate any junction/join tables (e.g., `user_roles`, `order_items`) with their relationship context.

## Step 8 — Summarize findings to the user

Present a clear summary covering:

1. **Connection**: name, type, database, host.
2. **Scale**: total number of schemas, tables, and estimated total rows.
3. **Large tables**: list tables flagged in Step 5 with their row counts.
4. **PII columns**: list tables and columns flagged in Step 6.
5. **Key domain tables**: brief description of each annotated table from Step 7.
6. **Foreign key highlights**: mention tables with many FKs (likely central entities) and tables with zero FKs (likely leaf or lookup tables).
7. **Recommendations**: suggest next steps — e.g., "Run `dq schema describe` on the `orders` table to explore its relationships" or "The `events` table has 50M rows; use indexed filters when querying."

Keep the summary scannable. Use a table or bullet list, not walls of text.
