package schema

import (
	"github.com/vovanwin/configgen/pkg/types"
)

// Field is exported here for simplicity — thin wrapper over pkg/types.Field
type Field = types.Field

// Intersect returns fields common to both a and b (keys + same kinds)
func Intersect(a, b map[string]*Field) map[string]*Field {
	out := make(map[string]*Field)
	for k, fa := range a {
		fb, ok := b[k]
		if !ok {
			continue
		}
		if fa.Kind != fb.Kind {
			// different kinds — ignore
			continue
		}
		if fa.Kind == types.KindObject {
			children := Intersect(fa.Children, fb.Children)
			if len(children) == 0 {
				continue
			}
			out[k] = &Field{Name: fa.Name, YAMLName: fa.YAMLName, Kind: types.KindObject, Children: children}
			continue
		}
		out[k] = fa
	}
	return out
}
