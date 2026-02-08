# ConfigGen ROADMAP

## Текущее состояние

ConfigGen — генератор типобезопасных Go-конфигов из TOML. Генерирует структуры + загрузчик на koanf.
Работает хорошо для **статических** конфигов, которые читаются при старте и не меняются.

---

## Фаза 1: Feature Flags (приоритет: высокий)

Основная идея: feature flags — это **динамические** значения, которые можно менять в runtime без рестарта.
Статические конфиги (порты, DSN) остаются как есть. Feature flags — отдельная подсистема.

### 1.1 Определение флагов в TOML

Новый файл `flags.toml` рядом с конфигами:

```toml
[flags]
# Включить новый UI каталога
new_catalog_ui = { type = "bool", default = false, description = "Новый UI каталога" }

# Лимит запросов в минуту
rate_limit = { type = "int", default = 100, description = "Rate limit (req/min)" }

# Процент трафика на новый алгоритм
new_algo_percent = { type = "float", default = 0.0, description = "% трафика на новый алгоритм" }

# Сообщение для баннера
banner_message = { type = "string", default = "", description = "Текст баннера, пусто = скрыт" }
```

### 1.2 Генерация кода для флагов

configgen генерирует `configgen_flags.go`:

```go
// Типизированные геттеры с дефолтами, вшитыми при генерации
type Flags struct {
    store FlagStore
}

func (f *Flags) NewCatalogUI() bool    { return f.store.GetBool("new_catalog_ui", false) }
func (f *Flags) RateLimit() int        { return f.store.GetInt("rate_limit", 100) }
func (f *Flags) NewAlgoPercent() float64 { return f.store.GetFloat("new_algo_percent", 0.0) }
func (f *Flags) BannerMessage() string { return f.store.GetString("banner_message", "") }

// FlagStore — интерфейс бэкенда (etcd, memory, file)
type FlagStore interface {
    GetBool(key string, def bool) bool
    GetInt(key string, def int) int
    GetFloat(key string, def float64) float64
    GetString(key string, def string) string
    Watch(ctx context.Context, onChange func(key string)) error
    Close() error
}
```

Дефолты **вшиваются в сгенерированный код** при генерации — если бэкенд недоступен, флаги возвращают безопасные значения.

### 1.3 Бэкенды FlagStore

| Бэкенд | Назначение |
|--------|-----------|
| `MemoryStore` | Тесты, локальная разработка. Значения из `flags.toml` |
| `EtcdStore` | Production. Watch на изменения, instant обновление |
| `FileStore` | CI/интеграционные тесты. Читает `flags_override.toml` |

Бэкенд выбирается при инициализации, а не при генерации — сгенерированный код одинаковый.

### 1.4 Интеграция с etcd

Новый пакет `github.com/vovanwin/configgen/flagstore/etcd`:

```go
store, err := etcd.NewStore(etcd.Config{
    Endpoints: []string{"localhost:2379"},
    Prefix:    "/myservice/flags/",
    TTL:       0, // без TTL — ключи живут пока не удалят
})

flags := config.NewFlags(store)
defer flags.Close()

// Подписка на изменения
go flags.Watch(ctx, func(key string) {
    log.Info("flag changed", "key", key)
})
```

Формат ключей в etcd: `/myservice/flags/new_catalog_ui` → `"true"`.
Простые строковые значения — легко менять через `etcdctl` или UI.

---

## Фаза 2: Тестирование конфигов и флагов (приоритет: высокий)

### 2.1 Тестовые хелперы (генерируемые)

configgen генерирует `configgen_test_helpers.go` (build tag `//go:build !production`):

```go
// TestConfig возвращает Config с разумными дефолтами для тестов
func TestConfig() *Config {
    return &Config{
        Env: EnvLocal,
        Server: Server{Host: "localhost", Port: 0, ...}, // port=0 → свободный
        Db: Db{Host: "localhost", Port: 5432, ...},
    }
}

// TestFlags возвращает Flags с MemoryStore и дефолтами из flags.toml
func TestFlags() *Flags {
    return NewFlags(NewMemoryStore(DefaultFlagValues()))
}

// WithFlag переопределяет один флаг для теста
func TestFlagsWith(overrides map[string]any) *Flags {
    store := NewMemoryStore(DefaultFlagValues())
    for k, v := range overrides {
        store.Set(k, v)
    }
    return NewFlags(store)
}
```

### 2.2 Валидация конфигов

Новый CLI-флаг `--validate`:

```bash
configgen --validate --configs=./configs
```

Проверяет:
- Все `config_*.toml` парсятся без ошибок
- В `intersect` режиме: все файлы содержат одинаковые поля
- Типы совпадают между файлами
- Duration-строки валидны (`"30s"` — ок, `"30x"` — ошибка)
- `flags.toml` — все флаги имеют `type`, `default`, допустимый тип

Подходит для CI pipeline: `configgen --validate` в pre-commit или CI.

### 2.3 Интеграционные тесты

Паттерн для интеграционных тестов с etcd:

```go
func TestWithEtcd(t *testing.T) {
    // testcontainers или встроенный etcd
    store := etcd.NewStore(etcd.Config{Endpoints: []string{etcdAddr}})
    flags := config.NewFlags(store)

    // Выставляем флаг через etcd клиент
    etcdClient.Put(ctx, "/myservice/flags/new_catalog_ui", "true")

    // Ждём propagation
    assert.Eventually(t, func() bool {
        return flags.NewCatalogUI() == true
    }, 5*time.Second, 100*time.Millisecond)
}
```

