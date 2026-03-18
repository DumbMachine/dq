# dq — Agent Orientation

## What is dq?
`dq` is an agent-first database CLI. It lets you discover database structure, run queries, and manage annotations.

## Quick Start
1. **Add a connection**: `dq connection add mydb --type postgres --host localhost --port 5432 --database myapp --user admin --password env:DB_PASS`
2. **Discover the database**: `dq discover -c mydb --output json` — returns full schema hierarchy (cached)
3. **Query**: `dq postgres -c mydb "SELECT * FROM users LIMIT 5" --output json`
4. **Annotate**: `dq annotate set -c mydb --table users --column email --note "PII - do not expose"`

## Key Commands
| Command | Purpose |
|---------|---------|
| `dq discover -c <name>` | Full DB overview — schemas, tables, columns, FKs, row counts |
| `dq postgres/mysql/sqlite -c <name> <sql>` | Execute SQL |
| `dq schema describe -c <name> --table <t>` | Detailed table info with annotations |
| `dq schema capabilities` | Runtime CLI self-introspection |
| `dq annotate set/get/remove` | Persist knowledge about data |

## Output
- JSON when piped, table when TTY
- Override with `--output json|table|csv|ndjson`
- Filter columns with `--fields col1,col2`
- Paginate with `--limit N --offset N`

## Dry Run
Add `--dry-run` to any mutation to preview without changing data (uses transaction rollback).
