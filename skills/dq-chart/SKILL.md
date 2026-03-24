---
name: dq-chart
description: Generate interactive HTML charts from dq query results. Covers line, bar, area, scatter, and pie charts with multi-series and group-by support.
---

# Recipe: Generate Charts from Query Results

Use this recipe when the user wants to visualize query results — trends, comparisons, distributions, or relationships. dq generates interactive ECharts HTML files that auto-open in the browser.

## Chart Types

| Type | Best for | Required flags |
|---|---|---|
| `line` | Time-series, trends | `--x`, `--y` |
| `bar` | Comparisons, rankings | `--x`, `--y` |
| `area` | Cumulative trends, stacked data | `--x`, `--y` |
| `scatter` | Correlations, distributions | `--x`, `--y` |
| `pie` | Proportions, market share | `--x` (labels), `--y` (values) |

## Step 1 — Write a query that produces chart-ready data

Good chart queries have:
- A categorical or temporal column for the x-axis
- One or more numeric columns for values
- Optional: a grouping column for multi-series charts

```bash
# Time-series: monthly revenue
dq postgres -c <connection> "SELECT
  TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month,
  SUM(amount) AS revenue
FROM orders
WHERE created_at >= NOW() - INTERVAL '12 months'
GROUP BY 1 ORDER BY 1" -o json

# Grouped: revenue by region per quarter
dq postgres -c <connection> "SELECT
  TO_CHAR(DATE_TRUNC('quarter', created_at), 'YYYY-QQ') AS quarter,
  region,
  SUM(amount) AS revenue
FROM orders
GROUP BY 1, 2 ORDER BY 1" -o json
```

> **Important:** Always use `-o json` when piping to `dq chart`. The chart command reads JSON format.

## Step 2 — Pipe to dq chart

### Basic line chart (single series)

```bash
dq postgres -c <connection> "SELECT month, revenue FROM monthly_stats ORDER BY month" -o json \
  | dq chart --type line --x month --y revenue --title "Monthly Revenue"
```

### Multi-series chart (multiple y columns)

```bash
dq postgres -c <connection> "SELECT month, revenue, cost, profit FROM monthly_stats ORDER BY month" -o json \
  | dq chart --type area --x month --y revenue,cost,profit --title "Financial Overview"
```

### Grouped chart (--group splits data into series)

```bash
dq postgres -c <connection> "SELECT quarter, region, SUM(revenue) AS revenue FROM sales GROUP BY 1, 2 ORDER BY 1" -o json \
  | dq chart --type bar --x quarter --y revenue --group region --title "Revenue by Region"
```

### Pie chart

```bash
dq postgres -c <connection> "SELECT status, COUNT(*) AS count FROM users GROUP BY status" -o json \
  | dq chart --type pie --x status --y count --title "User Status Distribution"
```

### Scatter plot

```bash
dq postgres -c <connection> "SELECT age, salary FROM employees" -o json \
  | dq chart --type scatter --x age --y salary --title "Age vs Salary"
```

## Step 3 — Customize output

| Flag | Purpose | Example |
|---|---|---|
| `--title` | Chart title | `--title "Q4 Revenue"` |
| `--save <path>` | Save to specific file | `--save report.html` |
| `--no-open` | Don't auto-open browser | For scripted/batch use |
| `--from <file>` | Read from file instead of stdin | `--from results.json` |

## Patterns

### Save query results, then chart later

```bash
# Save results
dq postgres -c <connection> "SELECT ..." -o json > results.json

# Chart from saved file
dq chart --type line --x month --y revenue --from results.json
```

### Auto-infer columns

If `--x` and `--y` are not specified, dq uses the first column as x and remaining columns as y:

```bash
dq postgres -c <connection> "SELECT month, revenue FROM stats ORDER BY month" -o json | dq chart
```

### Multiple charts from the same data

```bash
# Save once, chart multiple ways
dq postgres -c <connection> "SELECT month, revenue, users, arpu FROM metrics ORDER BY month" -o json > /tmp/metrics.json

dq chart --type line --x month --y revenue --title "Revenue Trend" --from /tmp/metrics.json --save revenue.html --no-open
dq chart --type bar --x month --y users --title "User Growth" --from /tmp/metrics.json --save users.html --no-open
dq chart --type scatter --x users --y arpu --title "Users vs ARPU" --from /tmp/metrics.json --save arpu.html --no-open
```

## Query Patterns for Common Charts

### Month-over-Month comparison

```sql
SELECT
  TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month,
  SUM(amount) AS revenue,
  LAG(SUM(amount)) OVER (ORDER BY DATE_TRUNC('month', created_at)) AS prev_month,
  ROUND((SUM(amount) - LAG(SUM(amount)) OVER (ORDER BY DATE_TRUNC('month', created_at))) * 100.0
    / NULLIF(LAG(SUM(amount)) OVER (ORDER BY DATE_TRUNC('month', created_at)), 0), 1) AS mom_pct
FROM orders
GROUP BY 1 ORDER BY 1
```

### Cohort retention (chart-ready)

```sql
SELECT
  cohort_month,
  month_number,
  COUNT(DISTINCT user_id) AS users
FROM (
  SELECT
    user_id,
    TO_CHAR(MIN(created_at) OVER (PARTITION BY user_id), 'YYYY-MM') AS cohort_month,
    EXTRACT(MONTH FROM AGE(created_at, MIN(created_at) OVER (PARTITION BY user_id)))::int AS month_number
  FROM events
) sub
GROUP BY 1, 2
ORDER BY 1, 2
```

Then chart with: `| dq chart --type line --x month_number --y users --group cohort_month`

### Distribution histogram (bucket data in SQL)

```sql
SELECT
  WIDTH_BUCKET(amount, 0, 1000, 20) AS bucket,
  COUNT(*) AS count
FROM orders
GROUP BY 1 ORDER BY 1
```
