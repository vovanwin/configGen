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
		{&model.Field{Kind: model.KindObject, TOMLName: "server", StructName: "Server"}, "Server"},
		{&model.Field{Kind: model.KindObject, TOMLName: "my_config", StructName: "MyConfig"}, "MyConfig"},
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
			Name:       "Server",
			TOMLName:   "server",
			Kind:       model.KindObject,
			StructName: "Server",
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

func TestGenerateWithFlags(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*model.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
	}

	flagDefs := []*model.FlagDef{
		{Name: "NewCatalogUi", TOMLName: "new_catalog_ui", Kind: model.FlagKindBool, Default: false, Description: "Новый UI каталога"},
		{Name: "RateLimit", TOMLName: "rate_limit", Kind: model.FlagKindInt, Default: 100, Description: "Rate limit"},
		{Name: "Threshold", TOMLName: "threshold", Kind: model.FlagKindFloat, Default: 0.5, Description: "Порог"},
		{Name: "BannerText", TOMLName: "banner_text", Kind: model.FlagKindString, Default: "hello", Description: "Текст баннера"},
	}

	opts := Options{
		OutputDir:   tmpDir,
		PackageName: "testconfig",
		WithLoader:  false,
		WithFlags:   true,
		FlagDefs:    flagDefs,
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate вернул ошибку: %v", err)
	}

	// Проверяем configgen_flags.go
	flagsPath := filepath.Join(tmpDir, "configgen_flags.go")
	flagsContent, err := os.ReadFile(flagsPath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_flags.go: %v", err)
	}
	flagsStr := string(flagsContent)

	if !strings.Contains(flagsStr, "type FlagStore interface") {
		t.Error("configgen_flags.go должен содержать интерфейс FlagStore")
	}
	if !strings.Contains(flagsStr, "type Flags struct") {
		t.Error("configgen_flags.go должен содержать структуру Flags")
	}
	if !strings.Contains(flagsStr, "func (f *Flags) NewCatalogUi() bool") {
		t.Error("configgen_flags.go должен содержать геттер NewCatalogUi")
	}
	if !strings.Contains(flagsStr, "func (f *Flags) RateLimit() int") {
		t.Error("configgen_flags.go должен содержать геттер RateLimit")
	}
	if !strings.Contains(flagsStr, "func (f *Flags) Threshold() float64") {
		t.Error("configgen_flags.go должен содержать геттер Threshold")
	}
	if !strings.Contains(flagsStr, "func (f *Flags) BannerText() string") {
		t.Error("configgen_flags.go должен содержать геттер BannerText")
	}
	if !strings.Contains(flagsStr, "func DefaultFlagValues()") {
		t.Error("configgen_flags.go должен содержать DefaultFlagValues")
	}
	if !strings.Contains(flagsStr, `"new_catalog_ui": false`) {
		t.Error("configgen_flags.go должен содержать дефолт для new_catalog_ui")
	}
	if !strings.Contains(flagsStr, `"rate_limit":     100`) {
		t.Error("configgen_flags.go должен содержать дефолт для rate_limit")
	}

	// Проверяем configgen_flagstore.go
	storePath := filepath.Join(tmpDir, "configgen_flagstore.go")
	storeContent, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_flagstore.go: %v", err)
	}
	storeStr := string(storeContent)

	if !strings.Contains(storeStr, "type MemoryStore struct") {
		t.Error("configgen_flagstore.go должен содержать MemoryStore")
	}
	if !strings.Contains(storeStr, "type FileStore struct") {
		t.Error("configgen_flagstore.go должен содержать FileStore")
	}
	if !strings.Contains(storeStr, "func (s *MemoryStore) Set(") {
		t.Error("configgen_flagstore.go должен содержать метод Set для MemoryStore")
	}

	// Проверяем configgen_flags_test_helpers.go
	helpersPath := filepath.Join(tmpDir, "configgen_flags_test_helpers.go")
	helpersContent, err := os.ReadFile(helpersPath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_flags_test_helpers.go: %v", err)
	}
	helpersStr := string(helpersContent)

	if !strings.Contains(helpersStr, "//go:build !production") {
		t.Error("configgen_flags_test_helpers.go должен содержать build tag")
	}
	if !strings.Contains(helpersStr, "func TestFlags()") {
		t.Error("configgen_flags_test_helpers.go должен содержать TestFlags")
	}
	if !strings.Contains(helpersStr, "func TestFlagsWith(") {
		t.Error("configgen_flags_test_helpers.go должен содержать TestFlagsWith")
	}
}

