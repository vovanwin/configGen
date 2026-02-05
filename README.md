# ConfigGen

Генератор типобезопасных конфигураций для Go. Читает TOML файлы и генерирует Go структуры + загрузчик на основе [koanf](https://github.com/knadh/koanf).

## Установка

```bash
go install github.com/vovanwin/configgen/cmd/configgen@latest
```

## Как это работает

ConfigGen анализирует ваши TOML файлы конфигурации и генерирует два файла:

- **config.gen.go** — Go структуры (`Config`, `Value` и вложенные) с `toml:"..."` тегами
- **loader.gen.go** — загрузчик на koanf с поддержкой окружений и мержа файлов

### Логика генерации схемы

Генератор берёт все файлы `config_*.toml` (кроме `config_local.toml`) и строит из них схему полей. Режим задаётся флагом `--mode`:

- **intersect** (по умолчанию) — в сгенерированную структуру попадают только поля, которые есть **во всех** файлах конфигурации с одинаковым типом. Если в `config_dev.toml` есть поле `debug = true`, а в `config_prod.toml` его нет — поле не попадёт в структуру.
- **union** — в структуру попадают **все** поля из всех файлов. При конфликте типов побеждает первый встреченный.

После этого схема мержится с `value.toml` (если есть) через union — поля из обоих источников объединяются.

**Когда появляются новые поля в сгенерированных структурах:**
1. Вы добавили новую секцию или поле **во все** `config_*.toml` файлы (при `--mode=intersect`)
2. Вы добавили поле **хотя бы в один** `config_*.toml` (при `--mode=union`)
3. Вы добавили поле в `value.toml`

**Когда поле пропадёт:**
1. Вы удалили его из одного из `config_*.toml` при `--mode=intersect`
2. Вы удалили его из всех `config_*.toml` при `--mode=union`

### Поддерживаемые типы

| TOML | Go | Пример |
|------|-----|--------|
| `"string"` | `string` | `host = "localhost"` |
| `123` | `int` | `port = 8080` |
| `1.5` | `float64` | `ratio = 1.5` |
| `true` / `false` | `bool` | `debug = true` |
| `"30s"`, `"5m"`, `"1h"` | `time.Duration` | `timeout = "30s"` |
| `[1, 2, 3]` | `[]int` | `ports = [80, 443]` |
| `["a", "b"]` | `[]string` | `hosts = ["a", "b"]` |
| `[section]` | вложенная структура | `[server]` -> `Server` |

Комментарии из TOML (`# ...`) переносятся в Go код как комментарии к полям.

## Внедрение в проект

### 1. Создайте конфиги

```
your-service/
├── configs/
│   ├── value.toml           # Константы, общие для всех окружений
│   ├── config_dev.toml      # Development
│   ├── config_stg.toml      # Staging
│   ├── config_prod.toml     # Production
│   └── config_local.toml    # Локальные переопределения (добавьте в .gitignore)
```

**value.toml** — значения, которые не меняются между окружениями:
```toml
[app]
name = "my-service"
version = "1.0.0"

[limits]
max_connections = 1000
request_timeout = "30s"
```

**config_dev.toml**:
```toml
[server]
host = "localhost"
port = 8080
read_timeout = "5s"

[db]
host = "localhost"
port = 5432
name = "myapp_dev"
pool_size = 5
```

**config_prod.toml** — те же секции и поля, но с production значениями:
```toml
[server]
host = "0.0.0.0"
port = 80
read_timeout = "15s"

[db]
host = "db-prod.internal"
port = 5432
name = "myapp_prod"
pool_size = 50
```

### 2. Добавьте go:generate

В `main.go` вашего сервиса:
```go
//go:generate go run github.com/vovanwin/configgen/cmd/configgen --configs=./configs --output=./internal/config --package=config
```

Запустите:
```bash
go generate ./...
```

Это создаст `internal/config/config.gen.go` и `internal/config/loader.gen.go`.

### 3. Добавьте зависимость koanf

Сгенерированный загрузчик использует koanf:
```bash
go get github.com/knadh/koanf/v2 github.com/knadh/koanf/parsers/toml/v2 github.com/knadh/koanf/providers/file
```

### 4. Используйте в коде

```go
package main

import (
    "fmt"
    "log"
    "your-service/internal/config"
)

func main() {
    cfg, err := config.Load(&config.LoadOptions{
        ConfigDir:           "./configs",
        EnableLocalOverride: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Типобезопасный доступ
    fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
    fmt.Printf("DB pool: %d\n", cfg.Db.PoolSize)
    fmt.Printf("Read timeout: %v\n", cfg.Server.ReadTimeout) // time.Duration

    // Константы из value.toml
    values := config.GetValue()
    fmt.Printf("App: %s v%s\n", values.App.Name, values.App.Version)

    // Глобальный доступ (thread-safe)
    currentCfg := config.Get()
    _ = currentCfg

    // Проверка окружения
    if config.IsProduction() {
        // ...
    }
}
```

## Порядок загрузки (runtime)

1. **value.toml** — базовые константы (опционально)
2. **config_{env}.toml** — значения окружения (обязательно)
3. **config_local.toml** — локальные переопределения (опционально)

Каждый следующий файл мержится поверх предыдущего. Окружение определяется из `LoadOptions.Environment` или переменной окружения `APP_ENV` (по умолчанию `dev`).

## Сгенерированный API

| Функция          | Описание                              |
|------------------|---------------------------------------|
| `Load(opts)`     | Загрузить конфигурацию                |
| `MustLoad(opts)` | Загрузить или panic                   |
| `Get()`          | Текущий конфиг (thread-safe)          |
| `GetValue()`     | Константы из value.toml (thread-safe) |
| `GetEnv()`       | Текущее окружение                     |
| `IsProduction()` | `true` если `prod`                    |
| `IsStg()`        | `true` если `stg`                     |
| `IsLocal()`      | `true` если `local`                   |

## Флаги CLI

```
--configs      Директория с TOML файлами (./configs)
--output       Куда писать сгенерированный код (./internal/config)
--package      Имя Go пакета (config)
--env-prefix   Переменная окружения для определения env (APP_ENV)
--with-loader  Генерировать loader.gen.go (true)
--mode         Режим схемы: intersect | union (intersect)
```

## Пример

Полный рабочий пример в `example/service/`.

```bash
# Генерация + тесты + сборка
task all

# Запуск примера
task example:run
```
