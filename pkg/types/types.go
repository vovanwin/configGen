package types

type Kind int

const (
	KindString Kind = iota
	KindInt
	KindFloat
	KindBool
	KindObject
)

// Field describes one YAML key
type Field struct {
	Name     string
	YAMLName string
	Kind     Kind
	Children map[string]*Field
}

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
	default:
		return "unknown"
	}
}
