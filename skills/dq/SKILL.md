---
name: dq
description: Agent-first database CLI for discovering, querying, introspecting, and annotating databases. Use when the user wants to explore a database, run SQL, inspect schema, manage connections, or persist knowledge about data.
---

# dq

dq is an agent-first cli to query databases. It supports PostgreSQL, MySQL, and SQLite. Output is structured JSON when piped and human-readable tables when interactive.

Every query command requires `-c <connection>` to specify which database to use.

## Default Behavior

If the user mentions a database or wants to explore data:
- Check existing connections with `dq connection list --output json`.
- If no connection exists, ask the user for details and add one with `dq connection add`.
- Orient yourself with `dq discover -c <name> --output json` before querying.
- Use annotations to persist knowledge learned during the conversation.

When exploring an unfamiliar database, follow this priority order:
1. Run `dq discover` to get the full schema overview (cached after first call).
2. Read annotations merged into the discover output for prior knowledge.
3. Use `dq schema describe` for deep dives into specific tables.
4. Query with the backend-specific command (`dq postgres`, `dq mysql`, `dq sqlite`).

Boundary rules:
- Always use `--explain` before running mutations to check the query plan.
- Use `SELECT COUNT(*)` with the same WHERE clause to estimate affected rows before mutating.
- Never expose columns marked as PII in annotations unless the user explicitly asks.
- Respect `--limit` to avoid flooding context with large result sets.
- Use `--fields` to select only the columns you need.

## Quick Start

1. **Add a connection**: `dq connection add mydb --type postgres --host localhost --port 5432 --database myapp --user admin --password env:DB_PASS`
2. **Discover the database**: `dq discover -c mydb --output json` — returns full schema hierarchy (cached)
3. **Query**: `dq postgres -c mydb "SELECT * FROM users LIMIT 5" --output json`
4. **Annotate**: `dq annotate set -c mydb --table users --column email --note "PII - do not expose"`

## Connections

Connection config is stored in `~/.config/dq/config.yaml`.

Add a connection:

```bash
dq connection add prod-pg \
  --type postgres \
  --host db.prod.example.com \
  --port 5432 \
  --database myapp \
  --user readonly \
  --password "env:PROD_DB_PASSWORD" \
  --ssl-mode require
```

For SQLite:

```bash
dq connection add local-db --type sqlite --path ./data.db
```

List connections:

```bash
dq connection list --output json
```

Test a connection:

```bash
dq connection test prod-pg --output json
```

Show connection details:

```bash
dq connection show prod-pg --output json
# Use --reveal to show password
dq connection show prod-pg --reveal --output json
```

Remove a connection:

```bash
dq connection remove prod-pg
```

Password formats (most to least secure):
- `--store-in-keyring` — stores password in OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service). Config file stores only `keyring:<name>` reference.
- `env:VAR_NAME` — reads from environment variable at connect time
- plain text — stored as-is in config file (warns on creation)

Use `--password-stdin` to avoid leaking the password in shell history or `/proc`.

## Discover — Agent Orientation

This is the most important command. It returns a hierarchical view of the entire database: schemas, tables, columns (with types, PKs, nullability), foreign keys, indexes, row counts, sizes, and annotations.

```bash
dq discover -c prod-pg --output json
```

Results are cached to `~/.config/dq/cache/<connection>/discover.json`. Subsequent calls return the cache instantly. Use `--refresh` to re-introspect from the live database.

```bash
dq discover -c prod-pg --refresh --output json
```

The cache includes a `cached_at` timestamp so you can decide if it is stale.

Annotations from `~/.config/dq/annotations/<connection>.yaml` are merged into the discover output automatically. You see schema and business context in one call.

## Choosing A Query Strategy

| Task | Command | Why |
|---|---|---|
| Explore database structure | `dq discover -c <name>` | Full overview, cached, annotations merged |
| Deep dive into one table | `dq schema describe -c <name> --table <t>` | Columns, indexes, constraints, annotations |
| Run a SELECT query | `dq postgres -c <name> <sql>` | Returns structured result with metadata |
| Estimate mutation impact | `dq postgres -c <name> "SELECT COUNT(*) ..." ` | Count rows matching WHERE clause before mutating |
| Check query plan | `dq postgres -c <name> <sql> --explain` | Shows query plan without executing |
| Check CLI capabilities | `dq schema capabilities --output json` | Runtime self-introspection |

