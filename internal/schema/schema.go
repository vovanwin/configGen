package schema

import (
	"github.com/vovanwin/configgen/pkg/types"
)

// Intersect возвращает поля общие для всех переданных map (ключи + одинаковые типы)
func Intersect(maps ...map[string]*types.Field) map[string]*types.Field {
	if len(maps) == 0 {
		return nil
	}
	if len(maps) == 1 {
		return maps[0]
	}

	result := maps[0]
	for i := 1; i < len(maps); i++ {
		result = intersectTwo(result, maps[i])
	}
	return result
}

// intersectTwo находит пересечение двух map
func intersectTwo(a, b map[string]*types.Field) map[string]*types.Field {
	out := make(map[string]*types.Field)

	for k, fa := range a {
		fb, ok := b[k]
		if !ok {
			continue
		}
		if fa.Kind != fb.Kind {
			continue
		}

		if fa.Kind == types.KindObject {
			children := intersectTwo(fa.Children, fb.Children)
			if len(children) == 0 {
				continue
			}
			out[k] = &types.Field{
				Name:     fa.Name,
				TOMLName: fa.TOMLName,
				Kind:     types.KindObject,
				Children: children,
			}
			continue
		}

		if fa.Kind == types.KindSlice {
			if fa.ItemKind != fb.ItemKind {
				continue
			}
			out[k] = &types.Field{
				Name:     fa.Name,
				TOMLName: fa.TOMLName,
				Kind:     types.KindSlice,
				ItemKind: fa.ItemKind,
			}
			continue
		}

		out[k] = fa
	}
	return out
}

// Merge объединяет несколько map полей, более поздние переопределяют более ранние
// Используется для слияния value.toml + config_{env}.toml + config_local.toml
func Merge(maps ...map[string]*types.Field) map[string]*types.Field {
	if len(maps) == 0 {
		return nil
	}

	result := make(map[string]*types.Field)
	for _, m := range maps {
		for k, f := range m {
			if existing, ok := result[k]; ok && existing.Kind == types.KindObject && f.Kind == types.KindObject {
				// Рекурсивно мержим вложенные объекты
				result[k] = &types.Field{
					Name:     f.Name,
					TOMLName: f.TOMLName,
					Kind:     types.KindObject,
					Children: Merge(existing.Children, f.Children),
				}
			} else {
				result[k] = f
			}
		}
	}
	return result
}

// Union возвращает все поля из всех map (для генерации схемы)
func Union(maps ...map[string]*types.Field) map[string]*types.Field {
	if len(maps) == 0 {
		return nil
	}

	result := make(map[string]*types.Field)
	for _, m := range maps {
		for k, f := range m {
			if existing, ok := result[k]; ok {
				// Если типы совпадают, мержим для объектов
				if existing.Kind == types.KindObject && f.Kind == types.KindObject {
					result[k] = &types.Field{
						Name:     f.Name,
						TOMLName: f.TOMLName,
						Kind:     types.KindObject,
						Children: Union(existing.Children, f.Children),
					}
				}
				// Иначе оставляем существующий (первый побеждает при конфликте типов)
			} else {
				result[k] = f
			}
		}
	}
	return result
}
