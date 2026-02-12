package model

// FlagKind представляет тип feature flag
type FlagKind int

const (
	FlagKindBool FlagKind = iota
	FlagKindInt
	FlagKindFloat
	FlagKindString
	FlagKindEnum
)

func (k FlagKind) String() string {
	switch k {
	case FlagKindBool:
		return "bool"
	case FlagKindInt:
		return "int"
	case FlagKindFloat:
		return "float64"
	case FlagKindString:
		return "string"
	case FlagKindEnum:
		return "enum"
	default:
		return "unknown"
	}
}

// FlagDef описывает один feature flag
type FlagDef struct {
	Name        string   // CamelCase имя (NewCatalogUi)
	TOMLName    string   // snake_case имя (new_catalog_ui)
	Kind        FlagKind // Тип значения
	Default     any      // Типизированный дефолт
	Description string   // Описание флага
	EnumValues  []string // Допустимые значения для enum типа (пусто для не-enum)
}
