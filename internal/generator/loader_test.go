package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vovanwin/configgen/internal/model"
)

func TestGenerateLoaderWithEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*model.Field{
		"server": {
			Name:       "Server",
			TOMLName:   "server",
			Kind:       model.KindObject,
			StructName: "Server",
			Children: map[string]*model.Field{
				"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
				"port": {Name: "Port", TOMLName: "port", Kind: model.KindInt},
			},
		},
	}

	opts := Options{
		OutputDir:       tmpDir,
		PackageName:     "testconfig",
		WithLoader:      true,
		EnvPrefix:       "TEST_ENV",
		WithEnvOverride: true,
		EnvVarPrefix:    "APP_",
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Проверяем сгенерированный код
	loaderPath := filepath.Join(tmpDir, "configgen_loader.go")
	content, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_loader.go: %v", err)
	}
	loaderStr := string(content)

	// Проверяем импорт env provider (с алиасом)
	if !strings.Contains(loaderStr, `kenv "github.com/knadh/koanf/providers/env"`) {
		t.Error("env provider import not found")
	}

	// Проверяем EnableEnv в LoadOptions
	if !strings.Contains(loaderStr, "EnableEnv bool") {
		t.Error("EnableEnv field not found in LoadOptions")
	}

	// Проверяем загрузку env vars
	if !strings.Contains(loaderStr, `envPrefix := "APP_"`) {
		t.Error("env prefix not found")
	}

	if !strings.Contains(loaderStr, `kenv.Provider(envPrefix, "."`) {
		t.Error("kenv.Provider not found")
	}

	// Проверяем комментарий с примером (__ как разделитель секций)
	if !strings.Contains(loaderStr, "APP_SERVER__HOST=localhost") {
		t.Error("example comment not found")
	}
}

func TestGenerateLoaderWithoutEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*model.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
	}

	opts := Options{
		OutputDir:       tmpDir,
		PackageName:     "config",
		WithLoader:      true,
		WithEnvOverride: false,
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	loaderPath := filepath.Join(tmpDir, "configgen_loader.go")
	content, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_loader.go: %v", err)
	}
	loaderStr := string(content)

	// Проверяем что env provider НЕ импортируется
	if strings.Contains(loaderStr, `"github.com/knadh/koanf/providers/env"`) {
		t.Error("env provider import should not be present when WithEnvOverride=false")
	}

	// Проверяем что EnableEnv НЕ в LoadOptions
	if strings.Contains(loaderStr, "EnableEnv bool") {
		t.Error("EnableEnv field should not be present when WithEnvOverride=false")
	}
}
