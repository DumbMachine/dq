# dq

database cli to connect to dbs ( postgres, mysql, sqlite ) for agents. named connections. structured output.
json when piped, tables when human. agents figure out the rest.

## install

```sh
curl -fsSL https://raw.githubusercontent.com/DumbMachine/dq/main/install.sh | sh
```

or build it yourself

```sh
make build
```

## skills

dq ships with [skills](skills/) that teach agents how to use the CLI — from basic queries to multi-step DBA workflows.

```sh
# install all skills at once
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq

# or pick only what you need
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-cold-start
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-data-profiling
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-find-missing-indexes
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-find-table
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-orphan-check
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-query-impact-analysis
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-safe-backfill
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-safe-mutation
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-sample-and-summarize
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-slow-query-investigation
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-table-health-check
npx skills add https://github.com/dumbmachine/dq/tree/main/skills/dq-trace-relationships
```

## connections

you name them once, use them forever. config lives in `~/.config/dq/config.yaml`.

```sh
# postgres
dq connection add prod --type postgres --host db.prod.internal --port 5432 \
  --database myapp --user readonly --password "env:DB_PASS"

# sqlite, because sometimes that's all you need
dq connection add local --type sqlite --path ./app.db

# what do i have
dq connection list

# does it actually work
dq connection test prod

# nuke it
dq connection remove prod
```

passwords take `env:VAR_NAME` so you're not putting secrets in yaml like an animal.

## discover

the whole point. one command, full picture. schemas, tables, columns,
types, pks, fks, row counts, sizes. your agent's pgadmin sidebar.

```sh
dq discover -c prod
```

cached after first call. instant on repeat. `--refresh` to re-introspect.

annotations get merged in automatically — if you previously noted that
`users.email` is PII, discover tells you that without asking.

## queries

you pick the backend. you give it sql. it gives you data.

```sh
dq postgres -c prod "SELECT * FROM users ORDER BY created_at DESC LIMIT 5"
dq mysql -c staging "SELECT COUNT(*) FROM orders"
dq sqlite -c local "SELECT * FROM events WHERE date > '2026-01-01'"
```

every result comes wrapped:

```json
{
  "meta": {"connection": "prod", "row_count": 5, "duration_ms": 12},
  "columns": [{"name": "id", "type": "INT8"}, ...],
  "rows": [{"id": 1, "email": "alice@example.com"}, ...]
}
```

pipe it, parse it, whatever. `| jq .rows` and move on.

### don't blow up your context window

```sh
dq postgres -c prod "SELECT * FROM big_table" --limit 20 --offset 100
dq postgres -c prod "SELECT * FROM users" --fields id,email
```

### don't blow up your database

```sh
dq postgres -c prod "DELETE FROM users WHERE status = 3" --dry-run
```

wraps in `BEGIN`, runs it, reads the result, `ROLLBACK`. you get real
affected_rows and real constraint errors. nothing actually changes.

### explain

```sh
dq postgres -c prod "SELECT * FROM users WHERE email = 'a@b.com'" --explain
```

## schema

live introspection. no caching. for when you need specifics.

```sh
dq schema tables -c prod
dq schema columns -c prod --table users
dq schema indexes -c prod --table users
dq schema constraints -c prod --table users

# all of the above in one shot, plus annotations
dq schema describe -c prod --table users
```

### capabilities

```sh
dq schema capabilities --output json
```

the cli describes itself. every command, every flag, every exit code.
agents call this to figure out what's available without reading docs.

## annotations

agents learn things. "this column is PII." "status=3 means deactivated."
"total is in cents." annotations persist that between conversations.

```sh
dq annotate set -c prod --table users --note "core accounts table"
dq annotate set -c prod --table users --column email --note "PII, do not log"
dq annotate set -c prod --table orders --column total --note "cents, /100 for dollars"

dq annotate get -c prod --table users
dq annotate remove -c prod --table users --column email
```

stored as yaml in `~/.config/dq/annotations/`. merged into discover output.
your agent builds a knowledge base about your data over time without you doing anything.

## output

auto-detects. tty gets tables, pipes get json. override with `--output` or `DQ_OUTPUT` env var.

```
--output json      structured, parseable
--output table     human-readable, aligned
--output csv       for spreadsheet people
--output ndjson    one json object per line, for streaming
```

errors go to stderr. always json. always structured.

```json
{"error":"auth","message":"password env var not set","suggestion":"export DB_PASS=..."}
```

## exit codes

```
0  success
1  error
2  usage (you typed it wrong)
3  not found
4  auth
5  conflict
6  timeout
7  dry-run completed (nothing changed, on purpose)
```

## the workflow

```sh
dq discover -c prod                     # what's in here
dq postgres -c prod "SELECT ..."        # get data
dq annotate set -c prod --table ...     # remember what you learned
dq schema describe -c prod --table ...  # deep dive
dq postgres -c prod "UPDATE ..." --dry-run  # preview changes
```

one call to orient. one call to query. repeat.
