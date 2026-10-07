package reportgen

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Options controls where Generate puts its output.
type Options struct {
	OutPath  string // if set, write the PDF here in addition to returning it
	TypstDir string // if set, write an editable Typst project here
}

// assetRoot unpacks the embedded templates and fonts once per process.
// Typst needs them on disk because it only reads paths below --root.
var assetRoot = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "reportgen-*")
	if err != nil {
		return "", err
	}
	if err := extractAssets(dir); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
})

// Generate compiles report JSON into a PDF and returns its bytes.
// Requires the typst binary on PATH. Safe for concurrent use.
func Generate(reportJSON []byte, opt Options) ([]byte, error) {
	if !json.Valid(reportJSON) {
		return nil, fmt.Errorf("reportgen: input is not valid JSON")
	}
	root, err := assetRoot()
	if err != nil {
		return nil, fmt.Errorf("reportgen: unpack templates: %w", err)
	}
	if opt.TypstDir != "" {
		if err := extractAssets(opt.TypstDir); err != nil {
			return nil, fmt.Errorf("reportgen: export Typst project: %w", err)
		}
		if err := os.WriteFile(filepath.Join(opt.TypstDir, "data.json"), reportJSON, 0o644); err != nil {
			return nil, fmt.Errorf("reportgen: export Typst data: %w", err)
		}
	}

	data, err := os.CreateTemp(root, "data-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(data.Name())
	if _, err := data.Write(reportJSON); err != nil {
		data.Close()
		return nil, err
	}
	if err := data.Close(); err != nil {
		return nil, err
	}

	pdfFile, err := os.CreateTemp(root, "out-*.pdf")
	if err != nil {
		return nil, err
	}
	pdfFile.Close()
	defer os.Remove(pdfFile.Name())

	cmd := exec.Command(
		"typst", "compile",
		"--font-path", filepath.Join(root, "assets", "fonts"),
		"--root", root,
		"--input", "data=/"+filepath.Base(data.Name()),
		filepath.Join(root, "templates", "report.typ"),
		pdfFile.Name(),
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reportgen: typst compile: %w", err)
	}

	pdf, err := os.ReadFile(pdfFile.Name())
	if err != nil {
		return nil, err
	}
	if len(pdf) == 0 {
		return nil, fmt.Errorf("reportgen: typst produced an empty PDF")
	}

	if opt.OutPath != "" {
		if err := os.MkdirAll(filepath.Dir(opt.OutPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(opt.OutPath, pdf, 0o644); err != nil {
			return nil, err
		}
	}
	return pdf, nil
}

func extractAssets(dest string) error {
	return fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case path == ".":
			return nil
		case d.IsDir():
			return os.MkdirAll(filepath.Join(dest, path), 0o755)
		}
		b, err := assets.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, path), b, 0o644)
	})
}
