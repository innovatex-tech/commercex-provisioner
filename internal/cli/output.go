package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// ─── Exit Codes ───────────────────────────────────────────────────────────────

const (
	ExitOK            = 0
	ExitRuntime       = 1
	ExitValidation    = 2
	ExitNotFound      = 3
	ExitAuth          = 4
	ExitConflict      = 5
	ExitConfirmation  = 6
)

// ─── Output Format ────────────────────────────────────────────────────────────

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// DetectFormat returns FormatTable if stdout is a TTY, FormatJSON otherwise.
// Respects --format flag override and NO_COLOR / INNOVATEX_FORMAT env vars.
func DetectFormat() Format {
	if f := os.Getenv("INNOVATEX_FORMAT"); f == "json" {
		return FormatJSON
	}
	if fi, err := os.Stdout.Stat(); err == nil {
		if fi.Mode()&os.ModeCharDevice != 0 {
			return FormatTable
		}
	}
	return FormatJSON
}

// ─── Envelope ─────────────────────────────────────────────────────────────────

type Envelope struct {
	OK   bool        `json:"ok"`
	Data interface{} `json:"data,omitempty"`
	Error *APIError   `json:"error,omitempty"`
	Meta *Meta       `json:"meta,omitempty"`
}

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type Meta struct {
	SchemaVersion string `json:"schema_version,omitempty"`
}

// ─── Writer ───────────────────────────────────────────────────────────────────

// Writer handles format-aware output to stdout (data) and stderr (diagnostics).
type Writer struct {
	format Format
	stdout io.Writer
	stderr io.Writer
}

func NewWriter(format Format) *Writer {
	return &Writer{
		format: format,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

func NewWriterTo(stdout, stderr io.Writer, format Format) *Writer {
	return &Writer{
		format: format,
		stdout: stdout,
		stderr: stderr,
	}
}

// Success writes a success envelope to stdout.
func (w *Writer) Success(data interface{}) {
	env := Envelope{
		OK:   true,
		Data: data,
		Meta: &Meta{SchemaVersion: "1.0.0"},
	}
	w.writeJSON(env)
}

// Error writes an error envelope to stdout and returns a CLIError.
func (w *Writer) Error(code string, message string, retryable bool) *CLIError {
	env := Envelope{
		OK: false,
		Error: &APIError{
			Code:      code,
			Message:   message,
			Retryable: retryable,
		},
		Meta: &Meta{SchemaVersion: "1.0.0"},
	}
	w.writeJSON(env)
	return &CLIError{Code: code, Message: message, Retryable: retryable}
}

// Errorf writes an error envelope to stdout, prints diagnostics to stderr, and returns a CLIError.
func (w *Writer) Errorf(code string, message string, retryable bool, details ...interface{}) *CLIError {
	if len(details) > 0 {
		fmt.Fprintf(w.stderr, "%s\n", fmt.Sprint(details...))
	}
	return w.Error(code, message, retryable)
}

// Diag writes a human-readable diagnostic to stderr (never stdout).
func (w *Writer) Diag(format string, args ...interface{}) {
	fmt.Fprintf(w.stderr, format+"\n", args...)
}

func (w *Writer) writeJSON(v interface{}) {
	enc := json.NewEncoder(w.stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func exitCodeFor(code string) int {
	switch code {
	case "validation_error":
		return ExitValidation
	case "not_found":
		return ExitNotFound
	case "auth_error":
		return ExitAuth
	case "conflict":
		return ExitConflict
	case "confirmation_required":
		return ExitConfirmation
	default:
		return ExitRuntime
	}
}

// ─── CLIError ─────────────────────────────────────────────────────────────────

// CLIError is an error that carries a structured error code for agent routing.
type CLIError struct {
	Code      string
	Message   string
	Retryable bool
}

func (e *CLIError) Error() string {
	return e.Message
}

// ExitCode returns the numeric exit code for this error class.
func (e *CLIError) ExitCode() int {
	return exitCodeFor(e.Code)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// IsTTY returns true if fd is a terminal.
func IsTTY(fd *os.File) bool {
	fi, err := fd.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// ParseFormat converts a string to Format, defaulting to table.
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "table", "":
		return FormatTable
	default:
		return FormatTable
	}
}
