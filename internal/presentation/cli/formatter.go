package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
)

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

	w := tabwriter.NewWriter(f.Out, 0, 0, 3, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(w, "\t")
		}
		_, _ = fmt.Fprint(w, h)
	}
	_, _ = fmt.Fprintln(w)

	for _, row := range rows {
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
