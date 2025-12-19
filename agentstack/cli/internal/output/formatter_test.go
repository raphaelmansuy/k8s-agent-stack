/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		wantErr  bool
	}{
		{"table", FormatTable, false},
		{"TABLE", FormatTable, false},
		{"", FormatTable, false},
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"yaml", FormatYAML, false},
		{"yml", FormatYAML, false},
		{"wide", FormatWide, false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatterJSON(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatJSON).WithWriter(&buf)

	data := map[string]string{"key": "value"}
	err := f.Print(data)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"key"`) || !strings.Contains(output, `"value"`) {
		t.Errorf("JSON output = %v, expected key-value pair", output)
	}
}

func TestFormatterYAML(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatYAML).WithWriter(&buf)

	data := map[string]string{"key": "value"}
	err := f.Print(data)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "key: value") {
		t.Errorf("YAML output = %v, expected key: value", output)
	}
}

func TestTablePrinter(t *testing.T) {
	var buf bytes.Buffer
	tp := NewTablePrinter().WithWriter(&buf)

	tp.SetHeaders("Name", "Status", "Age").
		AddRow("agent-1", "Running", "5m").
		AddRow("agent-2", "Stopped", "1h")

	err := tp.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()

	// Check headers
	if !strings.Contains(output, "NAME") {
		t.Errorf("Expected NAME header in output: %v", output)
	}
	if !strings.Contains(output, "STATUS") {
		t.Errorf("Expected STATUS header in output: %v", output)
	}
	if !strings.Contains(output, "AGE") {
		t.Errorf("Expected AGE header in output: %v", output)
	}

	// Check rows
	if !strings.Contains(output, "agent-1") {
		t.Errorf("Expected agent-1 in output: %v", output)
	}
	if !strings.Contains(output, "Running") {
		t.Errorf("Expected Running in output: %v", output)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    int // seconds
		expected string
	}{
		{30, "30s"},
		{65, "1m5s"},
		{3665, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			d := time.Duration(tt.input) * time.Second
			got := FormatDuration(d)
			if got != tt.expected {
				t.Errorf("FormatDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := FormatBytes(tt.input)
			if got != tt.expected {
				t.Errorf("FormatBytes(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestColorize(t *testing.T) {
	result := Colorize("test", ColorRed)
	if !strings.HasPrefix(result, ColorRed) || !strings.HasSuffix(result, ColorReset) {
		t.Errorf("Colorize() = %v, expected color codes", result)
	}
}

func TestSuccessErrorWarningInfo(t *testing.T) {
	// Just verify they don't panic and contain the message
	s := Success("done")
	if !strings.Contains(s, "done") {
		t.Errorf("Success() = %v, expected to contain 'done'", s)
	}

	e := Error("failed")
	if !strings.Contains(e, "failed") {
		t.Errorf("Error() = %v, expected to contain 'failed'", e)
	}

	w := Warning("caution")
	if !strings.Contains(w, "caution") {
		t.Errorf("Warning() = %v, expected to contain 'caution'", w)
	}

	i := Info("note")
	if !strings.Contains(i, "note") {
		t.Errorf("Info() = %v, expected to contain 'note'", i)
	}
}
