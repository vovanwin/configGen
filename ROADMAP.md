# ConfigGen ROADMAP

## Текущее состояние

ConfigGen — генератор типобезопасных Go-конфигов из TOML. Генерирует структуры + загрузчик на koanf.
Поддерживает **статические** конфиги и **feature flags** с динамическими значениями.

---

## Реализовано

### Фаза 1: Feature Flags (шаги 1–5)

- [x] **Модель FlagDef** — `FlagKind` enum (bool/int/float/string) + `FlagDef` struct
- [x] **Парсер flags.toml** — `ParseFlagsFile()` с валидацией типов, coerce int64→int, сортировкой
- [x] **Генерация configgen_flags.go** — `FlagStore` интерфейс, `Flags` struct с типизированными геттерами, `DefaultFlagValues()`, `Store()` accessor
- [x] **Генерация configgen_flagstore.go** — `MemoryStore` (thread-safe, с `Set()`) + `FileStore` (TOML override файл)
- [x] **Генерация configgen_flags_test_helpers.go** — `TestFlags()` и `TestFlagsWith()` с build tag `//go:build !production`
- [x] **CLI --with-flags** — автоматическая генерация при наличии `flags.toml`
- [x] **CLI --validate** — проверка всех TOML без генерации кода
- [x] **Init шаблон** — `flags.toml` создаётся при `--init`

### Фаза 2: Тестирование (частично)

- [x] **Тестовые хелперы** — `TestFlags()`, `TestFlagsWith(overrides)` генерируются автоматически
- [x] **--validate** — валидация конфигов и флагов в CI
- [x] **Unit тесты** — парсер (6 тестов), генератор (3 теста + flagDefaultLiteral)

---

## Что можно сделать дальше

### Бэкенды FlagStore

- [ ] **EtcdStore** — production бэкенд с Watch на изменения
  - Prefix `/myservice/flags/key` → строковое значение
  - Автоматический reconnect
  - `Watch(ctx, onChange)` через etcd watcher
- [ ] **RedisStore** — альтернатива etcd для простых кейсов
- [ ] **CompositeStore** — fallback chain: etcd → file → defaults

### Улучшения флагов

- [ ] **Флаги с enum** — `type = "enum", values = ["v1", "v2"], default = "v1"`
- [ ] **Категории флагов** — группировка в `[flags.ui]`, `[flags.backend]`
- [ ] **TTL/Expiry** — автоматическое отключение флага после даты
- [ ] **Percentage rollout** — `{ type = "percent", default = 0 }` для канареечных выкаток
- [ ] **User/Segment targeting** — включение по user ID или группе

### Улучшения статических конфигов

- [ ] **Env var override** — `SERVER_HOST=override` через koanf env provider
- [ ] **Validation rules** — `# validate: min=1,max=65535` в комментариях TOML
- [ ] **Вложенные секции** — `[server.tls]` → `ServerTls` struct
- [ ] **Map support** — `[labels]` → `map[string]string`
- [ ] **Sensitive fields** — маскирование паролей в логах

### UI и управление

- [ ] **Debug UI расширения** — toggle bool-флагов через POST, input для int/string/float
- [ ] **Audit log** — кто, когда, что изменил (для production)
- [ ] **Bulk operations** — импорт/экспорт флагов как JSON/TOML
- [ ] **Notifications** — webhook при изменении флага

### DX улучшения

- [ ] **--watch** — авто-регенерация при изменении TOML файлов
- [ ] **--diff** — показать что изменится без записи на диск
- [ ] **go:generate** — автоматическая интеграция через `//go:generate`
- [ ] **LSP hints** — подсветка неиспользуемых флагов в IDE
- [ ] **TestConfig()** — генерация конфига с разумными дефолтами для тестов

### Интеграции

- [ ] **Temporal** — флаги в workflow через side effect
- [ ] **OpenTelemetry** — метрики flag evaluation count
- [ ] **Feature flag SDK** — клиентская библиотека без configgen (для внешних сервисов)

---

## Порядок реализации (рекомендуемый)

| # | Что | Сложность | Приоритет |
|---|-----|-----------|-----------|
| 1 | EtcdStore с Watch | Средняя | Высокий |
| 2 | Debug UI toggle/input (POST) | Низкая | Средний |
| 3 | Env var override в loader | Низкая | Средний |
| 4 | Validation rules | Средняя | Средний |
| 5 | TestConfig() генерация | Низкая | Средний |
| 6 | --watch режим | Низкая | Низкий |
| 7 | Enum флаги | Низкая | Низкий |
| 8 | Вложенные секции | Средняя | Низкий |
| 9 | Percentage rollout | Средняя | Низкий |
| 10 | Audit log + webhooks | Высокая | Низкий |
