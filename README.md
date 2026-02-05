# ConfigGen

Генератор типобезопасных конфигураций для Go приложений. Создает Go структуры и загрузчики из TOML файлов конфигурации.

## Возможности

- **Типобезопасные конфиги** — генерирует Go структуры из TOML файлов
- **Поддержка окружений** — local, dev, stg, prod
- **Многослойная конфигурация** — базовые значения + окружение + локальные переопределения
- **Поддержка Duration** — автоматический парсинг `"30s"`, `"5m"`, `"1h"`
- **Валидация схемы** — гарантирует консистентность между конфигами окружений
- **Интеграция с Vault** — заготовка для HashiCorp Vault (будущее)
- **RTC** — заготовка для обновления конфигов в реальном времени (будущее)

## Установка

```bash
go install github.com/vovanwin/configgen/cmd/configgen@latest
```

## Быстрый старт

### 1. Создайте структуру конфиг файлов

```
your-service/
├── configs/
│   ├── value.toml         # Константы (общие для всех окружений)
│   ├── config_dev.toml    # Development
│   ├── config_stg.toml    # Staging
│   ├── config_prod.toml   # Production
│   └── config_local.toml  # Локальные переопределения (не коммитить!)
└── internal/
    └── config/
        ├── config.gen.go  # Сгенерированный
        └── loader.gen.go  # Сгенерированный
```

### 2. Создайте конфиги

**value.toml** — константы, общие для всех окружений:
```toml
[app]
name = "my-service"
version = "1.0.0"

[limits]
max_connections = 1000
request_timeout = "30s"

[features]
enable_metrics = true
enable_tracing = true
```

**config_dev.toml** — окружение разработки:
```toml
[server]
host = "localhost"
port = 8080
read_timeout = "5s"
write_timeout = "10s"

[db]
host = "localhost"
port = 5432
name = "myapp_dev"
user = "dev_user"
password = "dev_password"
pool_size = 5

[log]
level = "debug"
format = "text"
```

**config_prod.toml** — продакшн:
```toml
[server]
host = "0.0.0.0"
port = 80
read_timeout = "15s"
write_timeout = "60s"

[db]
host = "db-prod.internal"
port = 5432
name = "myapp_prod"
user = "prod_user"
password = ""  # Используйте Vault
pool_size = 50

[log]
level = "warn"
format = "json"
```

### 3. Сгенерируйте код

Добавьте в main.go вашего сервиса:
```go
//go:generate go run github.com/vovanwin/configgen/cmd/configgen --configs=./configs --output=./internal/config --package=config
```

Запустите:
```bash
go generate ./...
```

### 4. Используйте в приложении

```go
package main

import (
    "fmt"
    "log"
    "your-service/internal/config"
)

func main() {
    // Загрузка конфига на основе переменной окружения APP_ENV
    // Порядок: value.toml -> config_{env}.toml -> config_local.toml (если есть)
    cfg, err := config.Load(&config.LoadOptions{
        ConfigDir:           "./configs",
        EnableLocalOverride: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Типобезопасный доступ к значениям
    fmt.Printf("Сервер: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
    fmt.Printf("БД пул: %d\n", cfg.Db.PoolSize)
    fmt.Printf("Таймаут чтения: %v\n", cfg.Server.ReadTimeout)

    // Проверка окружения
    if config.IsProduction() {
        fmt.Println("Работаем в PRODUCTION")
    }

    // Доступ к константам (из value.toml)
    values := config.GetValue()
    fmt.Printf("Приложение: %s v%s\n", values.App.Name, values.App.Version)

    // Глобальный доступ к конфигу (thread-safe)
    currentCfg := config.Get()
    fmt.Printf("Текущее окружение: %s\n", config.GetEnv())
}
```

## Порядок загрузки конфигурации

1. **value.toml** — базовые константы (опционально)
2. **config_{env}.toml** — значения для конкретного окружения
3. **config_local.toml** — локальные переопределения (опционально, не коммитить)

Более поздние файлы переопределяют более ранние. Окружение определяется переменной `APP_ENV`.

