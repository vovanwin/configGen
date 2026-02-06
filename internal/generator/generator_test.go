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

	// Проверяем что config.gen.go создан
	configPath := filepath.Join(tmpDir, "config.gen.go")
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("не удалось прочитать config.gen.go: %v", err)
	}

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

	if !strings.Contains(configStr, "Env") || !strings.Contains(configStr, "`toml:\"-\"`") {
		t.Error("config.gen.go должен содержать поле Env Environment")
	}

	if !strings.Contains(configStr, "func (c *Config) IsProduction()") {
		t.Error("config.gen.go должен содержать метод IsProduction")
	}

	if !strings.Contains(configStr, "func (c *Config) IsStg()") {
		t.Error("config.gen.go должен содержать метод IsStg")
	}

	if !strings.Contains(configStr, "func (c *Config) IsLocal()") {
		t.Error("config.gen.go должен содержать метод IsLocal")
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

	if !strings.Contains(loaderStr, "koanf") {
		t.Error("loader.gen.go должен использовать koanf")
	}

	if !strings.Contains(loaderStr, "allConfigs") {
		t.Error("loader.gen.go должен хранить все конфиги в map")
	}

	if !strings.Contains(loaderStr, "func GetAll()") {
		t.Error("loader.gen.go должен содержать функцию GetAll")
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

	configPath := filepath.Join(tmpDir, "config.gen.go")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.gen.go должен быть создан")
	}

	loaderPath := filepath.Join(tmpDir, "loader.gen.go")
	if _, err := os.Stat(loaderPath); !os.IsNotExist(err) {
		t.Error("loader.gen.go не должен быть создан когда WithLoader=false")
	}
}
