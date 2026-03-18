---
name: recipe-sample-and-summarize
description: Get representative rows from a table and summarize column distributions, flag anomalies, and present findings. Use when the user wants to understand the shape and quality of data in a table.
---

# Recipe: Sample and Summarize a Table

Use this recipe when the user asks to explore, profile, or understand the data in a table. The goal is to quickly build a picture of what the data looks like without reading every row.

## Prerequisites

- A working connection (verify with `dq connection list --output json`)
- The table name (use `dq discover -c <connection> --output json` if unknown)

## Steps

### 1. Get table metadata: row count and column info

Start with the row count so you can decide how to sample, and column info so you can classify columns by type.

```bash
dq schema describe -c <connection> --table <table> --output json
```

This returns columns with types, indexes, constraints, and annotations. Parse the column list and classify each column:

- **Categorical**: `varchar`, `text`, `char`, `enum`, `boolean`
- **Numeric**: `int`, `bigint`, `smallint`, `numeric`, `decimal`, `float`, `double`, `real`
- **Timestamp**: `timestamp`, `timestamptz`, `date`, `datetime`
- **Other**: `json`, `bytea`, `uuid`, `array` (skip these for distribution analysis)

Get the exact row count:

```bash
dq postgres -c <connection> "SELECT COUNT(*) AS total_rows FROM <table>" --output json
```

**Decision point**: If `total_rows` is 0, stop and tell the user the table is empty. If `total_rows` < 100, you can read the entire table instead of sampling.

### 2. Sample representative rows

For small tables (< 1,000 rows), fetch everything:

```bash
dq postgres -c <connection> "SELECT * FROM <table> LIMIT 1000" --output json --fields <key_columns>
```

For large tables, use efficient sampling:

**PostgreSQL** (preferred -- fast, avoids full scan):

```bash
dq postgres -c <connection> "SELECT * FROM <table> TABLESAMPLE BERNOULLI(1) LIMIT 100" --output json
```

If `TABLESAMPLE` returns too few rows (table is small relative to the percentage), increase the percentage or fall back to `ORDER BY RANDOM()`.

**MySQL / SQLite** (no TABLESAMPLE support):

```bash
dq mysql -c <connection> "SELECT * FROM <table> ORDER BY RAND() LIMIT 100" --output json
```

```bash
dq sqlite -c <connection> "SELECT * FROM <table> ORDER BY RANDOM() LIMIT 100" --output json
```

Review the sample rows to get an initial feel for the data. Note any columns that are entirely NULL or have suspicious patterns.

### 3. Analyze categorical columns: value distributions

For each categorical column, get the top values by frequency. This reveals enums, status fields, and data quality issues.

```bash
dq postgres -c <connection> "SELECT <column> AS value, COUNT(*) AS count, ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) AS pct FROM <table> GROUP BY <column> ORDER BY count DESC LIMIT 20" --output json
```

Run this for each categorical column. If there are many categorical columns, combine them into a single query per column to minimize round trips.

**What to look for**:
- Single-value columns (100% one value) -- these are effectively constants
- Highly skewed distributions (99% one value) -- potential default values
- Unexpected NULL counts -- may indicate data quality issues
- Cardinality: if a "categorical" column has nearly as many distinct values as rows, it may actually be a unique identifier

### 4. Analyze numeric columns: statistical summary

For each numeric column, compute descriptive statistics:

```bash
dq postgres -c <connection> "SELECT COUNT(<column>) AS non_null_count, COUNT(*) - COUNT(<column>) AS null_count, MIN(<column>) AS min, MAX(<column>) AS max, ROUND(AVG(<column>)::numeric, 2) AS avg, ROUND(STDDEV(<column>)::numeric, 2) AS stddev, PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY <column>) AS p25, PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY <column>) AS median, PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY <column>) AS p75, PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY <column>) AS p99 FROM <table>" --output json
```

**MySQL equivalent** (no PERCENTILE_CONT):

