package dotenv

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	input := `
KEY1=value1
KEY2=value2
KEY3=value with spaces
`
	vars, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	expect := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value with spaces",
	}

	for k, v := range expect {
		if vars[k] != v {
			t.Errorf("key %q: got %q, want %q", k, vars[k], v)
		}
	}
}

func TestParseDoubleQuoted(t *testing.T) {
	input := `KEY="hello world"
ESCAPED="line1\nline2\ttab"
QUOTES="say \"hi\""
`
	vars, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if vars["KEY"] != "hello world" {
		t.Errorf("KEY: got %q, want %q", vars["KEY"], "hello world")
	}
	if vars["ESCAPED"] != "line1\nline2\ttab" {
		t.Errorf("ESCAPED: got %q, want %q", vars["ESCAPED"], "line1\nline2\ttab")
	}
	if vars["QUOTES"] != `say "hi"` {
		t.Errorf("QUOTES: got %q, want %q", vars["QUOTES"], `say "hi"`)
	}
}

func TestParseSingleQuoted(t *testing.T) {
	input := `KEY='hello world'
LITERAL='no\nescapes'
`
	vars, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if vars["KEY"] != "hello world" {
		t.Errorf("KEY: got %q, want %q", vars["KEY"], "hello world")
	}
	if vars["LITERAL"] != `no\nescapes` {
		t.Errorf("LITERAL: got %q, want %q", vars["LITERAL"], `no\nescapes`)
	}
}

func TestParseExportPrefix(t *testing.T) {
	input := `export KEY1=value1
export KEY2="value2"
`
	vars, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if vars["KEY1"] != "value1" {
		t.Errorf("KEY1: got %q, want %q", vars["KEY1"], "value1")
	}
	if vars["KEY2"] != "value2" {
		t.Errorf("KEY2: got %q, want %q", vars["KEY2"], "value2")
	}
}

func TestParseCommentsAndBlanks(t *testing.T) {
	input := `
# This is a comment
KEY1=value1

# Another comment
KEY2=value2

`
	vars, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(vars) != 2 {
		t.Errorf("expected 2 vars, got %d", len(vars))
	}
}

func TestParseInlineComment(t *testing.T) {
	input := `KEY=value # this is a comment
`
	vars, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if vars["KEY"] != "value" {
		t.Errorf("KEY: got %q, want %q", vars["KEY"], "value")
	}
}

func TestParseErrorReportsLineNumber(t *testing.T) {
	// Error is on line 3 — assert the reported line number is correct, not
	// just that an error occurs. (Catches off-by-one or direction errors in
	// the line counter.)
	input := "KEY1=value1\nKEY2=value2\nBAD_LINE_NO_EQUALS\n"
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for line without =")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("expected error to mention 'line 3', got: %v", err)
	}
}

func TestParseEmptySingleQuoted(t *testing.T) {
	// '' is a valid empty literal — must not be mistaken for an unterminated
	// single quote.
	vars, err := Parse(strings.NewReader("KEY=''\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if v, ok := vars["KEY"]; !ok || v != "" {
		t.Errorf("KEY: got %q (ok=%v), want empty string", v, ok)
	}
}

func TestParseEmptyDoubleQuoted(t *testing.T) {
	// Same idea for "".
	vars, err := Parse(strings.NewReader(`KEY=""` + "\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if v, ok := vars["KEY"]; !ok || v != "" {
		t.Errorf("KEY: got %q (ok=%v), want empty string", v, ok)
	}
}

func TestParseTrailingBackslashInDoubleQuoted(t *testing.T) {
	// `KEY="foo\"` — backslash is the last char inside the quotes. The escape
	// handler must not try to read past the end (which would panic).
	vars, err := Parse(strings.NewReader(`KEY="foo\"` + "\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if vars["KEY"] != `foo\` {
		t.Errorf("KEY: got %q, want %q", vars["KEY"], `foo\`)
	}
}

func TestParseEmptyValue(t *testing.T) {
	input := `KEY=
`
	vars, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if vars["KEY"] != "" {
		t.Errorf("KEY: got %q, want empty string", vars["KEY"])
	}
}

func TestParseMissingEquals(t *testing.T) {
	input := `INVALID_LINE
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for line without =")
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"has space", `"has space"`},
		{"has\nnewline", `"has\nnewline"`},
		{"has\ttab", `"has\ttab"`},
		{`has"quote`, `"has\"quote"`},
		{"", `""`},
	}

	for _, tt := range tests {
		got := Quote(tt.input)
		if got != tt.want {
			t.Errorf("Quote(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	vars := map[string]string{
		"SIMPLE":     "value",
		"WITH_SPACE": "hello world",
		"WITH_QUOTE": `say "hi"`,
		"EMPTY":      "",
	}

	var buf bytes.Buffer
	if err := Marshal(&buf, vars); err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	parsed, err := Parse(&buf)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for k, v := range vars {
		if parsed[k] != v {
			t.Errorf("round-trip key %q: got %q, want %q", k, parsed[k], v)
		}
	}
}
