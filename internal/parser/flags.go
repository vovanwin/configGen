package parser

import (
	"fmt"
	"os"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/vovanwin/configgen/internal/model"
)

// flagEntry представляет один флаг из flags.toml
type flagEntry struct {
	Type        string `toml:"type"`
	Default     any    `toml:"default"`
	Description string `toml:"description"`
}

// flagsFile корневая структура flags.toml
type flagsFile struct {
	Flags map[string]flagEntry `toml:"flags"`
}

// ParseFlagsFile читает flags.toml и возвращает отсортированный слайс FlagDef
func ParseFlagsFile(path string) ([]*model.FlagDef, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}

	var ff flagsFile
	if _, err := toml.Decode(string(b), &ff); err != nil {
		return nil, fmt.Errorf("декодирование %s: %w", path, err)
	}

	if len(ff.Flags) == 0 {
		return nil, nil
	}

	var defs []*model.FlagDef
	for name, entry := range ff.Flags {
		def, err := flagEntryToDef(name, entry)
		if err != nil {
			return nil, fmt.Errorf("флаг %q: %w", name, err)
		}
		defs = append(defs, def)
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].TOMLName < defs[j].TOMLName
	})

	return defs, nil
}

func flagEntryToDef(name string, entry flagEntry) (*model.FlagDef, error) {
	kind, err := parseFlagKind(entry.Type)
	if err != nil {
		return nil, err
	}

	def, err := coerceDefault(entry.Default, kind)
	if err != nil {
		return nil, fmt.Errorf("default: %w", err)
	}

	return &model.FlagDef{
		Name:        ToGoName(name),
		TOMLName:    name,
		Kind:        kind,
		Default:     def,
		Description: entry.Description,
	}, nil
}

func parseFlagKind(t string) (model.FlagKind, error) {
	switch t {
	case "bool":
		return model.FlagKindBool, nil
	case "int":
		return model.FlagKindInt, nil
	case "float":
		return model.FlagKindFloat, nil
	case "string":
		return model.FlagKindString, nil
	default:
		return 0, fmt.Errorf("неподдерживаемый тип %q (допустимы: bool, int, float, string)", t)
	}
}

func coerceDefault(val any, kind model.FlagKind) (any, error) {
	switch kind {
	case model.FlagKindBool:
		v, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("ожидался bool, получен %T", val)
		}
		return v, nil
	case model.FlagKindInt:
		switch v := val.(type) {
		case int64:
			return int(v), nil
		case int:
			return v, nil
		default:
			return nil, fmt.Errorf("ожидался int, получен %T", val)
		}
	case model.FlagKindFloat:
		switch v := val.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return nil, fmt.Errorf("ожидался float, получен %T", val)
		}
	case model.FlagKindString:
		v, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("ожидался string, получен %T", val)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("неизвестный тип %d", kind)
	}
}