## Running Queries

Queries use the backend-specific command. The SQL is passed as the first argument.

```bash
dq postgres -c prod-pg "SELECT * FROM users ORDER BY created_at DESC LIMIT 10" --output json
```

```bash
dq mysql -c staging "SELECT COUNT(*) FROM orders WHERE status = 'pending'" --output json
```

```bash
dq sqlite -c local-db "SELECT * FROM events WHERE date > '2026-01-01'" --output json
```

Shared flags for all query commands:

| Flag | Purpose |
|---|---|
| `--output json\|table\|csv\|ndjson` | Output format |
| `--fields col1,col2` | Filter columns in output |
| `--limit N` | Maximum rows to return |
| `--offset N` | Skip rows (pagination) |
| `--timeout 30s` | Query timeout |
| `--explain` | Prepend EXPLAIN to the query |

Result envelope:

```json
{
  "meta": {
    "connection": "prod-pg",
    "database": "myapp",
    "row_count": 10,
    "duration_ms": 42,
    "limit": 10
  },
  "columns": [
    {"name": "id", "type": "INT8"},
    {"name": "email", "type": "VARCHAR"}
  ],
  "rows": [
    {"id": 1, "email": "alice@example.com"}
  ]
}
```

### Query Patterns

**With Pagination:**

```bash
dq postgres -c mydb "SELECT * FROM users ORDER BY created_at DESC" --limit 10 --offset 20
```

**Field Filtering:**

```bash
dq postgres -c mydb "SELECT * FROM users" --fields id,email,created_at
```

**With Timeout:**

```bash
dq postgres -c mydb "SELECT * FROM large_table" --timeout 60s
```

**EXPLAIN:**

```bash
dq postgres -c mydb "SELECT * FROM users WHERE email = 'test@example.com'" --explain
```

**Export Results:**

```bash
# Pipe query output to a file using shell redirection
dq postgres -c mydb "SELECT * FROM users" --output csv > users.csv
dq postgres -c mydb "SELECT * FROM users" --output ndjson > users.ndjson
```

## Safe Mutation Pattern

Before any mutation, always:

1. **Check the query plan**: `dq postgres -c prod-pg "DELETE FROM users WHERE status = 'inactive'" --explain --output json`
2. **Count affected rows**: `dq postgres -c prod-pg "SELECT COUNT(*) AS affected FROM users WHERE status = 'inactive'" --output json`
3. **Preview a sample**: `dq postgres -c prod-pg "SELECT * FROM users WHERE status = 'inactive'" --limit 10 --output json`
4. **Ask the user to confirm** before executing the mutation.

## Schema Introspection

Schema commands always hit the live database (no caching).

List tables:

```bash
dq schema tables -c prod-pg --output json
```

List columns:

```bash
dq schema columns -c prod-pg --table users --output json
```

List indexes:

```bash
dq schema indexes -c prod-pg --table users --output json
```

List constraints:

```bash
dq schema constraints -c prod-pg --table users --output json
```

Full table description (columns + indexes + constraints + annotations):

```bash
dq schema describe -c prod-pg --table users --output json
```

Runtime CLI self-introspection:

```bash
dq schema capabilities --output json
```

The capabilities command returns all available commands, flags, output formats, exit codes, backends, and features. Use this to discover what dq can do at runtime.

## Annotations — Persistent Knowledge Base

Annotations persist knowledge about data between conversations. Store them as notes on tables or columns.

Set a table annotation:

```bash
dq annotate set -c prod-pg --table users --note "Core user accounts table"
```

Set a column annotation:

```bash
dq annotate set -c prod-pg --table users --column email --note "PII - never expose in logs"
dq annotate set -c prod-pg --table orders --column total --note "Amount in cents, divide by 100 for dollars"
```

Get annotations:

```bash
# All annotations for a connection
dq annotate get -c prod-pg --output json

# For a specific table
dq annotate get -c prod-pg --table users --output json
```

Remove annotations:

