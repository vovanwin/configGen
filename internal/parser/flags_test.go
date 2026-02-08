package parser

import (
	"os"
	"path/filepath"
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

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