`FileStore` для интеграционных тестов без etcd:

```go
func TestWithFileFlags(t *testing.T) {
    // Создаём временный flags_override.toml
    tmpFile := writeTempFlags(t, map[string]any{"rate_limit": 50})
    store := file.NewStore(tmpFile)
    flags := config.NewFlags(store)

    assert.Equal(t, 50, flags.RateLimit())
}
```

---

## Фаза 3: Улучшения статических конфигов (приоритет: средний)

### 3.1 Переопределение через env vars

Поддержка `ENV_VAR` override для любого поля конфига:

```toml
[server]
host = "localhost"       # env: SERVER_HOST
port = 8080              # env: SERVER_PORT
```

В сгенерированном загрузчике:
```go
// После загрузки TOML — перезаписываем из env vars
k.Load(env.Provider("MYSERVICE_", ".", func(s string) string {
    return strings.Replace(strings.ToLower(s), "_", ".", -1)
}), nil)
```

Это стандартный паттерн для Docker/K8s, где секреты приходят через env vars.

### 3.2 Валидация значений (struct tags)

Поддержка validation rules в комментариях TOML:

```toml
[server]
port = 8080              # validate: min=1,max=65535
host = "localhost"       # validate: required
read_timeout = "30s"     # validate: min=1s,max=5m
```

Генерирует метод `Validate() error` на Config:

```go
func (c *Config) Validate() error {
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        return fmt.Errorf("server.port: must be between 1 and 65535, got %d", c.Server.Port)
    }
    if c.Server.Host == "" {
        return fmt.Errorf("server.host: required")
    }
    // ...
    return nil
}
```

### 3.3 Вложенные секции (глубина > 1)

Сейчас поддерживается один уровень вложенности (`[server]`). Добавить поддержку:

```toml
[server.tls]
cert_file = "/path/to/cert"
key_file = "/path/to/key"
```

Генерирует:
```go
type Server struct {
    Host string `toml:"host"`
    Tls  ServerTls `toml:"tls"`
}

type ServerTls struct {
    CertFile string `toml:"cert_file"`
    KeyFile  string `toml:"key_file"`
}
```

---

## Фаза 4: Config Service (приоритет: низкий, требует проектирования)

### Архитектура

Отдельный сервис не нужен для MVP. Вот почему:

**etcd — уже является сервисом** для хранения и watch. Для управления флагами достаточно:
1. `etcdctl` — для DevOps
2. Простой Web UI — для менеджеров/QA

Полноценный Config Service имеет смысл когда:
- Несколько сервисов шарят одни флаги
- Нужен audit log (кто, когда, что поменял)
- Нужны правила раскатки (% трафика, canary)
- Нужна авторизация (кто может менять какие флаги)

### 4.1 Минимальный Web UI (если нужен)

Легковесный HTTP-хендлер, монтируемый на debug-порт самого сервиса:

```go
// В platform debug сервере
server.WithDebugHandler("/flags", flagsui.Handler(flags))
```

Показывает:
- Список всех флагов с текущими значениями и дефолтами
- Toggle для bool-флагов
- Input для числовых/строковых
- Кнопка "Reset to default"

Нет отдельного сервиса — UI живёт внутри каждого сервиса на debug-порту.

### 4.2 Полноценный Config Service (если вырастет потребность)

Если когда-то понадобится:
- gRPC API для CRUD флагов
- Web UI (React/Svelte)
- Audit log в PostgreSQL
- RBAC
- Webhooks при изменении

Но это **не нужно сейчас**. etcd + `etcdctl` + встроенный UI на debug-порту покрывает 90% кейсов.

---

## Фаза 5: DX улучшения (приоритет: низкий)

### 5.1 `--watch` режим

```bash
configgen --watch --configs=./configs --output=./internal/config
```

Следит за изменениями TOML файлов и перегенерирует код автоматически. Удобно при разработке.

### 5.2 `--diff` режим

```bash
configgen --diff --configs=./configs --output=./internal/config
```

Показывает что изменится в сгенерированном коде без записи на диск. Для code review.

### 5.3 Поддержка `map[string]string`

```toml
[labels]
env = "prod"
team = "backend"
version = "1.0"
```

Если все значения секции — строки и нет вложенности, генерировать `map[string]string` вместо структуры (по опциональному флагу).

---

## Порядок реализации

| # | Что | Зависит от | Сложность |
|---|-----|-----------|-----------|
| 1 | `flags.toml` парсинг + модель `FlagDef` | — | Низкая |
| 2 | Генерация `configgen_flags.go` с `FlagStore` интерфейсом | 1 | Средняя |
| 3 | `MemoryStore` + `FileStore` | 2 | Низкая |
| 4 | `--validate` CLI | — | Низкая |
| 5 | Тестовые хелперы генерация | 2, 3 | Низкая |
| 6 | `EtcdStore` с Watch | 2 | Средняя |
| 7 | Env var override в loader | — | Низкая |
| 8 | Validation rules | — | Средняя |
| 9 | Вложенные секции | — | Средняя |
| 10 | Debug UI для флагов | 6 | Средняя |