```bash
dq annotate remove -c prod-pg --table users --column email
dq annotate remove -c prod-pg --table users
```

Annotations are stored as YAML in `~/.config/dq/annotations/<connection>.yaml` and are automatically merged into `dq discover` output.

When you learn something about the data during a conversation, annotate it so future conversations benefit.

## Output Formats

| Format | When used | Flag |
|---|---|---|
| `json` | Default when piped, structured output | `--output json` |
| `table` | Default when TTY, human-readable | `--output table` |
| `csv` | Data export | `--output csv` |
| `ndjson` | Streaming, line-delimited JSON | `--output ndjson` |

Override auto-detection with `--output` or the `DQ_OUTPUT` environment variable.

Errors are always written to stderr as JSON:

```json
{"error":"type","message":"description","suggestion":"what to do"}
```

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Error |
| 2 | Usage error (missing flags, bad args) |
| 3 | Not found |
| 4 | Auth failure |
| 5 | Conflict |
| 6 | Timeout |

## Error Handling

| Error | Meaning | Action |
|---|---|---|
| Connection refused | Database not reachable | Verify host, port, and network with `dq connection test` |
| Auth failure | Bad credentials | Check password format (`env:VAR` / `keyring:name`), verify env var is set or keyring entry exists |
| Connection not found | Name not in config | Run `dq connection list` to see available connections |
| Timeout | Query exceeded limit | Increase `--timeout` or optimize the query |
| Unsupported database type | Backend not registered | Use `postgres`, `mysql`, or `sqlite` |

## Troubleshooting

### Connection Issues

```bash
# Test connectivity
dq connection test prod --output json

# Show connection config
dq connection show prod --output json

# Re-add with correct credentials
dq connection remove prod
dq connection add prod --type postgres --host ... --password "env:DB_PASS"
```

### Investigate Database State via SQL

```bash
# Find slow queries (PostgreSQL)
dq postgres -c prod "SELECT pid, state, query, now() - query_start AS duration FROM pg_stat_activity WHERE state = 'active' ORDER BY duration DESC" --output json

# Check locks (PostgreSQL)
dq postgres -c prod "SELECT l.pid, l.mode, l.granted, c.relname FROM pg_locks l LEFT JOIN pg_class c ON l.relation = c.oid WHERE NOT l.granted" --output json

# Table sizes (PostgreSQL)
dq postgres -c prod "SELECT relname, pg_size_pretty(pg_total_relation_size(oid)) AS size FROM pg_class WHERE relkind = 'r' ORDER BY pg_total_relation_size(oid) DESC LIMIT 10" --output json

# Database stats (PostgreSQL)
dq postgres -c prod "SELECT pg_size_pretty(pg_database_size(current_database())) AS size, (SELECT count(*) FROM pg_stat_activity WHERE state = 'active') AS active_conns" --output json
```

### Schema Investigation

```bash
# Orient yourself
dq discover -c prod --output json

# Deep dive into a specific table
dq schema describe -c prod --table orders --output json

# Refresh cached discovery
dq discover -c prod --refresh --output json
```

### Annotate Findings

```bash
# Persist what you learn for future sessions
dq annotate set -c prod --table orders --note "Needs index on created_at, queries slow"
dq annotate set -c prod --table users --column status --note "1=active, 2=suspended, 3=deactivated"
```

## Guidelines

- Always include `--output json` when parsing results programmatically.
- Use `dq discover` to orient before querying — do not guess table or column names.
- Annotate PII columns immediately when you identify them.
- For mutations, use `--explain` and `SELECT COUNT(*)` to assess impact, then confirm with the user before executing.
- Use `--fields` and `--limit` to keep output within context window limits.

## Typical Agent Workflow

```bash
# 1. Orient
dq discover -c prod-pg --output json

# 2. Query
dq postgres -c prod-pg "SELECT * FROM users ORDER BY created_at DESC LIMIT 5" --output json

# 3. Annotate what you learn
dq annotate set -c prod-pg --table users --column status --note "1=active, 2=suspended, 3=deactivated"

# 4. Deep dive if needed
dq schema describe -c prod-pg --table orders --output json

# 5. Check impact before mutations
dq postgres -c prod-pg "UPDATE users SET status = 2 WHERE last_login < '2025-01-01'" --explain --output json
dq postgres -c prod-pg "SELECT COUNT(*) AS affected FROM users WHERE last_login < '2025-01-01'" --output json
```

