package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/vovanwin/configgen/internal/schema"
)

// Generate writes config.gen.go into outDir using package pkgName and schema fields
func Generate(outDir, pkgName string, fields map[string]*schema.Field) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	tmplPath := filepath.Join("templates", "config.go.tmpl")
	tmplB, err := ioutil.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	pl, err := template.New("cfg").Funcs(template.FuncMap{
		"GoType": goType,
		"keys": func(m map[string]*schema.Field) []string {
			ks := make([]string, 0, len(m))
			for k := range m {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			return ks
		},
	}).Parse(string(tmplB))
	if err != nil {
		return err
	}

	// sort keys for deterministic output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := &bytes.Buffer{}
	data := map[string]any{
		"Package": pkgName,
		"Fields":  fields,
		"Keys":    keys,
	}
	if err := pl.Execute(buf, data); err != nil {
		return err
	}

	outFile := filepath.Join(outDir, "config.gen.go")
	return ioutil.WriteFile(outFile, buf.Bytes(), 0o644)
}

func goType(f *schema.Field) string {
	switch f.Kind {
	case 0:
		return "string"
	case 1:
		return "int"
	case 2:
		return "float64"
	case 3:
		return "bool"
	case 4:
		// object type -> struct name
		return toGoStructName(f.YAMLName)
	default:
		return "interface{}"
	}
}

func toGoStructName(s string) string {
	// reuse simple CamelCase
	b := []rune(s)
	out := make([]rune, 0, len(b))
	capNext := true
	for _, r := range b {
		if r == '_' || r == '-' || r == ' ' {
			capNext = true
			continue
		}
		if capNext {
			if 'a' <= r && r <= 'z' {
				r = r - 'a' + 'A'
			}
			capNext = false
		}
		out = append(out, r)
	}
	// make sure first letter is uppercase
	return string(out)
}
