package cmd

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dumbmachine/db-cli/internal/chart"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	chartType   string
	chartTitle  string
	chartX      string
	chartY      string
	chartGroup  string
	chartFrom   string
	chartSave   string
	chartNoOpen bool
)

var chartCmd = &cobra.Command{
	Use:   "chart",
	Short: "Generate interactive HTML charts from query results",
	Long: `Generate interactive ECharts visualizations from dq query output.

Pipe from a query:
  dq postgres -c mydb "SELECT month, revenue FROM stats" -o json | dq chart --type line --x month --y revenue

Read from a file:
  dq chart --type bar --x category --y count --from results.json

Multiple series:
  ... | dq chart --type line --x month --y revenue,cost,profit

Group by column:
  ... | dq chart --type bar --x month --y revenue --group region

Chart types: line, bar, area, scatter, pie`,
	RunE: runChart,
}

func init() {
	chartCmd.Flags().StringVar(&chartType, "type", "line", "Chart type: line, bar, area, scatter, pie")
	chartCmd.Flags().StringVar(&chartTitle, "title", "", "Chart title")
	chartCmd.Flags().StringVar(&chartX, "x", "", "Column for x-axis (or labels for pie)")
	chartCmd.Flags().StringVar(&chartY, "y", "", "Column(s) for y-axis, comma-separated for multiple series")
	chartCmd.Flags().StringVar(&chartGroup, "group", "", "Column to group by (creates multiple series)")
	chartCmd.Flags().StringVar(&chartFrom, "from", "", "Read input from JSON file instead of stdin")
	chartCmd.Flags().StringVar(&chartSave, "save", "", "Save chart to file path (default: temp file)")
	chartCmd.Flags().BoolVar(&chartNoOpen, "no-open", false, "Don't auto-open chart in browser")
}

func runChart(cmd *cobra.Command, args []string) error {
	data, err := readChartInput()
	if err != nil {
		return err
	}

	rows, err := chart.ParseInput(data)
	if err != nil {
		return err
	}

	cfg := buildChartConfig(rows)

	option, err := chart.BuildOption(rows, cfg)
	if err != nil {
		return err
	}

	optionJSON, err := chart.OptionJSON(option)
	if err != nil {
		return err
	}

	outPath, err := writeChartHTML(cfg, optionJSON, len(rows))
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("output")
	if format == "" {
		format = output.DefaultFormat()
	}

	result := map[string]any{
		"status": "generated",
		"file":   outPath,
		"type":   chartType,
		"rows":   len(rows),
	}
	if err := output.Print(format, result); err != nil {
		return err
	}

	if !chartNoOpen {
		return chart.OpenBrowser(outPath)
	}
	return nil
}

func readChartInput() ([]byte, error) {
	if chartFrom != "" {
		data, err := os.ReadFile(chartFrom)
		if err != nil {
			return nil, fmt.Errorf("reading input file: %w", err)
		}
		return data, nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, fmt.Errorf("no input: pipe query output or use --from <file>\n\nExample:\n  dq postgres -c mydb \"SELECT x, y FROM t\" -o json | dq chart --type line --x x --y y")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	return data, nil
}

func buildChartConfig(rows []map[string]any) chart.Config {
	var yColumns []string
	if chartY != "" {
		for _, col := range strings.Split(chartY, ",") {
			yColumns = append(yColumns, strings.TrimSpace(col))
		}
	}

	cfg := chart.Config{
		Type:     chartType,
		Title:    chartTitle,
		XColumn:  chartX,
		YColumns: yColumns,
		GroupBy:  chartGroup,
	}

	// Auto-infer columns if not specified
	if cfg.XColumn == "" || len(cfg.YColumns) == 0 {
		cols := chart.InferColumns(rows)
		if cfg.XColumn == "" && len(cols) >= 1 {
			cfg.XColumn = cols[0]
		}
		if len(cfg.YColumns) == 0 && len(cols) >= 2 {
			cfg.YColumns = cols[1:]
		}
	}

	return cfg
}

func writeChartHTML(cfg chart.Config, optionJSON string, rowCount int) (string, error) {
	outPath := chartSave
	if outPath == "" {
		outPath = filepath.Join(os.TempDir(), "dq-chart.html")
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("creating chart file: %w", err)
	}
	defer f.Close()

	title := cfg.Title
	if title == "" {
		title = "dq chart"
	}

	err = chart.Render(f, chart.TemplateData{
		Title:      title,
		ChartType:  cfg.Type,
		RowCount:   rowCount,
		OptionJSON: template.JS(optionJSON),
	})
	if err != nil {
		return "", err
	}

	return outPath, nil
}
