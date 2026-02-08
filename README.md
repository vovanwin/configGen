# ConfigGen

Генератор типобезопасных конфигураций для Go. Читает TOML файлы и генерирует Go структуры + загрузчик на основе [koanf](https://github.com/knadh/koanf). Поддерживает **feature flags** с динамическими значениями и разными бэкендами.

## Установка

```bash
go install github.com/vovanwin/configgen/cmd/configgen@latest
```

## Как это работает

ConfigGen анализирует ваши TOML файлы конфигурации и генерирует:

- **configgen_config.go** — Go структуры (`Config` и вложенные) с `toml:"..."` тегами
- **configgen_loader.go** — загрузчик на koanf с поддержкой окружений и мержа файлов
- **configgen_flags.go** — `FlagStore` интерфейс + `Flags` struct с типизированными геттерами
- **configgen_flagstore.go** — `MemoryStore` и `FileStore` реализации
- **configgen_flags_test_helpers.go** — `TestFlags()` и `TestFlagsWith()` для тестов

### Логика генерации схемы

Генератор берёт все файлы `config_*.toml` и строит из них схему полей. Режим задаётся флагом `--mode`:

- **intersect** (по умолчанию) — в сгенерированную структуру попадают только поля, которые есть **во всех** файлах конфигурации с одинаковым типом.
- **union** — в структуру попадают **все** поля из всех файлов. При конфликте типов побеждает первый встреченный.

После этого схема мержится с `value.toml` (если есть) через union — поля из обоих источников объединяются.

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
│   ├── config_local.toml    # Локальные переопределения (.gitignore)
│   └── flags.toml           # Feature flags (опционально)
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

**config_prod.toml** — production значения:
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

### 2. Сгенерируйте код

```bash
configgen --configs=./configs --output=./internal/config --package=config
```

### 3. Добавьте зависимости

```bash
go get github.com/knadh/koanf/v2 github.com/knadh/koanf/parsers/toml/v2 github.com/knadh/koanf/providers/file
go get github.com/BurntSushi/toml  # если используются feature flags
```

### 4. Используйте в коде

```go
// Статический конфиг
cfg, err := config.Load(&config.LoadOptions{
    ConfigDir:      "./configs",
    EnableOverride: true,
})
fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)

if config.IsProduction() {
    // ...
}
```

## Feature Flags

### Определение

Создайте `flags.toml` в директории конфигов:

```toml
[flags]
new_catalog_ui = { type = "bool", default = false, description = "Включить новый UI каталога" }
rate_limit = { type = "int", default = 100, description = "Лимит запросов в секунду" }
score_threshold = { type = "float", default = 0.75, description = "Порог релевантности" }
banner_text = { type = "string", default = "Welcome!", description = "Текст баннера" }
```

Поддерживаемые типы: `bool`, `int`, `float`, `string`.

### Использование

```go
// Создание с MemoryStore (разработка/тесты)
store := config.NewMemoryStore(config.DefaultFlagValues())
flags := config.NewFlags(store)

// Типизированные геттеры — дефолты вшиты при генерации
if flags.NewCatalogUi() {
    // новый UI
}
limit := flags.RateLimit() // int

// Создание с FileStore (CI/интеграционные тесты)
store, _ := config.NewFileStore("./flags_override.toml", config.DefaultFlagValues())
flags := config.NewFlags(store)
```

### В тестах

```go
// С дефолтами
flags := config.TestFlags()

// С переопределениями
flags := config.TestFlagsWith(map[string]any{
    "new_catalog_ui": true,
    "rate_limit":     50,
})
```

### Бэкенды FlagStore

| Бэкенд | Назначение | Статус |
|--------|-----------|--------|
| `MemoryStore` | Тесты, локальная разработка. Thread-safe, с `Set()` | Готов |
| `FileStore` | CI/интеграционные тесты. Читает TOML override файл | Готов |
| `EtcdStore` | Production. Watch на изменения | Планируется |

## Порядок загрузки (runtime)

1. **value.toml** — базовые константы (опционально)
2. **config_{env}.toml** — значения окружения (обязательно)
3. **override.toml** — переопределения для текущего env (опционально)

Окружение определяется из `LoadOptions.Environment` или переменной окружения `APP_ENV` (по умолчанию `dev`).

## Сгенерированный API

### Конфигурация

| Функция | Описание |
|---------|----------|
| `Load(opts)` | Загрузить конфигурацию |
| `MustLoad(opts)` | Загрузить или panic |
| `Get()` | Текущий конфиг (thread-safe) |
| `GetAll()` | Конфиги всех окружений |
| `GetEnv()` | Текущее окружение |
| `IsProduction()` | `true` если `prod` |
| `IsStg()` | `true` если `stg` |
| `IsLocal()` | `true` если `local` |

### Feature Flags

| Функция | Описание |
|---------|----------|
| `NewFlags(store)` | Создать Flags с бэкендом |
| `flags.FlagName()` | Типизированный геттер |
| `flags.Store()` | Доступ к бэкенду |
| `DefaultFlagValues()` | Дефолтные значения всех флагов |
| `NewMemoryStore(vals)` | In-memory store |
| `NewFileStore(path, defaults)` | File-based store |
| `TestFlags()` | Flags с дефолтами для тестов |
| `TestFlagsWith(overrides)` | Flags с переопределениями |

## Флаги CLI

```
--configs      Директория с TOML файлами (./configs)
--output       Куда писать сгенерированный код (./internal/config)
--package      Имя Go пакета (config)
--env-prefix   Переменная окружения для определения env (APP_ENV)
--with-loader  Генерировать loader (true)
--with-flags   Генерировать feature flags если flags.toml найден (true)
--mode         Режим схемы: intersect | union (intersect)
--validate     Проверить все TOML без генерации кода
--init         Создать шаблонные конфиг-файлы
```

## Валидация

```bash
# Проверить что все TOML валидны без генерации
configgen --validate --configs=./configs
```

Подходит для CI pipeline и pre-commit hooks.

## Пример

Полный рабочий пример в `example/service/`.

```bash
# Генерация + тесты + сборка
task all

# Запуск примера
task example:run
```
