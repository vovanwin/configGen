package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vovanwin/configgen/internal/model"
)

func TestParseFlagsFile_HappyPath(t *testing.T) {
	content := `
[flags]
new_catalog_ui = { type = "bool", default = false, description = "Новый UI каталога" }
rate_limit = { type = "int", default = 100, description = "Rate limit" }
threshold = { type = "float", default = 0.5, description = "Порог" }
banner_text = { type = "string", default = "hello", description = "Текст баннера" }
`
	path := writeTempFile(t, "flags.toml", content)

	defs, err := ParseFlagsFile(path)
	if err != nil {
		t.Fatalf("ParseFlagsFile: %v", err)
	}

	if len(defs) != 4 {
		t.Fatalf("ожидалось 4 флага, получено %d", len(defs))
	}

	// Проверяем типы
	expected := map[string]model.FlagKind{
		"banner_text":    model.FlagKindString,
		"new_catalog_ui": model.FlagKindBool,
		"rate_limit":     model.FlagKindInt,
		"threshold":      model.FlagKindFloat,
	}

	for i, def := range defs {
		wantKind, ok := expected[def.TOMLName]
		if !ok {
			t.Errorf("неожиданный флаг %q на позиции %d", def.TOMLName, i)
			continue
		}
		if def.Kind != wantKind {
			t.Errorf("флаг %q: ожидался тип %v, получен %v", def.TOMLName, wantKind, def.Kind)
		}
	}

	// Проверяем дефолты
	if defs[0].Default != "hello" {
		t.Errorf("banner_text default: ожидался \"hello\", получен %v", defs[0].Default)
	}
	if defs[1].Default != false {
		t.Errorf("new_catalog_ui default: ожидался false, получен %v", defs[1].Default)
	}
	if defs[2].Default != 100 {
		t.Errorf("rate_limit default: ожидался 100, получен %v", defs[2].Default)
	}
	if defs[3].Default != 0.5 {
		t.Errorf("threshold default: ожидался 0.5, получен %v", defs[3].Default)
	}
}

func TestParseFlagsFile_Sorted(t *testing.T) {
	content := `
[flags]
zebra = { type = "bool", default = true, description = "Z" }
alpha = { type = "bool", default = false, description = "A" }
middle = { type = "bool", default = true, description = "M" }
`
	path := writeTempFile(t, "flags.toml", content)

	defs, err := ParseFlagsFile(path)
	if err != nil {
		t.Fatalf("ParseFlagsFile: %v", err)
	}

	if len(defs) != 3 {
		t.Fatalf("ожидалось 3 флага, получено %d", len(defs))
	}

	if defs[0].TOMLName != "alpha" || defs[1].TOMLName != "middle" || defs[2].TOMLName != "zebra" {
		t.Errorf("неправильная сортировка: %s, %s, %s", defs[0].TOMLName, defs[1].TOMLName, defs[2].TOMLName)
	}
}

func TestParseFlagsFile_InvalidType(t *testing.T) {
	content := `
[flags]
bad = { type = "map", default = false, description = "Bad" }
`
	path := writeTempFile(t, "flags.toml", content)

	_, err := ParseFlagsFile(path)
	if err == nil {
		t.Fatal("ожидалась ошибка для невалидного типа")
	}
}

func TestParseFlagsFile_TypeDefaultMismatch(t *testing.T) {
	content := `
[flags]
bad = { type = "bool", default = 42, description = "Mismatch" }
`
	path := writeTempFile(t, "flags.toml", content)

	_, err := ParseFlagsFile(path)
	if err == nil {
		t.Fatal("ожидалась ошибка для несовпадения type/default")
	}
}

func TestParseFlagsFile_Empty(t *testing.T) {
	content := `
[flags]
`
	path := writeTempFile(t, "flags.toml", content)

	defs, err := ParseFlagsFile(path)
	if err != nil {
		t.Fatalf("ParseFlagsFile: %v", err)
	}

	if len(defs) != 0 {
		t.Errorf("ожидалось 0 флагов, получено %d", len(defs))
	}
}

func TestParseFlagsFile_GoNames(t *testing.T) {
	content := `
[flags]
new_catalog_ui = { type = "bool", default = false, description = "UI" }
`
	path := writeTempFile(t, "flags.toml", content)

	defs, err := ParseFlagsFile(path)
	if err != nil {
		t.Fatalf("ParseFlagsFile: %v", err)
	}

	if defs[0].Name != "NewCatalogUi" {
		t.Errorf("ожидался Go name \"NewCatalogUi\", получен %q", defs[0].Name)
	}
}

func TestParseFlagsFile_Enum(t *testing.T) {
	content := `
[flags]
environment = { type = "enum", values = ["dev", "stg", "prod"], default = "dev", description = "Environment" }
log_format = { type = "enum", values = ["json", "text"], default = "json", description = "Log format" }
`
	path := writeTempFile(t, "flags.toml", content)
	defs, err := ParseFlagsFile(path)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(defs) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(defs))
	}

	// Проверяем environment флаг
	envFlag := defs[0]
	if envFlag.Kind != model.FlagKindEnum {
		t.Errorf("expected enum kind, got %v", envFlag.Kind)
	}
	if envFlag.Default != "dev" {
		t.Errorf("expected default 'dev', got %v", envFlag.Default)
	}
	if len(envFlag.EnumValues) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(envFlag.EnumValues))
	}
	expectedValues := []string{"dev", "stg", "prod"}
	for i, v := range expectedValues {
		if envFlag.EnumValues[i] != v {
			t.Errorf("expected enum value %q at position %d, got %q", v, i, envFlag.EnumValues[i])
		}
	}

	// Проверяем log_format флаг
	logFlag := defs[1]
	if logFlag.Kind != model.FlagKindEnum {
		t.Errorf("expected enum kind, got %v", logFlag.Kind)
	}
	if len(logFlag.EnumValues) != 2 {
		t.Errorf("expected 2 enum values, got %d", len(logFlag.EnumValues))
	}
}

func TestParseFlagsFile_EnumInvalidDefault(t *testing.T) {
	content := `
[flags]
environment = { type = "enum", values = ["dev", "stg"], default = "prod", description = "..." }
`
	path := writeTempFile(t, "flags.toml", content)
	_, err := ParseFlagsFile(path)

	if err == nil {
		t.Fatal("expected error for invalid enum default")
	}

	if !strings.Contains(err.Error(), "не входит в values") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseFlagsFile_EnumMissingValues(t *testing.T) {
	content := `
[flags]
environment = { type = "enum", default = "dev", description = "..." }
`
	path := writeTempFile(t, "flags.toml", content)
	_, err := ParseFlagsFile(path)

	if err == nil {
		t.Fatal("expected error for enum without values")
	}

	if !strings.Contains(err.Error(), "непустой массив values") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseFlagsFile_NonEnumWithValues(t *testing.T) {
	content := `
[flags]
rate_limit = { type = "int", values = ["100", "200"], default = 100, description = "..." }
`
	path := writeTempFile(t, "flags.toml", content)
	_, err := ParseFlagsFile(path)

	if err == nil {
		t.Fatal("expected error for non-enum with values field")
	}

	if !strings.Contains(err.Error(), "только для enum флагов") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
