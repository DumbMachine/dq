# dq — Troubleshooting Workflows

## Connection Issues
```bash
# Test connectivity
dq connection test prod --output json

# Show connection config
dq connection show prod --output json

# Re-add with correct credentials
dq connection remove prod
dq connection add prod --type postgres --host ... --password "env:DB_PASS"
```

## Investigate Database State via SQL
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

## Schema Investigation
```bash
# Orient yourself
dq discover -c prod --output json

# Deep dive into a specific table
dq schema describe -c prod --table orders --output json

# Refresh cached discovery
dq discover -c prod --refresh --output json
```

## Annotate Findings
```bash
# Persist what you learn for future sessions
dq annotate set -c prod --table orders --note "Needs index on created_at, queries slow"
dq annotate set -c prod --table users --column status --note "1=active, 2=suspended, 3=deactivated"
```
