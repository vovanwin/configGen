package types

import "time"

// Kind представляет тип поля конфигурации
type Kind int

const (
	KindString Kind = iota
	KindInt
	KindFloat
	KindBool
	KindObject
	KindSlice
	KindDuration
)

func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindFloat:
		return "float64"
	case KindBool:
		return "bool"
	case KindObject:
		return "object"
	case KindSlice:
		return "[]"
	case KindDuration:
		return "time.Duration"
	default:
		return "unknown"
	}
}

// Field представляет одно поле конфигурации
type Field struct {
	Name     string            // Имя в Go (CamelCase)
	TOMLName string            // Оригинальное имя из TOML
	Kind     Kind              // Тип поля
	Children map[string]*Field // Для вложенных объектов
	ItemKind Kind              // Для слайсов: тип элементов
	Comment  string            // Комментарий из TOML файла
}

// Source указывает откуда пришло значение конфига
type Source int

const (
	SourceDefault Source = iota // Из value.toml
	SourceEnv                   // Из config_{env}.toml
	SourceLocal                 // Из config_local.toml
	SourceVault                 // Из Vault (будущее)
	SourceRTC                   // Из RTC (будущее)
)

func (s Source) String() string {
	switch s {
	case SourceDefault:
		return "default"
	case SourceEnv:
		return "env"
	case SourceLocal:
		return "local"
	case SourceVault:
		return "vault"
	case SourceRTC:
		return "rtc"
	default:
		return "unknown"
	}
}

// Environment представляет окружение развертывания
type Environment string

const (
	EnvLocal      Environment = "local"
	EnvDev        Environment = "dev"
	EnvStaging    Environment = "stg"
	EnvProduction Environment = "prod"
)

// IsValid проверяет валидность окружения
func (e Environment) IsValid() bool {
	switch e {
	case EnvLocal, EnvDev, EnvStaging, EnvProduction:
		return true
	}
	return false
}

// LoadOptions настраивает загрузку конфигурации
type LoadOptions struct {
	// ConfigDir - директория с файлами конфигурации
	ConfigDir string

	// Environment - окружение для загрузки (local, dev, stg, prod)
	Environment Environment

	// EnvVarName - имя переменной окружения для определения окружения
	// По умолчанию: APP_ENV
	EnvVarName string

	// EnableLocalOverride позволяет config_local.toml переопределять значения
	EnableLocalOverride bool

	// VaultEnabled включает интеграцию с Vault (будущее)
	VaultEnabled bool

	// VaultConfig содержит настройки подключения к Vault (будущее)
	VaultConfig *VaultConfig

	// RTCEnabled включает обновления конфига в реальном времени (будущее)
	RTCEnabled bool

	// RTCConfig содержит настройки RTC (будущее)
	RTCConfig *RTCConfig

	// OnChange колбэк при изменении конфига (будущее, для RTC)
	OnChange func()
}

// VaultConfig содержит настройки Vault (заготовка)
type VaultConfig struct {
	Address   string
	Token     string
	Path      string
	Namespace string
	Timeout   time.Duration
}

// RTCConfig содержит настройки real-time config (заготовка)
type RTCConfig struct {
	// Endpoint для RTC сервиса
	Endpoint string

	// PollInterval для polling-based RTC
	PollInterval time.Duration

	// UseWebSocket включает обновления через WebSocket
	UseWebSocket bool
}

// DefaultLoadOptions возвращает настройки по умолчанию
func DefaultLoadOptions() *LoadOptions {
	return &LoadOptions{
		ConfigDir:           "./configs",
		Environment:         EnvDev,
		EnvVarName:          "APP_ENV",
		EnableLocalOverride: true,
		VaultEnabled:        false,
		RTCEnabled:          false,
	}
}
