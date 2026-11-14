// Package report renders analysis findings as text or JSON.
package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ckoehn/helm-values-tidy/internal/analyze"
)

// Report is the complete result for one chart.
type Report struct {
	Chart     string            `json:"chart"`
	Redundant []analyze.Finding `json:"redundant"`
	Unknown   []analyze.Finding `json:"unknown"`
}

// New splits findings by kind.
func New(chart string, findings []analyze.Finding) Report {
	r := Report{Chart: chart, Redundant: []analyze.Finding{}, Unknown: []analyze.Finding{}}
	for _, f := range findings {
		switch f.Kind {
		case analyze.Redundant:
			r.Redundant = append(r.Redundant, f)
		case analyze.Unknown:
			r.Unknown = append(r.Unknown, f)
		}
	}
	return r
}

// WriteJSON writes r as indented JSON.
func WriteJSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteText writes r in a human-readable form.
func WriteText(w io.Writer, r Report) error {
	if _, err := fmt.Fprintf(w, "Chart: %s\n\n", r.Chart); err != nil {
		return err
	}
	if err := section(w, "Redundant (equal to chart default)", r.Redundant); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return section(w, "Unknown (not defined by chart)", r.Unknown)
}

func section(w io.Writer, title string, fs []analyze.Finding) error {
	if _, err := fmt.Fprintf(w, "%s: %d\n", title, len(fs)); err != nil {
		return err
	}
	for _, f := range fs {
		if _, err := fmt.Fprintf(w, "  %s = %s\n", f.Path, format(f.Value)); err != nil {
			return err
		}
	}
	return nil
}

// format renders strings bare and everything else as compact JSON.
func format(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
