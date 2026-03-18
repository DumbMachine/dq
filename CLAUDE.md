# dq — Agent-First Database CLI

## Philosophy

dq is a **safe, thin wrapper over gorm** that gives agents a generic CLI interface for query execution. It is not a DBA tool, not an ORM, not a migration framework.

### Guiding Principles

1. **Thin wrapper, not a feature factory.** If something can be done by the agent running SQL through the query interface, don't build a dedicated command for it. Admin operations (vacuum, kill queries, table sizes, stats) are just SQL — agents can compose those themselves. The CLI provides: connect, query, get structured results back.

2. **Teach workflows through recipes, not commands.** Instead of encoding DBA logic into CLI commands, encode it into skill files (`skills/`). Recipes teach agents multi-step workflows using the small set of core commands. This keeps the binary simple and the knowledge extensible.

3. **Agent-first design (Agent DX > Human DX).** Follow the principles from [Rewrite Your CLI for AI Agents](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/): machine-readable output, schema introspection, context window discipline, input hardening, safety rails, agent knowledge packaging. Optimize for predictability and defense-in-depth, not discoverability and forgiveness.

4. **Don't reinvent the wheel.** Use existing, correct solutions. Prepared statements over hand-rolled quoting. GORM's built-in quoting over custom identifier escapers. Standard library over bespoke helpers. If you find yourself writing utility code, check if the dependency already handles it.

5. **Annotations are agent memory.** Annotations persist knowledge across sessions (PII columns, business logic notes, data quirks). They merge into discover output automatically. This is what makes dq agent-*first* rather than agent-*tolerant* — the agent builds a knowledge base about your data over time.

6. **Discover solves cold-start.** One call returns the full schema hierarchy with cached results. An agent doesn't need to fumble through `SHOW TABLES` → `SHOW COLUMNS` → repeat. Discover + annotations = instant orientation.

7. **Safety by default.** `--dry-run` wraps mutations in a transaction and rolls back. `--explain` shows the query plan without executing. `--limit` and `--fields` protect the context window. The agent is not a trusted operator.

8. **When you change something, change everything that references it.** Removing a feature means updating: Go code, driver interface, root.go imports, capabilities command, all skill files, README, and CLAUDE.md. No stale references. Ever.

## Build & Test

```bash
make build          # builds ./dq binary
go build ./...      # check compilation
go test ./...       # run tests
```

## Project Structure

- `cmd/` — Cobra command definitions (root, version, discover, connection/, query/, schema/, annotate/)
- `internal/` — Private packages (config, database drivers, cache, annotations, output, query executor, validation)
- `pkg/types/` — Shared type definitions
- `skills/` — Agent skill files and recipes (see `skills/SKILLS.md` for the index)

## What Belongs in the CLI vs. in a Recipe

| Belongs in CLI | Belongs in a recipe |
|----------------|---------------------|
| Connection management (add/list/test/show/remove) | DBA investigation workflows |
| Query execution (postgres/mysql/sqlite) | Data profiling |
| Schema introspection (tables/columns/indexes/constraints/describe) | Query impact analysis |
| Discover (full schema + cache + annotations) | Safe mutation patterns |
| Annotations (set/get/remove) | Missing index detection |
| Capabilities (self-introspection) | Orphan FK checks |
| Output formatting (json/table/csv/ndjson) | Backfill strategies |

If you're tempted to add a new command, ask: "Can the agent do this by running SQL through `dq postgres`?" If yes, write a recipe instead.

## Key Patterns

- **Output**: All commands use `output.Print(format, data)`. Auto-detect TTY→table, pipe→json.
- **Drivers**: Implement `database.Driver` interface (Connect + Type + introspection methods), register in `init()` via `database.Register()`.
- **Global flags**: Defined in `cmd/root.go`, accessed via `cmd.Flags().GetX()` in subcommands.
- **Connections**: Stored in `~/.config/dq/config.yaml` via `internal/config/`.
- **Cache**: `~/.config/dq/cache/<connection>/discover.json`
- **Annotations**: `~/.config/dq/annotations/<connection>.yaml`

## Dependencies

- cobra (CLI), gorm (database), tablewriter (tables), isatty (TTY detection), yaml.v3 (annotations)

## Exit Codes

0=success, 1=error, 2=usage, 3=not found, 4=auth, 5=conflict, 6=timeout, 7=dry-run-ok
