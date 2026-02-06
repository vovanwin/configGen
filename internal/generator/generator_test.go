package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vovanwin/configgen/internal/model"
)

func TestGoType(t *testing.T) {
	tests := []struct {
		field    *model.Field
		expected string
	}{
		{&model.Field{Kind: model.KindString}, "string"},
		{&model.Field{Kind: model.KindInt}, "int"},
		{&model.Field{Kind: model.KindFloat}, "float64"},
		{&model.Field{Kind: model.KindBool}, "bool"},
		{&model.Field{Kind: model.KindDuration}, "time.Duration"},
		{&model.Field{Kind: model.KindSlice, ItemKind: model.KindString}, "[]string"},
		{&model.Field{Kind: model.KindSlice, ItemKind: model.KindInt}, "[]int"},
		{&model.Field{Kind: model.KindObject, TOMLName: "server"}, "Server"},
		{&model.Field{Kind: model.KindObject, TOMLName: "my_config"}, "MyConfig"},
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
	fieldsNoDuration := map[string]*model.Field{
		"host": {Kind: model.KindString},
		"port": {Kind: model.KindInt},
	}

	if needsTime(fieldsNoDuration) {
		t.Error("needsTime должен вернуть false когда нет Duration полей")
	}

	fieldsWithDuration := map[string]*model.Field{
		"host":    {Kind: model.KindString},
		"timeout": {Kind: model.KindDuration},
	}

	if !needsTime(fieldsWithDuration) {
		t.Error("needsTime должен вернуть true когда есть Duration поле")
	}

	fieldsNestedDuration := map[string]*model.Field{
		"server": {
			Kind: model.KindObject,
			Children: map[string]*model.Field{
				"timeout": {Kind: model.KindDuration},
			},
		},
	}

	if !needsTime(fieldsNestedDuration) {
		t.Error("needsTime должен вернуть true когда Duration во вложенном объекте")
	}
}

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*model.Field{
		"server": {
			Name:     "Server",
			TOMLName: "server",
			Kind:     model.KindObject,
			Children: map[string]*model.Field{
				"host":    {Name: "Host", TOMLName: "host", Kind: model.KindString},
				"port":    {Name: "Port", TOMLName: "port", Kind: model.KindInt},
				"timeout": {Name: "Timeout", TOMLName: "timeout", Kind: model.KindDuration},
			},
		},
		"debug": {Name: "Debug", TOMLName: "debug", Kind: model.KindBool},
	}

	opts := Options{
		OutputDir:   tmpDir,
		PackageName: "testconfig",
		EnvPrefix:   "TEST_ENV",
		WithLoader:  true,
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate вернул ошибку: %v", err)
	}

	// Проверяем что configgen_config.go создан
	configPath := filepath.Join(tmpDir, "configgen_config.go")
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_config.go: %v", err)
	}

	configStr := string(configContent)

	if !strings.Contains(configStr, "package testconfig") {
		t.Error("configgen_config.go должен содержать правильное имя пакета")
	}

	if !strings.Contains(configStr, "type Config struct") {
		t.Error("configgen_config.go должен содержать структуру Config")
	}

	if !strings.Contains(configStr, "type Server struct") {
		t.Error("configgen_config.go должен содержать структуру Server")
	}

	if !strings.Contains(configStr, "time.Duration") {
		t.Error("configgen_config.go должен импортировать time для Duration")
	}

	if !strings.Contains(configStr, `toml:"host"`) {
		t.Error("configgen_config.go должен содержать TOML теги")
	}

	if !strings.Contains(configStr, "Env") || !strings.Contains(configStr, "`toml:\"-\"`") {
		t.Error("configgen_config.go должен содержать поле Env Environment")
	}

	if !strings.Contains(configStr, "func (c *Config) IsProduction()") {
		t.Error("configgen_config.go должен содержать метод IsProduction")
	}

	if !strings.Contains(configStr, "func (c *Config) IsStg()") {
		t.Error("configgen_config.go должен содержать метод IsStg")
	}

	if !strings.Contains(configStr, "func (c *Config) IsLocal()") {
		t.Error("configgen_config.go должен содержать метод IsLocal")
	}

	// Проверяем что configgen_loader.go создан
	loaderPath := filepath.Join(tmpDir, "configgen_loader.go")
	loaderContent, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_loader.go: %v", err)
	}

	loaderStr := string(loaderContent)

	if !strings.Contains(loaderStr, "func Load(") {
		t.Error("configgen_loader.go должен содержать функцию Load")
	}

	if !strings.Contains(loaderStr, "func MustLoad(") {
		t.Error("configgen_loader.go должен содержать функцию MustLoad")
	}

	if !strings.Contains(loaderStr, "TEST_ENV") {
		t.Error("configgen_loader.go должен использовать правильный EnvPrefix")
	}

	if !strings.Contains(loaderStr, "koanf") {
		t.Error("configgen_loader.go должен использовать koanf")
	}

	if !strings.Contains(loaderStr, "allConfigs") {
		t.Error("configgen_loader.go должен хранить все конфиги в map")
	}

	if !strings.Contains(loaderStr, "func GetAll()") {
		t.Error("configgen_loader.go должен содержать функцию GetAll")
	}
}

func TestGenerateWithoutLoader(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*model.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
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

	configPath := filepath.Join(tmpDir, "configgen_config.go")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("configgen_config.go должен быть создан")
	}

	loaderPath := filepath.Join(tmpDir, "configgen_loader.go")
	if _, err := os.Stat(loaderPath); !os.IsNotExist(err) {
		t.Error("configgen_loader.go не должен быть создан когда WithLoader=false")
	}
}