## Charts — Visualize Query Results

Generate interactive HTML charts from any query result. Charts auto-open in the browser.

```bash
# Pipe query output to chart
dq postgres -c prod-pg "SELECT month, revenue FROM monthly_stats ORDER BY month" -o json \
  | dq chart --type line --x month --y revenue --title "Revenue Trend"

# Multiple series
dq postgres -c prod-pg "SELECT month, revenue, cost FROM stats ORDER BY month" -o json \
  | dq chart --type area --x month --y revenue,cost

# Grouped bar chart
dq postgres -c prod-pg "SELECT quarter, region, SUM(amount) AS revenue FROM sales GROUP BY 1, 2" -o json \
  | dq chart --type bar --x quarter --y revenue --group region

# Pie chart
dq postgres -c prod-pg "SELECT status, COUNT(*) AS count FROM users GROUP BY status" -o json \
  | dq chart --type pie --x status --y count

# Save without opening
dq chart --type line --x month --y revenue --from results.json --save report.html --no-open
```

Chart types: `line`, `bar`, `area`, `scatter`, `pie`

## Playbooks — Reusable Analytics Workflows

Playbooks encode org-specific analytics workflows, business logic, and data knowledge as markdown files that persist across sessions.

```bash
# Generate a template
dq playbook init monthly-revenue

# Edit the template, then add it
dq playbook add monthly-revenue --file monthly-revenue.md

# List and filter playbooks
dq playbook list
dq playbook list --tag revenue

# Show a playbook's full content
dq playbook show monthly-revenue

# Remove
dq playbook remove monthly-revenue
```

When starting an analysis, check for relevant playbooks with `dq playbook list` and follow them with `dq playbook show <name>`.

## Optional Recipe Skills

For multi-step workflows, install additional skills:

```sh
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-cold-start
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-find-table
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-trace-relationships
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-data-profiling
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-sample-and-summarize
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-query-impact-analysis
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-safe-mutation
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-safe-backfill
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-slow-query-investigation
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-table-health-check
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-find-missing-indexes
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-orphan-check
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-chart
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-playbook
```

| Skill | Description |
|-------|-------------|
| [dq-cold-start](dq-cold-start/) | First encounter with an unknown database: connect, discover, flag PII, annotate key tables. |
| [dq-find-table](dq-find-table/) | Locate a table from a vague user description (fuzzy search, column search, annotate result). |
| [dq-trace-relationships](dq-trace-relationships/) | Map foreign key chains from a table to understand the data model. |
| [dq-data-profiling](dq-data-profiling/) | Profile a table: row count, null rates, distinct counts, min/max per column. |
| [dq-sample-and-summarize](dq-sample-and-summarize/) | Get representative rows, summarize value distributions, flag anomalies. |
| [dq-query-impact-analysis](dq-query-impact-analysis/) | "Is this query safe?" EXPLAIN + row count + cascade check + risk assessment. |
| [dq-safe-mutation](dq-safe-mutation/) | Explain, count, preview, confirm, execute pattern for INSERT/UPDATE/DELETE. |
| [dq-safe-backfill](dq-safe-backfill/) | Batch update pattern: preview, execute in chunks, track progress. |
| [dq-slow-query-investigation](dq-slow-query-investigation/) | Find slow queries via pg_stat_activity, EXPLAIN them, suggest fixes. |
| [dq-table-health-check](dq-table-health-check/) | Table health: size, bloat, dead tuples, index usage, missing indexes. |
| [dq-find-missing-indexes](dq-find-missing-indexes/) | Identify columns in WHERE/JOIN without indexes, generate CREATE INDEX suggestions. |
| [dq-orphan-check](dq-orphan-check/) | Find rows with broken foreign key references (dangling FKs). |
| [dq-chart](dq-chart/) | Generate interactive HTML charts (line, bar, area, scatter, pie) from query results. |
| [dq-playbook](dq-playbook/) | Create and manage playbooks — reusable analytics workflows and org knowledge. |
