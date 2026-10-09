package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/x/term"
)

type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
)

const (
	ExitSuccess = 0
	ExitGeneral = 1
	ExitUsage   = 2
	ExitNetwork = 3
)

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func DetermineExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "timed out") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "dial tcp") {
		return ExitNetwork
	}
	if strings.Contains(errStr, "unknown command") ||
		strings.Contains(errStr, "unknown flag") ||
		strings.Contains(errStr, "required flag") ||
		strings.Contains(errStr, "accepts ") ||
		strings.Contains(errStr, "requires at least") ||
		strings.Contains(errStr, "invalid argument") ||
		strings.Contains(errStr, "usage:") {
		return ExitUsage
	}
	return ExitGeneral
}

type Formatter struct {
	Format  OutputFormat
	Out     io.Writer
	Err     io.Writer
	JSONOut bool
}

func NewFormatter(out, err io.Writer, jsonOut bool) *Formatter {
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	fmtType := FormatTable
	if jsonOut {
		fmtType = FormatJSON
	}
	return &Formatter{
		Format:  fmtType,
		Out:     out,
		Err:     err,
		JSONOut: jsonOut,
	}
}

func (f *Formatter) PrintJSON(v any) error {
	enc := json.NewEncoder(f.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (f *Formatter) PrintTable(headers []string, rows [][]string) error {
	if f.JSONOut {
		var mapped []map[string]string
		for _, row := range rows {
			m := make(map[string]string)
			for i, h := range headers {
				if i < len(row) {
					m[h] = row[i]
				} else {
					m[h] = ""
				}
			}
			mapped = append(mapped, m)
		}
		return f.PrintJSON(mapped)
	}

	termW := 0
	if fFile, ok := f.Out.(*os.File); ok {
		if term.IsTerminal(fFile.Fd()) {
			if w, _, err := term.GetSize(fFile.Fd()); err == nil && w > 0 {
				termW = w
			}
		}
	}

	finalRows := rows
	if termW > 0 && len(headers) > 0 {
		maxCellW := max(12, (termW-len(headers)*3)/len(headers))
		finalRows = make([][]string, len(rows))
		for rIdx, row := range rows {
			rCopy := make([]string, len(row))
			for cIdx, cell := range row {
				if len(cell) > maxCellW && maxCellW > 4 {
					rCopy[cIdx] = cell[:maxCellW-3] + "..."
				} else {
					rCopy[cIdx] = cell
				}
			}
			finalRows[rIdx] = rCopy
		}
	}

	w := tabwriter.NewWriter(f.Out, 0, 0, 3, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(w, "\t")
		}
		_, _ = fmt.Fprint(w, strings.ToUpper(h))
	}
	_, _ = fmt.Fprintln(w)

	for _, row := range finalRows {
		for i, cell := range row {
			if i > 0 {
				_, _ = fmt.Fprint(w, "\t")
			}
			_, _ = fmt.Fprint(w, cell)
		}
		_, _ = fmt.Fprintln(w)
	}

	return w.Flush()
}

func (f *Formatter) PrintMessage(format string, a ...any) {
	if f.JSONOut {
		return
	}
	_, _ = fmt.Fprintf(f.Out, "%s\n", fmt.Sprintf(format, a...))
}

func (f *Formatter) PrintError(format string, a ...any) {
	if f.JSONOut {
		msg := fmt.Sprintf(format, a...)
		_ = json.NewEncoder(f.Err).Encode(map[string]string{"error": msg})
		return
	}
	_, _ = fmt.Fprintf(f.Err, "Error: %s\n", fmt.Sprintf(format, a...))
}
