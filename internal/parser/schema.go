package parser

import (
	"github.com/vovanwin/configgen/internal/model"
)

// Intersect возвращает поля общие для всех переданных map (ключи + одинаковые типы)
func Intersect(maps ...map[string]*model.Field) map[string]*model.Field {
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
func intersectTwo(a, b map[string]*model.Field) map[string]*model.Field {
	out := make(map[string]*model.Field)

	for k, fa := range a {
		fb, ok := b[k]
		if !ok {
			continue
		}
		if fa.Kind != fb.Kind {
			continue
		}

		if fa.Kind == model.KindObject {
			children := intersectTwo(fa.Children, fb.Children)
			if len(children) == 0 {
				continue
			}
			out[k] = &model.Field{
				Name:     fa.Name,
				TOMLName: fa.TOMLName,
				Kind:     model.KindObject,
				Children: children,
			}
			continue
		}

		if fa.Kind == model.KindSlice {
			if fa.ItemKind != fb.ItemKind {
				continue
			}
			out[k] = &model.Field{
				Name:     fa.Name,
				TOMLName: fa.TOMLName,
				Kind:     model.KindSlice,
				ItemKind: fa.ItemKind,
			}
			continue
		}

		out[k] = fa
	}
	return out
}

// Merge объединяет несколько map полей, более поздние переопределяют более ранние
func Merge(maps ...map[string]*model.Field) map[string]*model.Field {
	if len(maps) == 0 {
		return nil
	}

	result := make(map[string]*model.Field)
	for _, m := range maps {
		for k, f := range m {
			if existing, ok := result[k]; ok && existing.Kind == model.KindObject && f.Kind == model.KindObject {
				result[k] = &model.Field{
					Name:     f.Name,
					TOMLName: f.TOMLName,
					Kind:     model.KindObject,
					Children: Merge(existing.Children, f.Children),
				}
			} else {
				result[k] = f
			}
		}
	}
	return result
}

// Union возвращает все поля из всех map
func Union(maps ...map[string]*model.Field) map[string]*model.Field {
	if len(maps) == 0 {
		return nil
	}

	result := make(map[string]*model.Field)
	for _, m := range maps {
		for k, f := range m {
			if existing, ok := result[k]; ok {
				if existing.Kind == model.KindObject && f.Kind == model.KindObject {
					result[k] = &model.Field{
						Name:     f.Name,
						TOMLName: f.TOMLName,
						Kind:     model.KindObject,
						Children: Union(existing.Children, f.Children),
					}
				}
			} else {
				result[k] = f
			}
		}
	}
	return result
}
