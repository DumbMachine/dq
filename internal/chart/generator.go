package chart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/dumbmachine/db-cli/pkg/types"
)

// Config holds chart generation parameters.
type Config struct {
	Type     string   // line, bar, area, scatter, pie
	Title    string
	XColumn  string
	YColumns []string
	GroupBy  string
}

// BuildOption builds an ECharts option from query result rows.
func BuildOption(rows []map[string]any, cfg Config) (map[string]any, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data rows to chart")
	}

	var option map[string]any
	var err error

	switch cfg.Type {
	case "line", "bar", "area":
		option, err = buildCategoryChart(rows, cfg)
	case "scatter":
		option, err = buildScatterChart(rows, cfg)
	case "pie":
		option, err = buildPieChart(rows, cfg)
	default:
		return nil, fmt.Errorf("unsupported chart type: %s (supported: line, bar, area, scatter, pie)", cfg.Type)
	}
	if err != nil {
		return nil, err
	}

	applyTitle(option, cfg)
	return option, nil
}

// --- shared helpers ---

func defaultGrid() map[string]any {
	return map[string]any{"left": "8%", "right": "4%", "bottom": "12%", "top": "14%", "containLabel": true}
}

func defaultToolbox() map[string]any {
	return map[string]any{
		"feature": map[string]any{
			"saveAsImage": map[string]any{},
			"dataZoom":    map[string]any{},
			"restore":     map[string]any{},
		},
	}
}

func applyTitle(option map[string]any, cfg Config) {
	if cfg.Title != "" {
		option["title"] = map[string]any{"text": cfg.Title}
	}
}

func decorateSeries(s map[string]any, cfg Config, echartsType string) {
	if cfg.Type == "area" {
		s["areaStyle"] = map[string]any{"opacity": 0.3}
	}
	if echartsType == "line" {
		s["smooth"] = true
	}
}

// --- category charts (line, bar, area) ---

func buildCategoryChart(rows []map[string]any, cfg Config) (map[string]any, error) {
	if cfg.XColumn == "" {
		return nil, fmt.Errorf("--x flag is required for %s charts", cfg.Type)
	}
	if len(cfg.YColumns) == 0 {
		return nil, fmt.Errorf("--y flag is required for %s charts", cfg.Type)
	}

	echartsType := cfg.Type
	if echartsType == "area" {
		echartsType = "line"
	}

	if cfg.GroupBy != "" {
		return buildGroupedCategoryChart(rows, cfg, echartsType)
	}

	labels := make([]any, len(rows))
	for i, row := range rows {
		labels[i] = row[cfg.XColumn]
	}

	series := make([]map[string]any, len(cfg.YColumns))
	for j, yCol := range cfg.YColumns {
		data := make([]any, len(rows))
		for i, row := range rows {
			data[i] = row[yCol]
		}
		s := map[string]any{"name": yCol, "type": echartsType, "data": data}
		decorateSeries(s, cfg, echartsType)
		series[j] = s
	}

	option := map[string]any{
		"tooltip": map[string]any{"trigger": "axis"},
		"grid":    defaultGrid(),
		"xAxis": map[string]any{
			"type":      "category",
			"data":      labels,
			"axisLabel": map[string]any{"rotate": axisLabelRotation(labels)},
		},
		"yAxis":   map[string]any{"type": "value"},
		"series":  series,
		"toolbox": defaultToolbox(),
		"dataZoom": []map[string]any{
			{"type": "inside", "start": 0, "end": 100},
		},
	}

	if len(cfg.YColumns) > 1 {
		names := make([]any, len(cfg.YColumns))
		for i, c := range cfg.YColumns {
			names[i] = c
		}
		option["legend"] = map[string]any{"data": names}
	}

	return option, nil
}

func buildGroupedCategoryChart(rows []map[string]any, cfg Config, echartsType string) (map[string]any, error) {
	xSet := make(map[string]bool)
	groupSet := make(map[string]bool)
	var xOrder, groupOrder []string

	for _, row := range rows {
		x := fmt.Sprint(row[cfg.XColumn])
		g := fmt.Sprint(row[cfg.GroupBy])
		if !xSet[x] {
			xSet[x] = true
			xOrder = append(xOrder, x)
		}
		if !groupSet[g] {
			groupSet[g] = true
			groupOrder = append(groupOrder, g)
		}
	}

	dataMap := make(map[string]map[string]any)
	for _, g := range groupOrder {
		dataMap[g] = make(map[string]any)
	}
	for _, row := range rows {
		x := fmt.Sprint(row[cfg.XColumn])
		g := fmt.Sprint(row[cfg.GroupBy])
		dataMap[g][x] = row[cfg.YColumns[0]]
	}

	series := make([]map[string]any, len(groupOrder))
	for i, g := range groupOrder {
		data := make([]any, len(xOrder))
		for j, x := range xOrder {
			if v, ok := dataMap[g][x]; ok {
				data[j] = v
			} else {
				data[j] = nil
			}
		}
		s := map[string]any{"name": g, "type": echartsType, "data": data}
		decorateSeries(s, cfg, echartsType)
		series[i] = s
	}

	xLabels := make([]any, len(xOrder))
	for i, x := range xOrder {
		xLabels[i] = x
	}

	groupNames := make([]any, len(groupOrder))
	for i, g := range groupOrder {
		groupNames[i] = g
	}

	return map[string]any{
		"tooltip": map[string]any{"trigger": "axis"},
		"legend":  map[string]any{"data": groupNames},
		"grid":    defaultGrid(),
		"xAxis": map[string]any{
			"type":      "category",
			"data":      xLabels,
			"axisLabel": map[string]any{"rotate": axisLabelRotation(xLabels)},
		},
		"yAxis":   map[string]any{"type": "value"},
		"series":  series,
		"toolbox": defaultToolbox(),
		"dataZoom": []map[string]any{
			{"type": "inside", "start": 0, "end": 100},
		},
	}, nil
}

