// Package output provides output formatting utilities for the CLI.
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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Format represents an output format.
type Format string

const (
	// FormatTable outputs data in a table format.
	FormatTable Format = "table"
	// FormatJSON outputs data in JSON format.
	FormatJSON Format = "json"
	// FormatYAML outputs data in YAML format.
	FormatYAML Format = "yaml"
	// FormatWide outputs data in a wide table format with more columns.
	FormatWide Format = "wide"
)

// ParseFormat parses a format string into a Format type.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "table", "":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "wide":
		return FormatWide, nil
	default:
		return "", fmt.Errorf("unknown format: %s", s)
	}
}

// Formatter handles output formatting.
type Formatter struct {
	format Format
	writer io.Writer
}

// NewFormatter creates a new Formatter with the given format.
func NewFormatter(format Format) *Formatter {
	return &Formatter{
		format: format,
		writer: os.Stdout,
	}
}

// WithWriter sets the output writer.
func (f *Formatter) WithWriter(w io.Writer) *Formatter {
	f.writer = w
	return f
}

// Format returns the current format.
func (f *Formatter) Format() Format {
	return f.format
}

// Print prints the data in the configured format.
func (f *Formatter) Print(data interface{}) error {
	switch f.format {
	case FormatJSON:
		return f.printJSON(data)
	case FormatYAML:
		return f.printYAML(data)
	case FormatTable, FormatWide:
		// Table format requires special handling per type
		return fmt.Errorf("use PrintTable for table format")
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

func (f *Formatter) printJSON(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (f *Formatter) printYAML(data interface{}) error {
	encoder := yaml.NewEncoder(f.writer)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(data)
}

// TablePrinter prints data in a table format.
type TablePrinter struct {
	headers []string
	rows    [][]string
	writer  io.Writer
	wide    bool
}

// NewTablePrinter creates a new TablePrinter.
func NewTablePrinter() *TablePrinter {
	return &TablePrinter{
		writer: os.Stdout,
	}
}

// WithWriter sets the output writer.
func (t *TablePrinter) WithWriter(w io.Writer) *TablePrinter {
	t.writer = w
	return t
}

// WithWide enables wide output mode.
func (t *TablePrinter) WithWide(wide bool) *TablePrinter {
	t.wide = wide
	return t
}

// SetHeaders sets the table headers.
func (t *TablePrinter) SetHeaders(headers ...string) *TablePrinter {
	t.headers = headers
	return t
}

// AddRow adds a row to the table.
func (t *TablePrinter) AddRow(cells ...string) *TablePrinter {
	t.rows = append(t.rows, cells)
	return t
}

// Render renders the table to the writer.
func (t *TablePrinter) Render() error {
	if len(t.headers) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) {
				displayLen := len(stripANSI(cell))
				if displayLen > widths[i] {
					widths[i] = displayLen
				}
			}
		}
	}

	// Print headers
	var headerLine strings.Builder
	for i, h := range t.headers {
		if i > 0 {
			headerLine.WriteString("  ")
		}
		headerLine.WriteString(padRight(strings.ToUpper(h), widths[i]))
	}
	_, _ = fmt.Fprintln(t.writer, headerLine.String())

	// Print rows
	for _, row := range t.rows {
		var rowLine strings.Builder
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if i > 0 {
				rowLine.WriteString("  ")
			}
			rowLine.WriteString(padRight(cell, widths[i]))
		}
		_, _ = fmt.Fprintln(t.writer, rowLine.String())
	}

	return nil
}

func padRight(s string, width int) string {
	displayLen := len(stripANSI(s))
	if displayLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-displayLen)
}

func stripANSI(s string) string {
	// Simple regex-free ANSI stripper
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// Spinner provides a simple terminal spinner for long-running operations.
type Spinner struct {
	message string
	done    chan bool
	running bool
	writer  io.Writer
}

// NewSpinner creates a new Spinner with the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
		writer:  os.Stderr,
	}
}

// WithWriter sets the output writer.
func (s *Spinner) WithWriter(w io.Writer) *Spinner {
	s.writer = w
	return s
}

// Start starts the spinner animation.
func (s *Spinner) Start() {
	if s.running {
		return
	}
	s.running = true
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				_, _ = fmt.Fprintf(s.writer, "\r%s %s", frames[i], s.message)
				i = (i + 1) % len(frames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	if !s.running {
		return
	}
	s.running = false
	s.done <- true
	// Clear the spinner line
	_, _ = fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", len(s.message)+3))
}

// Success stops the spinner and prints a success message.
func (s *Spinner) Success(message string) {
	s.Stop()
	_, _ = fmt.Fprintf(s.writer, "✓ %s\n", message)
}

// Fail stops the spinner and prints a failure message.
func (s *Spinner) Fail(message string) {
	s.Stop()
	_, _ = fmt.Fprintf(s.writer, "%s %s\n", Colorize("✗", ColorRed), message)
}

// FailErr stops the spinner and prints a failure message with an error.
func (s *Spinner) FailErr(message string, err error) {
	s.Stop()
	_, _ = fmt.Fprintf(s.writer, "%s %s: %v\n", Colorize("✗", ColorRed), message, err)
}

// FormatDuration formats a duration in a human-readable way.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// FormatBytes formats a byte count in a human-readable way.
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// Color codes for terminal output.
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// Colorize applies a color to a string.
func Colorize(s, color string) string {
	return color + s + ColorReset
}

// ColorizeStatus returns a colorized status string.
func ColorizeStatus(status string) string {
	switch strings.ToLower(status) {
	case "active", "running", "healthy", "ready", "success":
		return Colorize(status, ColorGreen)
	case "draft", "pending", "working", "submitted", "starting":
		return Colorize(status, ColorYellow)
	case "archived", "stopped", "failed", "error", "cancelled", "unhealthy":
		return Colorize(status, ColorRed)
	default:
		return status
	}
}

// Success formats a success message.
func Success(s string) string {
	return Colorize("✓ "+s, ColorGreen)
}

// Error formats an error message.
func Error(s string) string {
	return Colorize("✗ "+s, ColorRed)
}

// Warning formats a warning message.
func Warning(s string) string {
	return Colorize("⚠ "+s, ColorYellow)
}

// Info formats an info message.
func Info(s string) string {
	return Colorize("ℹ "+s, ColorBlue)
}
