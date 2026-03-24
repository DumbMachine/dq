---
name: dq-playbook
description: Create, manage, and use playbooks — reusable analytics workflows, org knowledge, and standardized procedures that persist across sessions.
---

# Recipe: Playbooks — Reusable Analytics Workflows & Org Knowledge

Use this recipe when you need to create, manage, or follow playbooks. Playbooks are reusable templates that encode analytics workflows, organizational knowledge, business logic, data quirks, and standardized procedures. They persist across sessions and teach agents how to analyze data the way your org expects.

## When to Use Playbooks

| Use case | Example |
|---|---|
| Repeated analytics workflow | "Run our monthly revenue report" |
| Org-specific business logic | "How we calculate churn" |
| Data quirks and gotchas | "Orders table has duplicate rows before 2024-01" |
| Standardized procedures | "How to investigate a spike in error rates" |
| Onboarding knowledge | "Key tables and what they mean" |

## Playbook Format

Playbooks are markdown files with YAML frontmatter:

```markdown
---
name: monthly-revenue
description: Monthly revenue analysis with MoM trends and segment breakdown
tags: [revenue, monthly, finance]
connections: [prod-pg]
---

# Monthly Revenue Analysis

## Overview
Analyze monthly revenue trends, calculate MoM changes, and break down by segment.

## Procedure
1. Query monthly revenue aggregations
2. Calculate MoM percentage changes
3. Break down by customer segment
4. Generate line chart for trends
5. Generate pie chart for segment distribution
6. Summarize findings with key numbers

## SQL Templates
...

## Specifications
- Revenue is stored in cents — divide by 100 for display
- Use UTC timestamps
- Exclude test accounts (account_type = 'test')

## Advice
- Always check for duplicate orders before aggregating
- Weekend revenue is typically 40% lower — don't flag as anomaly

## Forbidden Actions
- Do not expose individual customer revenue
- Do not modify the orders table
```

## Step 1 — Create a Playbook

### Generate a template and edit it

```bash
dq playbook init monthly-revenue
# Edit monthly-revenue.md with your workflow
dq playbook add monthly-revenue --file monthly-revenue.md
```

### Create from an existing analysis

After completing a successful analysis, capture the workflow:

1. Write the steps you followed as a playbook markdown file
2. Include the SQL queries that worked
3. Note any data quirks or gotchas discovered
4. Add to dq:

```bash
dq playbook add churn-analysis --file churn-analysis.md
```

## Step 2 — List and Find Playbooks

```bash
# List all playbooks
dq playbook list

# Filter by tag
dq playbook list --tag revenue

# JSON output for programmatic use
dq playbook list -o json
```

## Step 3 — Use a Playbook

When starting an analysis, check for relevant playbooks:

```bash
# Show the full playbook content
dq playbook show monthly-revenue

# JSON output includes metadata + content
dq playbook show monthly-revenue -o json
```

Then follow the procedure and SQL templates in the playbook. The playbook's **Specifications** and **Advice** sections encode org knowledge that prevents common mistakes.

## Step 4 — Update a Playbook

To update a playbook with new learnings:

1. Export or edit the playbook file
2. Re-add with the same name (overwrites):

```bash
dq playbook add monthly-revenue --file updated-playbook.md
```

## Step 5 — Remove a Playbook

```bash
dq playbook remove monthly-revenue
```

## Playbook Design Guidelines

### Good playbooks are:

- **Outcome-focused**: Start with what question the playbook answers
- **Step-by-step**: Numbered procedures with one action per step
- **SQL-rich**: Include tested query templates with placeholders
- **Org-aware**: Capture business logic, data quirks, and gotchas
- **Chart-aware**: Suggest visualization types for results

### Frontmatter fields

| Field | Required | Purpose |
|---|---|---|
| `name` | Yes | Unique identifier (used as filename) |
| `description` | Yes | One-line summary — shown in `playbook list` |
| `tags` | No | Categorization for filtering |
| `connections` | No | Which connections this playbook applies to |

### Recommended sections

| Section | Purpose |
|---|---|
| **Overview** | Goal and scope |
| **Procedure** | Step-by-step instructions |
| **SQL Templates** | Reusable queries with placeholders |
| **Specifications** | Output format, units, filters, postconditions |
| **Advice** | Tips, gotchas, known data quirks |
| **Forbidden Actions** | What to never do (safety rails) |

## Example Playbooks

### Churn Investigation

```bash
dq playbook init churn-investigation
```

Then fill in:
- SQL to identify churned users (no activity in 30 days)
- Cohort analysis query by signup month
- Chart: line chart of churn rate over time
- Chart: bar chart of churn by acquisition channel
- Advice: exclude free-tier users, use `last_active_at` not `last_login_at`

### Data Quality Audit

```bash
dq playbook init data-quality-audit
```

Then fill in:
- Use `dq-data-profiling` skill to profile each key table
- SQL to check referential integrity
- SQL to find duplicate records
- SQL to check for orphaned foreign keys
- Specifications: flag any column with >20% null rate
- Chart: bar chart of null rates by column

### Incident Response (Error Spike)

```bash
dq playbook init incident-error-spike
```

Then fill in:
- SQL to count errors by type in the last hour
- SQL to identify affected users
- SQL to check if a deploy correlates with the spike
- Chart: line chart of error rate over last 24h with 1-min granularity
- Advice: check `deployments` table for recent deploys within 30 min of spike onset
