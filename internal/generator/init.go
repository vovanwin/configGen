package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

var initFiles = map[string]string{
	"value.toml": `# value.toml — базовые константы, общие для всех окружений
# Эти значения не меняются между окружениями

[app]
name = "my-service"
version = "0.1.0"
`,

	"config_prod.toml": `# config_prod.toml — production конфигурация

[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"

[db]
host = "localhost"
port = 5432
name = "mydb"
user = "app"
password = "secret"
max_open_conns = 25
max_idle_conns = 5

[redis]
addr = "localhost:6379"
password = ""
db = 0

[log]
level = "info"
format = "json"
`,

	"config_local.toml.example": `# config_local.toml.example — скопируйте в config_local.toml и настройте под себя
# config_local.toml добавлен в .gitignore

[server]
host = "127.0.0.1"
port = 8080
read_timeout = "5s"
write_timeout = "5s"

[db]
host = "localhost"
port = 5432
name = "mydb_dev"
user = "postgres"
password = "postgres"
max_open_conns = 5
max_idle_conns = 2

[redis]
addr = "localhost:6379"
password = ""
db = 0

[log]
level = "debug"
format = "text"
`,
}

// Init создаёт начальные конфиг-файлы в указанной директории
func Init(configsDir string) error {
	if err := os.MkdirAll(configsDir, 0o755); err != nil {
		return fmt.Errorf("создание директории %s: %w", configsDir, err)
	}

	for name, content := range initFiles {
		path := filepath.Join(configsDir, name)
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  skip: %s (already exists)\n", name)
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("запись %s: %w", name, err)
		}
		fmt.Printf("  created: %s\n", name)
	}

	return nil
}
