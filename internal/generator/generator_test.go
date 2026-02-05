package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vovanwin/configgen/pkg/types"
)

func TestGoType(t *testing.T) {
	tests := []struct {
		field    *types.Field
		expected string
	}{
		{&types.Field{Kind: types.KindString}, "string"},
		{&types.Field{Kind: types.KindInt}, "int"},
		{&types.Field{Kind: types.KindFloat}, "float64"},
		{&types.Field{Kind: types.KindBool}, "bool"},
		{&types.Field{Kind: types.KindDuration}, "time.Duration"},
		{&types.Field{Kind: types.KindSlice, ItemKind: types.KindString}, "[]string"},
		{&types.Field{Kind: types.KindSlice, ItemKind: types.KindInt}, "[]int"},
		{&types.Field{Kind: types.KindObject, TOMLName: "server"}, "Server"},
		{&types.Field{Kind: types.KindObject, TOMLName: "my_config"}, "MyConfig"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := goType(tt.field)
			if result != tt.expected {
				t.Errorf("goType() = %q, ожидалось %q", result, tt.expected)
			}
		})
	}
}

func TestToGoStructName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"server", "Server"},
		{"my_config", "MyConfig"},
		{"database-settings", "DatabaseSettings"},
		{"APP", "APP"},
		{"api_v2", "ApiV2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toGoStructName(tt.input)
			if result != tt.expected {
				t.Errorf("toGoStructName(%q) = %q, ожидалось %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]int{
		"zebra": 1,
		"apple": 2,
		"mango": 3,
	}

	keys := sortedKeys(m)

	expected := []string{"apple", "mango", "zebra"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("sortedKeys()[%d] = %q, ожидалось %q", i, k, expected[i])
		}
	}
}

func TestNeedsTime(t *testing.T) {
	// Без Duration
	fieldsNoDuration := map[string]*types.Field{
		"host": {Kind: types.KindString},
		"port": {Kind: types.KindInt},
	}

	if needsTime(fieldsNoDuration) {
		t.Error("needsTime должен вернуть false когда нет Duration полей")
	}

	// С Duration
	fieldsWithDuration := map[string]*types.Field{
		"host":    {Kind: types.KindString},
		"timeout": {Kind: types.KindDuration},
	}

	if !needsTime(fieldsWithDuration) {
		t.Error("needsTime должен вернуть true когда есть Duration поле")
	}

	// Вложенный Duration
	fieldsNestedDuration := map[string]*types.Field{
		"server": {
			Kind: types.KindObject,
			Children: map[string]*types.Field{
				"timeout": {Kind: types.KindDuration},
			},
		},
	}

	if !needsTime(fieldsNestedDuration) {
		t.Error("needsTime должен вернуть true когда Duration во вложенном объекте")
	}
}

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*types.Field{
		"server": {
			Name:     "Server",
			TOMLName: "server",
			Kind:     types.KindObject,
			Children: map[string]*types.Field{
				"host":    {Name: "Host", TOMLName: "host", Kind: types.KindString},
				"port":    {Name: "Port", TOMLName: "port", Kind: types.KindInt},
				"timeout": {Name: "Timeout", TOMLName: "timeout", Kind: types.KindDuration},
			},
		},
		"debug": {Name: "Debug", TOMLName: "debug", Kind: types.KindBool},
	}

	opts := Options{
		OutputDir:   tmpDir,
		PackageName: "testconfig",
		EnvPrefix:   "TEST_ENV",
		WithLoader:  true,
		WithVault:   false,
		WithRTC:     false,
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate вернул ошибку: %v", err)
	}

	// Проверяем что config.gen.go создан
	configPath := filepath.Join(tmpDir, "config.gen.go")
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("не удалось прочитать config.gen.go: %v", err)
	}

	// Проверяем содержимое
	configStr := string(configContent)

	if !strings.Contains(configStr, "package testconfig") {
		t.Error("config.gen.go должен содержать правильное имя пакета")
	}

	if !strings.Contains(configStr, "type Config struct") {
		t.Error("config.gen.go должен содержать структуру Config")
	}

	if !strings.Contains(configStr, "type Server struct") {
		t.Error("config.gen.go должен содержать структуру Server")
	}

	if !strings.Contains(configStr, "time.Duration") {
		t.Error("config.gen.go должен импортировать time для Duration")
	}

	if !strings.Contains(configStr, `toml:"host"`) {
		t.Error("config.gen.go должен содержать TOML теги")
	}

	// Проверяем что loader.gen.go создан
	loaderPath := filepath.Join(tmpDir, "loader.gen.go")
	loaderContent, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("не удалось прочитать loader.gen.go: %v", err)
	}

	loaderStr := string(loaderContent)

	if !strings.Contains(loaderStr, "func Load(") {
		t.Error("loader.gen.go должен содержать функцию Load")
	}

	if !strings.Contains(loaderStr, "func MustLoad(") {
		t.Error("loader.gen.go должен содержать функцию MustLoad")
	}

	if !strings.Contains(loaderStr, "TEST_ENV") {
		t.Error("loader.gen.go должен использовать правильный EnvPrefix")
	}
}

func TestGenerateWithoutLoader(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*types.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: types.KindString},
	}

	opts := Options{
		OutputDir:   tmpDir,
		PackageName: "config",
		WithLoader:  false,
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate вернул ошибку: %v", err)
	}

	// config.gen.go должен существовать
	configPath := filepath.Join(tmpDir, "config.gen.go")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.gen.go должен быть создан")
	}

	// loader.gen.go НЕ должен существовать
	loaderPath := filepath.Join(tmpDir, "loader.gen.go")
	if _, err := os.Stat(loaderPath); !os.IsNotExist(err) {
		t.Error("loader.gen.go не должен быть создан когда WithLoader=false")
	}
}

func TestGenerateWithVaultAndRTC(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*types.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: types.KindString},
	}

	opts := Options{
		OutputDir:   tmpDir,
		PackageName: "config",
		EnvPrefix:   "APP_ENV",
		WithLoader:  true,
		WithVault:   true,
		WithRTC:     true,
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate вернул ошибку: %v", err)
	}

	loaderPath := filepath.Join(tmpDir, "loader.gen.go")
	loaderContent, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("не удалось прочитать loader.gen.go: %v", err)
	}

	loaderStr := string(loaderContent)

	if !strings.Contains(loaderStr, "VaultConfig") {
		t.Error("loader.gen.go должен содержать VaultConfig когда WithVault=true")
	}

	if !strings.Contains(loaderStr, "RTCConfig") {
		t.Error("loader.gen.go должен содержать RTCConfig когда WithRTC=true")
	}
}
