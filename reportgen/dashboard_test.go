package reportgen_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Marist-Community-and-Belonging/report-gen-pdf/reportgen"
)

// processedDir points at the dashboard-primitives extracts. Override with
// OCB_PROCESSED_DIR when the sibling checkout lives elsewhere.
func processedDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("OCB_PROCESSED_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "dashboard-primitives", "data", "processed")
	}
	if _, err := os.Stat(filepath.Join(dir, "overview-2023.json")); err != nil {
		t.Skipf("no processed extracts at %s", dir)
	}
	return dir
}

func TestBuildFromProcessed(t *testing.T) {
	b, err := reportgen.BuildFromProcessed(processedDir(t), 2023)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Intro    string `json:"intro"`
		Chapters []struct {
			Heading string `json:"heading"`
			Figures []struct {
				Kind   string `json:"kind"`
				Title  string `json:"title"`
				Series []struct {
					DisplayValues []float64 `json:"display_values"`
				} `json:"series"`
			} `json:"figures"`
		} `json:"chapters"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(doc.Intro), []byte("19 institutions: 10 peers and 8 aspirants")) {
		t.Errorf("intro omits dashboard comparison counts: %q", doc.Intro)
	}
	// One chapter per dashboard page, minus the live campus-involvement feed.
	want := []string{
		"Overview",
		"Multi-year trends",
		"Diversity and access",
		"Success and equity",
		"Affordability",
		"Marist profile",
	}
	if len(doc.Chapters) != len(want) {
		t.Fatalf("want %d chapters, got %d", len(want), len(doc.Chapters))
	}
	foundSubgroupRates := false
	foundCompositionValues := false
	for i, c := range doc.Chapters {
		if c.Heading != want[i] {
			t.Errorf("chapter %d: want %q, got %q", i, want[i], c.Heading)
		}
		if len(c.Figures) == 0 {
			t.Errorf("chapter %q has no figures", c.Heading)
		}
		for _, f := range c.Figures {
			switch f.Kind {
			case "bar", "grouped", "line", "bullet", "stacked", "scatter", "stats":
			default:
				t.Errorf("chapter %q: unknown figure kind %q", c.Heading, f.Kind)
			}
			if c.Heading == "Success and equity" && f.Title == "Subgroup six-year graduation rates" {
				foundSubgroupRates = true
			}
			if f.Title == "Undergraduate composition" && len(f.Series) > 0 && len(f.Series[0].DisplayValues) > 0 {
				foundCompositionValues = true
			}
		}
	}
	if !foundSubgroupRates {
		t.Error("success chapter omits dashboard subgroup comparison rates")
	}
	if !foundCompositionValues {
		t.Error("composition chart omits exact reported percentages")
	}
}

func TestBuildFromFS(t *testing.T) {
	if _, err := reportgen.BuildFromFS(os.DirFS(processedDir(t)), ".", 2023); err != nil {
		t.Fatal(err)
	}
}

func TestBuildFromProcessedRequiresOverviewMetrics(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "overview-2023.json"), []byte(`{"metrics":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := reportgen.BuildFromProcessed(dir, 2023); err == nil || !strings.Contains(err.Error(), "no metrics") {
		t.Fatalf("expected missing overview metrics error, got %v", err)
	}
}

// The earliest collection has no prior years, so trend charts must drop out
// rather than render an empty chapter or a single-point line.
func TestEarliestYearHasNoStubCharts(t *testing.T) {
	b, err := reportgen.BuildFromProcessed(processedDir(t), 2019)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Chapters []struct {
			Heading string `json:"heading"`
			Figures []struct {
				Kind    string `json:"kind"`
				Title   string `json:"title"`
				XLabels []any  `json:"x_labels"`
			} `json:"figures"`
		} `json:"chapters"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for _, c := range doc.Chapters {
		if len(c.Figures) == 0 {
			t.Errorf("chapter %q kept with no figures", c.Heading)
		}
		if c.Heading == "Multi-year trends" {
			t.Errorf("trends chapter kept for the earliest collection")
		}
		for _, f := range c.Figures {
			if f.Kind == "line" && len(f.XLabels) < 2 {
				t.Errorf("figure %q: line chart with %d points", f.Title, len(f.XLabels))
			}
		}
	}
}

func TestGenerate(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not on PATH")
	}
	report, err := reportgen.BuildFromProcessed(processedDir(t), 2023)
	if err != nil {
		t.Fatal(err)
	}
	typstDir := filepath.Join(t.TempDir(), "report-typst")
	pdf, err := reportgen.Generate(report, reportgen.Options{TypstDir: typstDir})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("not a PDF: % x", pdf[:min(8, len(pdf))])
	}
	exportedPDF := filepath.Join(t.TempDir(), "exported.pdf")
	cmd := exec.Command(
		"typst", "compile",
		"--font-path", filepath.Join(typstDir, "assets", "fonts"),
		"--root", typstDir,
		filepath.Join(typstDir, "templates", "report.typ"),
		exportedPDF,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile exported Typst project: %v\n%s", err, output)
	}
	exported, err := os.ReadFile(exportedPDF)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(exported, []byte("%PDF")) {
		t.Fatalf("exported Typst project did not produce a PDF")
	}
}