```bash
dq mysql -c <connection> "SELECT COUNT(<column>) AS non_null_count, COUNT(*) - COUNT(<column>) AS null_count, MIN(<column>) AS min_val, MAX(<column>) AS max_val, ROUND(AVG(<column>), 2) AS avg_val, ROUND(STDDEV(<column>), 2) AS stddev FROM <table>" --output json
```

**SQLite equivalent** (no STDDEV or percentiles natively):

```bash
dq sqlite -c <connection> "SELECT COUNT(<column>) AS non_null_count, COUNT(*) - COUNT(<column>) AS null_count, MIN(<column>) AS min_val, MAX(<column>) AS max_val, ROUND(AVG(<column>), 2) AS avg_val FROM <table>" --output json
```

**What to look for**:
- Large gap between p99 and max suggests outliers
- Stddev much larger than the mean suggests high variance or outliers
- Min of 0 or negative values in columns that should be positive (e.g., prices, ages)
- All NULLs in a non-nullable-looking column

### 5. Analyze timestamp columns: range and distribution

For each timestamp column, get the time range and distribution by bucket:

```bash
dq postgres -c <connection> "SELECT MIN(<column>) AS earliest, MAX(<column>) AS latest, MAX(<column>) - MIN(<column>) AS span, COUNT(*) AS total_rows, COUNT(<column>) AS non_null_count FROM <table>" --output json
```

Then bucket by an appropriate interval. Choose the bucket size based on the span:
- Span < 7 days: bucket by hour
- Span < 90 days: bucket by day
- Span < 2 years: bucket by month
- Span >= 2 years: bucket by quarter or year

```bash
# Example: bucket by month
dq postgres -c <connection> "SELECT DATE_TRUNC('month', <column>) AS month, COUNT(*) AS count FROM <table> WHERE <column> IS NOT NULL GROUP BY month ORDER BY month" --output json
```

**MySQL equivalent**:

```bash
dq mysql -c <connection> "SELECT DATE_FORMAT(<column>, '%Y-%m') AS month, COUNT(*) AS count FROM <table> WHERE <column> IS NOT NULL GROUP BY month ORDER BY month" --output json
```

**What to look for**:
- Gaps in the time series (missing months, sudden drops)
- Spikes that might indicate bulk imports or anomalies
- Future dates that should not exist
- Dates far in the past (e.g., 1970-01-01 suggesting epoch zero defaults)

### 6. Flag anomalies

Compile a list of anomalies found in steps 3-5. Common anomalies to flag:

| Anomaly | How to detect | Severity |
|---------|---------------|----------|
| Unexpected NULLs | Non-zero null_count in columns that appear required | Medium |
| Single-value columns | Only one distinct value across all rows | Low (may be intentional) |
| Outliers | max >> p99, or values outside 3 stddevs from mean | Medium |
| Future timestamps | max > current date for historical data | High |
| Epoch zero dates | min = '1970-01-01' | Medium |
| Empty strings vs NULLs | Both '' and NULL present in same column | Low |
| Negative values | min < 0 in columns like price, age, quantity | High |
| Cardinality mismatch | "status" column with 10,000 distinct values | Medium |

To check for empty strings vs NULLs:

```bash
dq postgres -c <connection> "SELECT COUNT(*) FILTER (WHERE <column> IS NULL) AS nulls, COUNT(*) FILTER (WHERE <column> = '') AS empty_strings, COUNT(*) FILTER (WHERE <column> IS NOT NULL AND <column> != '') AS has_value FROM <table>" --output json
```

### 7. Summarize findings to the user

Present a structured summary:

1. **Table overview**: row count, column count, table size if available
2. **Column-by-column findings**:
   - Categorical: top values, cardinality, null rate
   - Numeric: range, central tendency, outliers
   - Timestamp: range, distribution shape, gaps
3. **Anomalies**: list each anomaly with severity and the specific values found
4. **Recommendations**: suggest next steps (e.g., "investigate the 47 rows with negative prices", "the `type` column has only one value -- consider removing it")

Annotate important findings for future reference:

```bash
dq annotate set -c <connection> --table <table> --note "Profiled on <date>. <row_count> rows. Key findings: <summary>"
dq annotate set -c <connection> --table <table> --column <column> --note "<finding>"
```
