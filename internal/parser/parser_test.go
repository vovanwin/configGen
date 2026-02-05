package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vovanwin/configgen/internal/model"
)

func TestToGoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pool_size", "PoolSize"},
		{"host", "Host"},
		{"max_idle_time", "MaxIdleTime"},
		{"enableMetrics", "EnableMetrics"},
		{"read-timeout", "ReadTimeout"},
		{"APP_ENV", "APPENV"},
		{"simple", "Simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToGoName(tt.input)
			if result != tt.expected {
				t.Errorf("ToGoName(%q) = %q, ожидалось %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsDurationSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"30s", true},
		{"5m", true},
		{"1h", true},
		{"100ms", true},
		{"hello", false},
		{"123", false},
		{"", false},
		{"s", false}, // слишком короткое
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsDurationSuffix(tt.input)
			if result != tt.expected {
				t.Errorf("containsDurationSuffix(%q) = %v, ожидалось %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	// Создаем временный файл
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.toml")

	content := `
[server]
host = "localhost"
port = 8080
timeout = "30s"

[db]
pool_size = 10
enabled = true
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}

	fields, err := ParseFile(configPath)
	if err != nil {
		t.Fatalf("ParseFile вернул ошибку: %v", err)
	}

	// Проверяем server секцию
	server, ok := fields["server"]
	if !ok {
		t.Fatal("секция 'server' не найдена")
	}
	if server.Kind != model.KindObject {
		t.Errorf("server.Kind = %v, ожидалось KindObject", server.Kind)
	}
	if server.Name != "Server" {
		t.Errorf("server.Name = %q, ожидалось %q", server.Name, "Server")
	}

	// Проверяем поля server
	host, ok := server.Children["host"]
	if !ok {
		t.Fatal("поле 'host' не найдено")
	}
	if host.Kind != model.KindString {
		t.Errorf("host.Kind = %v, ожидалось KindString", host.Kind)
	}

	port, ok := server.Children["port"]
	if !ok {
		t.Fatal("поле 'port' не найдено")
	}
	if port.Kind != model.KindInt {
		t.Errorf("port.Kind = %v, ожидалось KindInt", port.Kind)
	}

	timeout, ok := server.Children["timeout"]
	if !ok {
		t.Fatal("поле 'timeout' не найдено")
	}
	if timeout.Kind != model.KindDuration {
		t.Errorf("timeout.Kind = %v, ожидалось KindDuration", timeout.Kind)
	}

	// Проверяем db секцию
	db, ok := fields["db"]
	if !ok {
		t.Fatal("секция 'db' не найдена")
	}

	poolSize, ok := db.Children["pool_size"]
	if !ok {
		t.Fatal("поле 'pool_size' не найдено")
	}
	if poolSize.Name != "PoolSize" {
		t.Errorf("poolSize.Name = %q, ожидалось %q", poolSize.Name, "PoolSize")
	}

	enabled, ok := db.Children["enabled"]
	if !ok {
		t.Fatal("поле 'enabled' не найдено")
	}
	if enabled.Kind != model.KindBool {
		t.Errorf("enabled.Kind = %v, ожидалось KindBool", enabled.Kind)
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("/несуществующий/путь/config.toml")
	if err == nil {
		t.Error("ожидалась ошибка для несуществующего файла")
	}
}

func TestParseFileInvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	content := `это не валидный TOML [[[`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("не удалось создать тестовый файл: %v", err)
	}

	_, err := ParseFile(configPath)
	if err == nil {
		t.Error("ожидалась ошибка для невалидного TOML")
	}
}
