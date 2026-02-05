package model

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
