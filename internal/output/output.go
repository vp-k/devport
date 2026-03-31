package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// isTTY reports whether w is a terminal. Injectable for testing.
var isTTY = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// ANSI colour codes.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorRed    = "\033[31m"
	colorBold   = "\033[1m"
)

// Printer writes output to a writer, applying colour only when the writer is a
// TTY.
type Printer struct {
	w     io.Writer
	color bool
}

// New returns a Printer that writes to w. Colour is enabled only when w is a
// TTY.
func New(w io.Writer) *Printer {
	return &Printer{w: w, color: isTTY(w)}
}

// NewPlain returns a Printer with colour disabled regardless of the writer.
func NewPlain(w io.Writer) *Printer {
	return &Printer{w: w, color: false}
}

// IsColor reports whether colour output is enabled.
func (p *Printer) IsColor() bool { return p.color }

func (p *Printer) colorize(code, s string) string {
	if !p.color {
		return s
	}
	return code + s + colorReset
}

// Printf writes a formatted string.
func (p *Printer) Printf(format string, args ...any) {
	fmt.Fprintf(p.w, format, args...)
}

// Println writes a line.
func (p *Printer) Println(s string) {
	fmt.Fprintln(p.w, s)
}

// Success prints a green success message.
func (p *Printer) Success(msg string) {
	p.Println(p.colorize(colorGreen, msg))
}

// Warn prints a yellow warning message.
func (p *Printer) Warn(msg string) {
	p.Println(p.colorize(colorYellow, msg))
}

// Error prints a red error message.
func (p *Printer) Error(msg string) {
	p.Println(p.colorize(colorRed, msg))
}

// Info prints a cyan info message.
func (p *Printer) Info(msg string) {
	p.Println(p.colorize(colorCyan, msg))
}

// Bold returns s wrapped in bold ANSI codes (no-op when colour is disabled).
func (p *Printer) Bold(s string) string {
	return p.colorize(colorBold, s)
}

// --- Table renderer ---

// Column defines a table column header and its minimum width.
type Column struct {
	Header string
	MinWidth int
}

// Table renders a simple fixed-width text table.
type Table struct {
	p       *Printer
	cols    []Column
	rows    [][]string
}

// NewTable creates a Table with the given column definitions.
func NewTable(p *Printer, cols []Column) *Table {
	return &Table{p: p, cols: cols}
}

// AddRow adds a row. Values beyond the column count are ignored; missing values
// are treated as empty strings.
func (t *Table) AddRow(values ...string) {
	row := make([]string, len(t.cols))
	for i := range row {
		if i < len(values) {
			row[i] = values[i]
		}
	}
	t.rows = append(t.rows, row)
}

// Render writes the table to the printer's writer.
func (t *Table) Render() {
	// Compute column widths: max of header, MinWidth, and all row values.
	widths := make([]int, len(t.cols))
	for i, col := range t.cols {
		w := len(col.Header)
		if col.MinWidth > w {
			w = col.MinWidth
		}
		widths[i] = w
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Header row.
	t.p.Println(t.p.colorize(colorBold, t.renderRow(headerValues(t.cols), widths)))

	// Separator.
	t.p.Println(separator(widths))

	// Data rows.
	for _, row := range t.rows {
		t.p.Println(t.renderRow(row, widths))
	}
}

func (t *Table) renderRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = padRight(cell, widths[i])
	}
	return strings.Join(parts, "  ")
}

func headerValues(cols []Column) []string {
	h := make([]string, len(cols))
	for i, c := range cols {
		h[i] = c.Header
	}
	return h
}

func separator(widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("-", w)
	}
	return strings.Join(parts, "  ")
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
