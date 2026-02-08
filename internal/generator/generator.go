package generator

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/vovanwin/configgen/internal/model"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Options настройки генерации кода
type Options struct {
	OutputDir   string           // Директория для сгенерированных файлов
	PackageName string           // Имя пакета
	EnvPrefix   string           // Префикс переменной окружения
	WithLoader  bool             // Генерировать loader.gen.go
	WithFlags   bool             // Генерировать flags файлы
	FlagDefs    []*model.FlagDef // Определения feature flags
}

// Generate генерирует config.gen.go и опционально loader.gen.go в указанную директорию
func Generate(opts Options, fields map[string]*model.Field) error {
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return fmt.Errorf("создание директории: %w", err)
	}

	if err := generateConfig(opts, fields); err != nil {
		return err
	}

	if opts.WithLoader {
		if err := generateLoader(opts, fields); err != nil {
			return err
		}
	}

	if opts.WithFlags && len(opts.FlagDefs) > 0 {
		if err := generateFlags(opts); err != nil {
			return err
		}
		if err := generateFlagStore(opts); err != nil {
			return err
		}
		if err := generateFlagTestHelpers(opts); err != nil {
			return err
		}
	}

	return nil
}

// generateConfig генерирует config.gen.go
func generateConfig(opts Options, fields map[string]*model.Field) error {
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
		"Package": opts.PackageName,
		"Fields":  fields,
		"Keys":    keys,
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("выполнение шаблона: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		outFile := filepath.Join(opts.OutputDir, "configgen_config.go")
		_ = os.WriteFile(outFile, buf.Bytes(), 0o644)
		return fmt.Errorf("форматирование кода: %w", err)
	}

	outFile := filepath.Join(opts.OutputDir, "configgen_config.go")
	return os.WriteFile(outFile, formatted, 0o644)
}

// generateLoader генерирует loader.gen.go
func generateLoader(opts Options, fields map[string]*model.Field) error {
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
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("выполнение шаблона loader: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		outFile := filepath.Join(opts.OutputDir, "configgen_loader.go")
		_ = os.WriteFile(outFile, buf.Bytes(), 0o644)
		return fmt.Errorf("форматирование loader кода: %w", err)
	}

	outFile := filepath.Join(opts.OutputDir, "configgen_loader.go")
	return os.WriteFile(outFile, formatted, 0o644)
}

// templateFuncs возвращает функции для использования в шаблонах
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"GoType":     goType,
		"GoItemType": goItemType,
		"keys": func(m map[string]*model.Field) []string {
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
func goType(f *model.Field) string {
	switch f.Kind {
	case model.KindString:
		return "string"
	case model.KindInt:
		return "int"
	case model.KindFloat:
		return "float64"
	case model.KindBool:
		return "bool"
	case model.KindDuration:
		return "time.Duration"
	case model.KindSlice:
		return "[]" + goItemType(f.ItemKind)
	case model.KindObject:
		return toGoStructName(f.TOMLName)
	default:
		return "any"
	}
}

// goItemType возвращает Go тип для элемента слайса
func goItemType(k model.Kind) string {
	switch k {
	case model.KindString:
		return "string"
	case model.KindInt:
		return "int"
	case model.KindFloat:
		return "float64"
	case model.KindBool:
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
func needsTime(fields map[string]*model.Field) bool {
	for _, f := range fields {
		if f.Kind == model.KindDuration {
			return true
		}
		if f.Kind == model.KindObject && needsTime(f.Children) {
			return true
		}
	}
	return false
}

// flagTemplateData данные для одного флага в шаблоне
type flagTemplateData struct {
	Name           string
	TOMLName       string
	Description    string
	GetterName     string // Имя геттера (= Name)
	GoType         string // "bool", "int", "float64", "string"
	StoreMethod    string // "GetBool", "GetInt", "GetFloat", "GetString"
	DefaultLiteral string // Литерал дефолта для кода
}

func buildFlagTemplateData(defs []*model.FlagDef) []flagTemplateData {
	result := make([]flagTemplateData, 0, len(defs))
	for _, d := range defs {
		result = append(result, flagTemplateData{
			Name:           d.Name,
			TOMLName:       d.TOMLName,
			Description:    d.Description,
			GetterName:     d.Name,
			GoType:         flagGoType(d.Kind),
			StoreMethod:    flagStoreMethod(d.Kind),
			DefaultLiteral: flagDefaultLiteral(d.Default, d.Kind),
		})
	}
	return result
}

func flagGoType(k model.FlagKind) string {
	switch k {
	case model.FlagKindBool:
		return "bool"
	case model.FlagKindInt:
		return "int"
	case model.FlagKindFloat:
		return "float64"
	case model.FlagKindString:
		return "string"
	default:
		return "any"
	}
}

func flagStoreMethod(k model.FlagKind) string {
	switch k {
	case model.FlagKindBool:
		return "GetBool"
	case model.FlagKindInt:
		return "GetInt"
	case model.FlagKindFloat:
		return "GetFloat"
	case model.FlagKindString:
		return "GetString"
	default:
		return "GetString"
	}
}

func flagDefaultLiteral(val any, kind model.FlagKind) string {
	switch kind {
	case model.FlagKindBool:
		if v, ok := val.(bool); ok && v {
			return "true"
		}
		return "false"
	case model.FlagKindInt:
		if v, ok := val.(int); ok {
			return strconv.Itoa(v)
		}
		return "0"
	case model.FlagKindFloat:
		if v, ok := val.(float64); ok {
			s := strconv.FormatFloat(v, 'f', -1, 64)
			if !strings.Contains(s, ".") {
				s += ".0"
			}
			return s
		}
		return "0.0"
	case model.FlagKindString:
		if v, ok := val.(string); ok {
			return strconv.Quote(v)
		}
		return `""`
	default:
		return `""`
	}
}

func generateFromTemplate(tmplName, tmplFile, outFile string, data map[string]any) error {
	tmplB, err := templatesFS.ReadFile(tmplFile)
	if err != nil {
		return fmt.Errorf("чтение шаблона %s: %w", tmplName, err)
	}

	tmpl, err := template.New(tmplName).Funcs(templateFuncs()).Parse(string(tmplB))
	if err != nil {
		return fmt.Errorf("парсинг шаблона %s: %w", tmplName, err)
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("выполнение шаблона %s: %w", tmplName, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		_ = os.WriteFile(outFile, buf.Bytes(), 0o644)
		return fmt.Errorf("форматирование %s: %w", tmplName, err)
	}

	return os.WriteFile(outFile, formatted, 0o644)
}

func generateFlags(opts Options) error {
	data := map[string]any{
		"Package": opts.PackageName,
		"Flags":   buildFlagTemplateData(opts.FlagDefs),
	}
	outFile := filepath.Join(opts.OutputDir, "configgen_flags.go")
	return generateFromTemplate("flags", "templates/flags.go.tmpl", outFile, data)
}

func generateFlagStore(opts Options) error {
	data := map[string]any{
		"Package": opts.PackageName,
	}
	outFile := filepath.Join(opts.OutputDir, "configgen_flagstore.go")
	return generateFromTemplate("flagstore", "templates/flagstore.go.tmpl", outFile, data)
}

func generateFlagTestHelpers(opts Options) error {
	data := map[string]any{
		"Package": opts.PackageName,
	}
	outFile := filepath.Join(opts.OutputDir, "configgen_flags_test_helpers.go")
	return generateFromTemplate("flags_test_helpers", "templates/flags_test_helpers.go.tmpl", outFile, data)
}
