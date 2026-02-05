package generator

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/vovanwin/configgen/pkg/types"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Options настройки генерации кода
type Options struct {
	OutputDir   string // Директория для сгенерированных файлов
	PackageName string // Имя пакета
	EnvPrefix   string // Префикс переменной окружения
	WithLoader  bool   // Генерировать loader.gen.go
	WithVault   bool   // Включить заготовку для Vault
	WithRTC     bool   // Включить заготовку для RTC
}

// Generate генерирует config.gen.go и опционально loader.gen.go в указанную директорию
func Generate(opts Options, fields map[string]*types.Field) error {
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return fmt.Errorf("создание директории: %w", err)
	}

	// Генерируем структуры конфига
	if err := generateConfig(opts, fields); err != nil {
		return err
	}

	// Генерируем загрузчик
	if opts.WithLoader {
		if err := generateLoader(opts, fields); err != nil {
			return err
		}
	}

	return nil
}

// generateConfig генерирует config.gen.go
func generateConfig(opts Options, fields map[string]*types.Field) error {
	tmplB, err := templatesFS.ReadFile("templates/config.go.tmpl")
	if err != nil {
		return fmt.Errorf("чтение шаблона: %w", err)
	}

	tmpl, err := template.New("cfg").Funcs(templateFuncs()).Parse(string(tmplB))
	if err != nil {
		return fmt.Errorf("парсинг шаблона: %w", err)
	}

	keys := sortedKeys(fields)

	buf := &bytes.Buffer{}
	data := map[string]any{
		"Package":   opts.PackageName,
		"Fields":    fields,
		"Keys":      keys,
		"WithVault": opts.WithVault,
		"WithRTC":   opts.WithRTC,
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("выполнение шаблона: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Сохраняем неформатированный для отладки
		outFile := filepath.Join(opts.OutputDir, "config.gen.go")
		_ = os.WriteFile(outFile, buf.Bytes(), 0o644)
		return fmt.Errorf("форматирование кода: %w", err)
	}

	outFile := filepath.Join(opts.OutputDir, "config.gen.go")
	return os.WriteFile(outFile, formatted, 0o644)
}

// generateLoader генерирует loader.gen.go
func generateLoader(opts Options, fields map[string]*types.Field) error {
	tmplB, err := templatesFS.ReadFile("templates/loader.go.tmpl")
	if err != nil {
		return fmt.Errorf("чтение шаблона loader: %w", err)
	}

	tmpl, err := template.New("loader").Funcs(templateFuncs()).Parse(string(tmplB))
	if err != nil {
		return fmt.Errorf("парсинг шаблона loader: %w", err)
	}

	buf := &bytes.Buffer{}
	data := map[string]any{
		"Package":   opts.PackageName,
		"EnvPrefix": opts.EnvPrefix,
		"WithVault": opts.WithVault,
		"WithRTC":   opts.WithRTC,
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("выполнение шаблона loader: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		outFile := filepath.Join(opts.OutputDir, "loader.gen.go")
		_ = os.WriteFile(outFile, buf.Bytes(), 0o644)
		return fmt.Errorf("форматирование loader кода: %w", err)
	}

	outFile := filepath.Join(opts.OutputDir, "loader.gen.go")
	return os.WriteFile(outFile, formatted, 0o644)
}

// templateFuncs возвращает функции для использования в шаблонах
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"GoType":     goType,
		"GoItemType": goItemType,
		"keys": func(m map[string]*types.Field) []string {
			return sortedKeys(m)
		},
		"needsTime":     needsTime,
		"formatComment": formatComment,
		"hasComment":    hasComment,
	}
}

// formatComment форматирует комментарий для Go кода
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

// hasComment проверяет есть ли комментарий
func hasComment(comment string) bool {
	return comment != ""
}

// goType возвращает Go тип для поля
func goType(f *types.Field) string {
	switch f.Kind {
	case types.KindString:
		return "string"
	case types.KindInt:
		return "int"
	case types.KindFloat:
		return "float64"
	case types.KindBool:
		return "bool"
	case types.KindDuration:
		return "time.Duration"
	case types.KindSlice:
		return "[]" + goItemType(f.ItemKind)
	case types.KindObject:
		return toGoStructName(f.TOMLName)
	default:
		return "any"
	}
}

// goItemType возвращает Go тип для элемента слайса
func goItemType(k types.Kind) string {
	switch k {
	case types.KindString:
		return "string"
	case types.KindInt:
		return "int"
	case types.KindFloat:
		return "float64"
	case types.KindBool:
		return "bool"
	default:
		return "any"
	}
}

// toGoStructName конвертирует имя в CamelCase для структур
func toGoStructName(s string) string {
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
	return string(out)
}

// sortedKeys возвращает отсортированные ключи map
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// needsTime проверяет использует ли какое-либо поле time.Duration
func needsTime(fields map[string]*types.Field) bool {
	for _, f := range fields {
		if f.Kind == types.KindDuration {
			return true
		}
		if f.Kind == types.KindObject && needsTime(f.Children) {
			return true
		}
	}
	return false
}