func TestGenerateWithoutFlags(t *testing.T) {
	tmpDir := t.TempDir()

	fields := map[string]*model.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
	}

	opts := Options{
		OutputDir:   tmpDir,
		PackageName: "config",
		WithLoader:  false,
		WithFlags:   false,
	}

	err := Generate(opts, fields)
	if err != nil {
		t.Fatalf("Generate вернул ошибку: %v", err)
	}

	flagsPath := filepath.Join(tmpDir, "configgen_flags.go")
	if _, err := os.Stat(flagsPath); !os.IsNotExist(err) {
		t.Error("configgen_flags.go не должен быть создан когда WithFlags=false")
	}

	storePath := filepath.Join(tmpDir, "configgen_flagstore.go")
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Error("configgen_flagstore.go не должен быть создан когда WithFlags=false")
	}
}

func TestFlagDefaultLiteral(t *testing.T) {
	tests := []struct {
		val      any
		kind     model.FlagKind
		expected string
	}{
		{false, model.FlagKindBool, "false"},
		{true, model.FlagKindBool, "true"},
		{100, model.FlagKindInt, "100"},
		{0, model.FlagKindInt, "0"},
		{0.5, model.FlagKindFloat, "0.5"},
		{0.0, model.FlagKindFloat, "0.0"},
		{1.0, model.FlagKindFloat, "1.0"},
		{"hello", model.FlagKindString, `"hello"`},
		{"", model.FlagKindString, `""`},
	}

	for _, tt := range tests {
		result := flagDefaultLiteral(tt.val, tt.kind)
		if result != tt.expected {
			t.Errorf("flagDefaultLiteral(%v, %v) = %q, ожидалось %q", tt.val, tt.kind, result, tt.expected)
		}
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

func TestGenerateWithEnumFlags(t *testing.T) {
	tmpDir := t.TempDir()

	flagDefs := []*model.FlagDef{
		{
			Name:        "Environment",
			TOMLName:    "environment",
			Kind:        model.FlagKindEnum,
			Default:     "dev",
			Description: "Deployment environment",
			EnumValues:  []string{"dev", "stg", "prod"},
		},
		{
			Name:        "LogFormat",
			TOMLName:    "log_format",
			Kind:        model.FlagKindEnum,
			Default:     "json",
			Description: "Log output format",
			EnumValues:  []string{"json", "text"},
		},
	}

	opts := Options{
		OutputDir:   tmpDir,
		PackageName: "testconfig",
		WithFlags:   true,
		FlagDefs:    flagDefs,
	}

	err := Generate(opts, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Проверяем сгенерированный код
	flagsPath := filepath.Join(tmpDir, "configgen_flags.go")
	content, err := os.ReadFile(flagsPath)
	if err != nil {
		t.Fatalf("не удалось прочитать configgen_flags.go: %v", err)
	}
	flagsStr := string(content)

	// Проверяем enum типы
	if !strings.Contains(flagsStr, "type EnvironmentEnum string") {
		t.Error("enum type EnvironmentEnum not generated")
	}
	if !strings.Contains(flagsStr, "type LogFormatEnum string") {
		t.Error("enum type LogFormatEnum not generated")
	}

	// Проверяем константы (go fmt может выравнивать пробелы)
	if !strings.Contains(flagsStr, `EnvironmentEnumDev`) || !strings.Contains(flagsStr, `= "dev"`) {
		t.Error("enum const EnvironmentEnumDev not generated")
	}
	if !strings.Contains(flagsStr, `EnvironmentEnumStg`) || !strings.Contains(flagsStr, `= "stg"`) {
		t.Error("enum const EnvironmentEnumStg not generated")
	}
	if !strings.Contains(flagsStr, `LogFormatEnumJson`) || !strings.Contains(flagsStr, `= "json"`) {
		t.Error("enum const LogFormatEnumJson not generated")
	}

	// Проверяем IsValid
	if !strings.Contains(flagsStr, "func (e EnvironmentEnum) IsValid() bool") {
		t.Error("IsValid for EnvironmentEnum not generated")
	}

	// Проверяем геттеры возвращают enum тип
	if !strings.Contains(flagsStr, "func (f *Flags) Environment() EnvironmentEnum") {
		t.Error("enum getter Environment not generated")
	}
	if !strings.Contains(flagsStr, "func (f *Flags) LogFormat() LogFormatEnum") {
		t.Error("enum getter LogFormat not generated")
	}

	// Проверяем использование GetString внутри геттера
	if !strings.Contains(flagsStr, `GetString("environment"`) {
		t.Error("enum store method not correct for Environment")
	}
	if !strings.Contains(flagsStr, `GetString("log_format"`) {
		t.Error("enum store method not correct for LogFormat")
	}

	// Проверяем дефолты в DefaultFlagValues
	if !strings.Contains(flagsStr, `"environment"`) {
		t.Error("default value for environment not in DefaultFlagValues")
	}
	if !strings.Contains(flagsStr, `"log_format"`) {
		t.Error("default value for log_format not in DefaultFlagValues")
	}
}
