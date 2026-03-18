# dq — Query Patterns

## Basic Query
```bash
dq postgres -c mydb "SELECT * FROM users WHERE id = 1" --output json
```

## With Pagination
```bash
dq postgres -c mydb "SELECT * FROM users ORDER BY created_at DESC" --limit 10 --offset 20
```

## Field Filtering
```bash
dq postgres -c mydb "SELECT * FROM users" --fields id,email,created_at
```

## Dry Run (Preview Mutations)
```bash
dq postgres -c mydb "DELETE FROM users WHERE status = 'inactive'" --dry-run
# Returns affected_rows count without actually deleting
```

## With Timeout
```bash
dq postgres -c mydb "SELECT * FROM large_table" --timeout 60s
```

## EXPLAIN
```bash
dq postgres -c mydb "SELECT * FROM users WHERE email = 'test@example.com'" --explain
```

## Export Results
```bash
# Pipe query output to a file using shell redirection
dq postgres -c mydb "SELECT * FROM users" --output csv > users.csv
dq postgres -c mydb "SELECT * FROM users" --output ndjson > users.ndjson
```