## Флаги CLI

```
--configs      Директория с конфиг файлами (по умолчанию: ./configs)
--output       Директория для сгенерированного кода (по умолчанию: ./internal/config)
--package      Имя пакета (по умолчанию: config)
--env-prefix   Имя переменной окружения (по умолчанию: APP_ENV)
--with-loader  Генерировать loader.gen.go (по умолчанию: true)
--with-vault   Включить заготовку для Vault (по умолчанию: false)
--with-rtc     Включить заготовку для RTC (по умолчанию: false)
--mode         Режим схемы: intersect или union (по умолчанию: intersect)
```

### Режимы схемы

- **intersect** (по умолчанию) — только поля, присутствующие во ВСЕХ конфиг файлах
- **union** — все поля из всех конфигов (при конфликте типов побеждает первый)

## Окружения

| Окружение | APP_ENV | Описание |
|-----------|---------|----------|
| Local | `local` | Локальная разработка |
| Development | `dev` | Сервер разработки |
| Staging | `stg` | Предпродакшн |
| Production | `prod` | Продакшн |

## Локальные переопределения

Создайте `config_local.toml` для персональных настроек:

```toml
# Переопределение БД для локального Docker
[db]
host = "127.0.0.1"
port = 5433

# Включить debug логирование
[log]
level = "debug"
```

Добавьте в `.gitignore`:
```
config_local.toml
```

## Интеграция с Vault (заготовка)

Включите заготовку для Vault:
```bash
configgen --with-vault ...
```

Использование:
```go
cfg, err := config.Load(&config.LoadOptions{
    VaultEnabled: true,
    VaultConfig: &config.VaultConfig{
        Address: "https://vault.internal:8200",
        Token:   os.Getenv("VAULT_TOKEN"),
        Path:    "secret/data/myapp",
    },
})
```

## Real-Time Config (заготовка)

Включите заготовку для RTC:
```bash
configgen --with-rtc ...
```

Использование:
```go
cfg, err := config.Load(&config.LoadOptions{
    RTCEnabled: true,
    RTCConfig: &config.RTCConfig{
        Endpoint:     "https://config-service.internal/api/config",
        PollInterval: "30s",
    },
    OnChange: func(newCfg *config.Config) {
        log.Println("Конфиг обновлен!")
    },
})
```

## Поддерживаемые типы

| Тип TOML | Тип Go |
|----------|--------|
| `"string"` | `string` |
| `123` | `int` |
| `1.5` | `float64` |
| `true/false` | `bool` |
| `"30s"`, `"5m"` | `time.Duration` |
| `[1, 2, 3]` | `[]int` |
| `["a", "b"]` | `[]string` |
| `[section]` | вложенная структура |

## Структура проекта

```
configgen/
├── cmd/configgen/        # CLI утилита
├── internal/
│   ├── generator/        # Генерация кода
│   │   └── templates/    # Go шаблоны
│   ├── parser/           # TOML парсер
│   └── schema/           # Операции со схемой
├── pkg/types/            # Общие типы
└── example/service/      # Пример использования
```

## Сгенерированные файлы

### config.gen.go

Содержит:
- `Config` — основная структура конфигурации
- Вложенные структуры для каждой секции (Server, Db, Log и т.д.)
- `Value` — структура для константных значений

### loader.gen.go

Содержит:
- `Load(opts)` — загрузка конфигурации
- `MustLoad(opts)` — загрузка или panic
- `Get()` — получить текущий конфиг (thread-safe)
- `GetValue()` — получить константы (thread-safe)
- `GetEnv()` — получить текущее окружение
- `IsProduction()` — проверка на продакшн
- `IsDevelopment()` — проверка на dev/local

## Пример

Смотрите директорию `example/service/` для полного примера использования.

```bash
# Сгенерировать конфиг
go run ./cmd/configgen --configs=./example/service/configs --output=./example/service/internal/config --package=config

# Запустить пример
cd example/service
APP_ENV=dev go run .
```

## Лицензия

MIT
