package asc

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// RenderTable writes a bordered Unicode table to stdout.
// Headers preserve their original casing and are center-aligned.
// Data rows are left-aligned for readability.
func RenderTable(headers []string, rows [][]string) {
	safeHeaders, safeRows := sanitizeHumanTableData(headers, rows)
	table := tablewriter.NewTable(
		os.Stdout,
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{
					AutoFormat: tw.Off,
				},
				Alignment: tw.CellAlignment{Global: tw.AlignCenter},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)
	table.Header(safeHeaders)
	_ = table.Bulk(safeRows)
	_ = table.Render()
}

// RenderMarkdown writes a Markdown-formatted table to stdout.
// Headers preserve their original casing. Data rows are left-aligned.
// Pipe characters in cell values are escaped automatically by the renderer.
func RenderMarkdown(headers []string, rows [][]string) {
	safeHeaders, safeRows := sanitizeHumanTableData(headers, rows)
	table := tablewriter.NewTable(
		os.Stdout,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{
					AutoFormat: tw.Off,
				},
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)
	table.Header(safeHeaders)
	_ = table.Bulk(safeRows)
	_ = table.Render()
}

func sanitizeHumanTableData(headers []string, rows [][]string) ([]string, [][]string) {
	safeHeaders := make([]string, len(headers))
	for i, header := range headers {
		safeHeaders[i] = SanitizeTerminalText(header)
	}

	safeRows := make([][]string, len(rows))
	for i, row := range rows {
		safeRows[i] = make([]string, len(row))
		for j, cell := range row {
			safeRows[i][j] = SanitizeTerminalText(cell)
		}
	}
	return safeHeaders, safeRows
}
