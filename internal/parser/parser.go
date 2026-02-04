package parser

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/vovanwin/configgen/internal/schema"
	"github.com/vovanwin/configgen/pkg/types"
	"gopkg.in/yaml.v3"
)

// ParseFile reads a YAML file and returns a map[string]*schema.Field representing its tree
func ParseFile(path string) (map[string]*schema.Field, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root any
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	m, err := buildFields(root)
	if err != nil {
		return nil, err
	}
	_ = filepath.Base(path)
	return m, nil
}

func buildFields(node any) (map[string]*schema.Field, error) {
	res := make(map[string]*schema.Field)

	m, ok := node.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("top-level must be map[string]any")
	}

	for k, v := range m {
		f, err := detectField(k, v)
		if err != nil {
			return nil, err
		}
		res[k] = f
	}
	return res, nil
}

func detectField(key string, val any) (*schema.Field, error) {
	switch v := val.(type) {
	case string:
		return &schema.Field{Name: toGoName(key), YAMLName: key, Kind: types.KindString}, nil
	case int, int64:
		return &schema.Field{Name: toGoName(key), YAMLName: key, Kind: types.KindInt}, nil
	case float32, float64:
		return &schema.Field{Name: toGoName(key), YAMLName: key, Kind: types.KindFloat}, nil
	case bool:
		return &schema.Field{Name: toGoName(key), YAMLName: key, Kind: types.KindBool}, nil
	case map[string]any:
		children := make(map[string]*schema.Field)
		for kk, vv := range v {
			cf, err := detectField(kk, vv)
			if err != nil {
				return nil, err
			}
			children[kk] = cf
		}
		return &schema.Field{Name: toGoName(key), YAMLName: key, Kind: types.KindObject, Children: children}, nil
	default:
		return nil, fmt.Errorf("unsupported type for key %s: %T", key, val)
	}
}

// simple CamelCase converter for keys like pool_size -> PoolSize
func toGoName(s string) string {
	b := []rune(s)
	out := make([]rune, 0, len(b))
	capNext := true
	for _, r := range b {
		if r == '_' || r == '-' || r == ' ' {
			capNext = true
			continue
		}
		if capNext {
			if 'a' <= r && r <= 'z' {
				r = r - 'a' + 'A'
			}
			capNext = false
		}
		out = append(out, r)
	}
	return string(out)
}
