package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed template.go.tmpl
var configTemplate string

// Options настройки генерации
type Options struct {
	ConfigDir string // Директория с конфигами (default: ./configs)
	OutputDir string // Директория для генерации (default: ./internal/config)
	Package   string // Имя пакета (default: config)
}

// DefaultOptions настройки по умолчанию
func DefaultOptions() Options {
	return Options{
		ConfigDir: "./configs",
		OutputDir: "./internal/config",
		Package:   "config",
	}
}

// Generate генерирует Go код из конфигов
func Generate(opts Options) error {
	if opts.ConfigDir == "" {
		opts.ConfigDir = DefaultOptions().ConfigDir
	}
	if opts.OutputDir == "" {
		opts.OutputDir = DefaultOptions().OutputDir
	}
	if opts.Package == "" {
		opts.Package = DefaultOptions().Package
	}

	// Собираем все конфиги
	var allFields []map[string]*Field

	// value.toml
	valuePath := filepath.Join(opts.ConfigDir, "value.toml")
	if _, err := os.Stat(valuePath); err == nil {
		fields, err := ParseFile(valuePath)
		if err != nil {
			return fmt.Errorf("value.toml: %w", err)
		}
		allFields = append(allFields, fields)
		fmt.Printf("  ✓ value.toml\n")
	}

	// config_*.toml
	pattern := filepath.Join(opts.ConfigDir, "config_*.toml")
	files, _ := filepath.Glob(pattern)

	for _, f := range files {
		if filepath.Base(f) == "config_local.toml" {
			continue
		}
		fields, err := ParseFile(f)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		allFields = append(allFields, fields)
		fmt.Printf("  ✓ %s\n", filepath.Base(f))
	}

	if len(allFields) == 0 {
		return fmt.Errorf("не найдены конфиг файлы в %s", opts.ConfigDir)
	}

	// Объединяем все поля
	merged := unionFields(allFields...)

	// Собираем ключи
	var keys []keyData
	var needsTime bool
	collectKeys(merged, "", &keys, &needsTime)

	// Данные для шаблона
	data := map[string]any{
		"Package":   opts.Package,
		"Keys":      keys,
		"NeedsTime": needsTime,
	}

	// Генерируем
	tmpl, err := template.New("config").Funcs(template.FuncMap{
		"comment": formatComment,
	}).Parse(configTemplate)
	if err != nil {
		return fmt.Errorf("шаблон: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("генерация: %w", err)
	}

	// Форматируем
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		os.MkdirAll(opts.OutputDir, 0755)
		os.WriteFile(filepath.Join(opts.OutputDir, "config.gen.go"), buf.Bytes(), 0644)
		return fmt.Errorf("форматирование: %w", err)
	}

	// Сохраняем
	os.MkdirAll(opts.OutputDir, 0755)
	outPath := filepath.Join(opts.OutputDir, "config.gen.go")
	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("запись: %w", err)
	}

	fmt.Printf("\n✓ Сгенерировано: %s\n", outPath)
	return nil
}

func unionFields(maps ...map[string]*Field) map[string]*Field {
	result := make(map[string]*Field)
	for _, m := range maps {
		for k, f := range m {
			if existing, ok := result[k]; ok && existing.Children != nil && f.Children != nil {
				existing.Children = unionFields(existing.Children, f.Children)
			} else if !ok {
				result[k] = f
			}
		}
	}
	return result
}

type keyData struct {
	Const   string
	Value   string
	Comment string
	Type    string
}

func collectKeys(fields map[string]*Field, prefix string, keys *[]keyData, needsTime *bool) {
	for _, k := range sortedKeys(fields) {
		f := fields[k]
		fullKey := f.TOMLName
		if prefix != "" {
			fullKey = prefix + "." + f.TOMLName
		}

		if f.Children != nil {
			collectKeys(f.Children, fullKey, keys, needsTime)
		} else {
			if f.Type == "time.Duration" {
				*needsTime = true
			}
			*keys = append(*keys, keyData{
				Const:   toConstName(fullKey),
				Value:   fullKey,
				Comment: f.Comment,
				Type:    f.Type,
			})
		}
	}
}

func toConstName(key string) string {
	parts := strings.Split(key, ".")
	var result []string
	for _, p := range parts {
		result = append(result, toCamelCase(p))
	}
	return strings.Join(result, "")
}

func sortedKeys(m map[string]*Field) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatComment(comment string) string {
	if comment == "" {
		return ""
	}
	lines := strings.Split(comment, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, "// "+line)
	}
	return strings.Join(result, "\n")
}