// --- scatter charts ---

func buildScatterChart(rows []map[string]any, cfg Config) (map[string]any, error) {
	if cfg.XColumn == "" || len(cfg.YColumns) == 0 {
		return nil, fmt.Errorf("--x and --y flags are required for scatter charts")
	}

	yCol := cfg.YColumns[0]

	if cfg.GroupBy != "" {
		return buildGroupedScatterChart(rows, cfg, yCol)
	}

	data := make([][]any, len(rows))
	for i, row := range rows {
		data[i] = []any{row[cfg.XColumn], row[yCol]}
	}

	return map[string]any{
		"tooltip": map[string]any{"trigger": "item"},
		"grid":    defaultGrid(),
		"xAxis":   map[string]any{"type": "value", "name": cfg.XColumn},
		"yAxis":   map[string]any{"type": "value", "name": yCol},
		"series": []map[string]any{
			{"type": "scatter", "data": data, "symbolSize": 8},
		},
		"toolbox": defaultToolbox(),
	}, nil
}

func buildGroupedScatterChart(rows []map[string]any, cfg Config, yCol string) (map[string]any, error) {
	groups := make(map[string][][]any)
	var groupOrder []string
	seen := make(map[string]bool)

	for _, row := range rows {
		g := fmt.Sprint(row[cfg.GroupBy])
		if !seen[g] {
			seen[g] = true
			groupOrder = append(groupOrder, g)
		}
		groups[g] = append(groups[g], []any{row[cfg.XColumn], row[yCol]})
	}

	series := make([]map[string]any, len(groupOrder))
	groupNames := make([]any, len(groupOrder))
	for i, g := range groupOrder {
		groupNames[i] = g
		series[i] = map[string]any{
			"name":       g,
			"type":       "scatter",
			"data":       groups[g],
			"symbolSize": 8,
		}
	}

	return map[string]any{
		"tooltip": map[string]any{"trigger": "item"},
		"legend":  map[string]any{"data": groupNames},
		"grid":    defaultGrid(),
		"xAxis":   map[string]any{"type": "value", "name": cfg.XColumn},
		"yAxis":   map[string]any{"type": "value", "name": yCol},
		"series":  series,
		"toolbox": defaultToolbox(),
	}, nil
}

// --- pie chart ---

func buildPieChart(rows []map[string]any, cfg Config) (map[string]any, error) {
	if cfg.XColumn == "" || len(cfg.YColumns) == 0 {
		return nil, fmt.Errorf("--x (labels) and --y (values) flags are required for pie charts")
	}

	yCol := cfg.YColumns[0]
	data := make([]map[string]any, len(rows))
	for i, row := range rows {
		data[i] = map[string]any{
			"name":  fmt.Sprint(row[cfg.XColumn]),
			"value": row[yCol],
		}
	}

	return map[string]any{
		"tooltip": map[string]any{"trigger": "item", "formatter": "{b}: {c} ({d}%)"},
		"series": []map[string]any{
			{
				"type":   "pie",
				"radius": []string{"40%", "70%"},
				"data":   data,
				"label":  map[string]any{"show": true, "formatter": "{b}: {d}%"},
				"emphasis": map[string]any{
					"itemStyle": map[string]any{
						"shadowBlur":    10,
						"shadowOffsetX": 0,
						"shadowColor":   "rgba(0, 0, 0, 0.5)",
					},
				},
			},
		},
		"toolbox": map[string]any{
			"feature": map[string]any{"saveAsImage": map[string]any{}},
		},
	}, nil
}

// --- public utilities ---

// OptionJSON marshals the ECharts option to a JSON string.
func OptionJSON(option map[string]any) (string, error) {
	b, err := json.Marshal(option)
	if err != nil {
		return "", fmt.Errorf("marshaling chart option: %w", err)
	}
	return string(b), nil
}

// ParseInput parses chart input from JSON. Accepts QueryResult format or a plain JSON array.
func ParseInput(data []byte) ([]map[string]any, error) {
	data = bytes.TrimSpace(data)

	var qr types.QueryResult
	if err := json.Unmarshal(data, &qr); err == nil && len(qr.Rows) > 0 {
		return qr.Rows, nil
	}

	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}

	return nil, fmt.Errorf("input must be dq query output (JSON) or a JSON array of objects")
}

// InferColumns returns sorted column names from the first row.
func InferColumns(rows []map[string]any) []string {
	if len(rows) == 0 {
		return nil
	}
	cols := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

// axisLabelRotation returns a rotation angle for x-axis labels based on label count.
func axisLabelRotation(labels []any) int {
	if len(labels) > 20 {
		return 45
	}
	return 0
}
