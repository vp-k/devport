package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// ---- TTY detection ----

func TestNewPlainDisablesColor(t *testing.T) {
	p := NewPlain(&bytes.Buffer{})
	if p.IsColor() {
		t.Error("NewPlain should have color disabled")
	}
}

func TestNewWithBufferDisablesColor(t *testing.T) {
	// bytes.Buffer is not a TTY.
	p := New(&bytes.Buffer{})
	if p.IsColor() {
		t.Error("New with non-TTY writer should have color disabled")
	}
}

func TestNewWithFileEnablesColor(t *testing.T) {
	// Inject isTTY to return true for any writer.
	orig := isTTY
	isTTY = func(_ io.Writer) bool { return true }
	t.Cleanup(func() { isTTY = orig })

	p := New(os.Stdout)
	if !p.IsColor() {
		t.Error("expected color enabled when isTTY returns true")
	}
}

// ---- Plain text output ----

func TestPrintf(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	p.Printf("hello %s", "world")
	if buf.String() != "hello world" {
		t.Errorf("Printf = %q", buf.String())
	}
}

func TestPrintln(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	p.Println("hello")
	if buf.String() != "hello\n" {
		t.Errorf("Println = %q", buf.String())
	}
}

// ---- Colour methods (plain mode — no ANSI codes) ----

func TestSuccessPlain(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	p.Success("ok")
	if buf.String() != "ok\n" {
		t.Errorf("Success plain = %q", buf.String())
	}
}

func TestWarnPlain(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	p.Warn("careful")
	if buf.String() != "careful\n" {
		t.Errorf("Warn plain = %q", buf.String())
	}
}

func TestErrorPlain(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	p.Error("oops")
	if buf.String() != "oops\n" {
		t.Errorf("Error plain = %q", buf.String())
	}
}

func TestInfoPlain(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	p.Info("note")
	if buf.String() != "note\n" {
		t.Errorf("Info plain = %q", buf.String())
	}
}

func TestBoldPlain(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	result := p.Bold("text")
	if result != "text" {
		t.Errorf("Bold plain = %q, want %q", result, "text")
	}
}

// ---- Colour methods (colour mode — ANSI codes present) ----

func colorPrinter(buf *bytes.Buffer) *Printer {
	return &Printer{w: buf, color: true}
}

func TestSuccessColor(t *testing.T) {
	var buf bytes.Buffer
	p := colorPrinter(&buf)
	p.Success("ok")
	s := buf.String()
	if !strings.Contains(s, colorGreen) {
		t.Errorf("Success color missing green code, got %q", s)
	}
	if !strings.Contains(s, "ok") {
		t.Errorf("Success color missing message, got %q", s)
	}
}

func TestWarnColor(t *testing.T) {
	var buf bytes.Buffer
	p := colorPrinter(&buf)
	p.Warn("careful")
	s := buf.String()
	if !strings.Contains(s, colorYellow) {
		t.Errorf("Warn color missing yellow code, got %q", s)
	}
}

func TestErrorColor(t *testing.T) {
	var buf bytes.Buffer
	p := colorPrinter(&buf)
	p.Error("oops")
	s := buf.String()
	if !strings.Contains(s, colorRed) {
		t.Errorf("Error color missing red code, got %q", s)
	}
}

func TestInfoColor(t *testing.T) {
	var buf bytes.Buffer
	p := colorPrinter(&buf)
	p.Info("note")
	s := buf.String()
	if !strings.Contains(s, colorCyan) {
		t.Errorf("Info color missing cyan code, got %q", s)
	}
}

func TestBoldColor(t *testing.T) {
	var buf bytes.Buffer
	p := colorPrinter(&buf)
	result := p.Bold("text")
	if !strings.Contains(result, colorBold) {
		t.Errorf("Bold color missing bold code, got %q", result)
	}
	if !strings.Contains(result, "text") {
		t.Errorf("Bold color missing message, got %q", result)
	}
}

// ---- Table renderer ----

func TestTableBasic(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	tbl := NewTable(p, []Column{
		{Header: "NAME"},
		{Header: "PORT"},
	})
	tbl.AddRow("my-app", "3001")
	tbl.AddRow("api", "4000")
	tbl.Render()

	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Error("missing NAME header")
	}
	if !strings.Contains(out, "PORT") {
		t.Error("missing PORT header")
	}
	if !strings.Contains(out, "my-app") {
		t.Error("missing my-app row")
	}
	if !strings.Contains(out, "3001") {
		t.Error("missing 3001 value")
	}
	if !strings.Contains(out, "---") {
		t.Error("missing separator line")
	}
}

func TestTableMinWidth(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	tbl := NewTable(p, []Column{
		{Header: "A", MinWidth: 20},
	})
	tbl.AddRow("x")
	tbl.Render()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// Header line should be at least 20 chars wide.
	if len(lines[0]) < 20 {
		t.Errorf("header line too short: %q", lines[0])
	}
}

func TestTableExtraValuesTruncated(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	tbl := NewTable(p, []Column{{Header: "A"}})
	// Pass more values than columns — should not panic.
	tbl.AddRow("a", "b", "c")
	tbl.Render()
}

func TestTableMissingValues(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	tbl := NewTable(p, []Column{{Header: "A"}, {Header: "B"}})
	// Pass fewer values than columns — missing ones become empty.
	tbl.AddRow("only-one")
	tbl.Render()

	out := buf.String()
	if !strings.Contains(out, "only-one") {
		t.Error("missing only-one value")
	}
}

func TestTableEmptyRows(t *testing.T) {
	var buf bytes.Buffer
	p := NewPlain(&buf)
	tbl := NewTable(p, []Column{{Header: "NAME"}, {Header: "PORT"}})
	tbl.Render() // no rows — should not panic
	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Error("missing header even with no rows")
	}
}

func TestTableColorHeader(t *testing.T) {
	var buf bytes.Buffer
	p := colorPrinter(&buf)
	tbl := NewTable(p, []Column{{Header: "NAME"}})
	tbl.AddRow("app")
	tbl.Render()
	out := buf.String()
	if !strings.Contains(out, colorBold) {
		t.Error("expected bold header in color mode")
	}
}

// ---- isTTY with real *os.File ----

func TestIsTTYWithNonTTYFile(t *testing.T) {
	// A regular temp file is not a TTY.
	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if isTTY(f) {
		t.Error("expected isTTY=false for a regular file")
	}
}

func TestIsTTYWithClosedFile(t *testing.T) {
	// A closed *os.File causes Stat() to return an error → isTTY returns false.
	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	f.Close() // close before calling isTTY

	if isTTY(f) {
		t.Error("expected isTTY=false for a closed file")
	}
}
