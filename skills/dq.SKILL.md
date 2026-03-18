---
name: dq
description: Agent-first database CLI for discovering, querying, introspecting, and annotating databases. Use when the user wants to explore a database, run SQL, inspect schema, manage connections, or persist knowledge about data.
---

# dq

dq is an agent-first database CLI. It supports PostgreSQL, MySQL, and SQLite. Output is structured JSON when piped and human-readable tables when interactive.

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
- Always use `--dry-run` before running mutations unless the user explicitly says to execute.
- Never expose columns marked as PII in annotations unless the user explicitly asks.
- Respect `--limit` to avoid flooding context with large result sets.
- Use `--fields` to select only the columns you need.

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
| Preview a mutation | `dq postgres -c <name> <sql> --dry-run` | Transaction rollback, shows affected rows |
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
| `--dry-run` | Wrap in transaction and rollback |
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
    "dry_run": false,
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

## Dry Run

Add `--dry-run` to any mutation. dq wraps the SQL in `BEGIN` → execute → `ROLLBACK`. You get real `affected_rows` and constraint errors without modifying data.

```bash
dq postgres -c prod-pg "DELETE FROM users WHERE status = 'inactive'" --dry-run --output json
```

Always use `--dry-run` first for destructive operations unless the user explicitly says to execute.

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
| 7 | Dry run completed successfully |

## Error Handling

| Error | Meaning | Action |
|---|---|---|
| Connection refused | Database not reachable | Verify host, port, and network with `dq connection test` |
| Auth failure | Bad credentials | Check password format (`env:VAR` / `keyring:name`), verify env var is set or keyring entry exists |
| Connection not found | Name not in config | Run `dq connection list` to see available connections |
| Timeout | Query exceeded limit | Increase `--timeout` or optimize the query |
| Unsupported database type | Backend not registered | Use `postgres`, `mysql`, or `sqlite` |

Guidelines:
- Always include `--output json` when parsing results programmatically.
- Use `dq discover` to orient before querying — do not guess table or column names.
- Annotate PII columns immediately when you identify them.
- Prefer `--dry-run` for mutations, then confirm with the user before executing.
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

# 5. Preview mutations
dq postgres -c prod-pg "UPDATE users SET status = 2 WHERE last_login < '2025-01-01'" --dry-run --output json
```
