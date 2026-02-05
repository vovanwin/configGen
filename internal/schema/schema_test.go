package schema

import (
	"testing"

	"github.com/vovanwin/configgen/pkg/types"
)

func TestIntersect(t *testing.T) {
	// Создаем две карты полей
	a := map[string]*types.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: types.KindString},
		"port":   {Name: "Port", TOMLName: "port", Kind: types.KindInt},
		"only_a": {Name: "OnlyA", TOMLName: "only_a", Kind: types.KindString},
	}

	b := map[string]*types.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: types.KindString},
		"port":   {Name: "Port", TOMLName: "port", Kind: types.KindInt},
		"only_b": {Name: "OnlyB", TOMLName: "only_b", Kind: types.KindString},
	}

	result := Intersect(a, b)

	// Должны быть только общие поля
	if len(result) != 2 {
		t.Errorf("len(result) = %d, ожидалось 2", len(result))
	}

	if _, ok := result["host"]; !ok {
		t.Error("поле 'host' должно быть в результате")
	}

	if _, ok := result["port"]; !ok {
		t.Error("поле 'port' должно быть в результате")
	}

	if _, ok := result["only_a"]; ok {
		t.Error("поле 'only_a' не должно быть в результате")
	}

	if _, ok := result["only_b"]; ok {
		t.Error("поле 'only_b' не должно быть в результате")
	}
}

func TestIntersectDifferentTypes(t *testing.T) {
	a := map[string]*types.Field{
		"value": {Name: "Value", TOMLName: "value", Kind: types.KindString},
	}

	b := map[string]*types.Field{
		"value": {Name: "Value", TOMLName: "value", Kind: types.KindInt},
	}

	result := Intersect(a, b)

	// Поле с разными типами не должно быть в результате
	if len(result) != 0 {
		t.Errorf("len(result) = %d, ожидалось 0 (разные типы)", len(result))
	}
}

func TestIntersectNested(t *testing.T) {
	a := map[string]*types.Field{
		"server": {
			Name:     "Server",
			TOMLName: "server",
			Kind:     types.KindObject,
			Children: map[string]*types.Field{
				"host": {Name: "Host", TOMLName: "host", Kind: types.KindString},
				"port": {Name: "Port", TOMLName: "port", Kind: types.KindInt},
			},
		},
	}

	b := map[string]*types.Field{
		"server": {
			Name:     "Server",
			TOMLName: "server",
			Kind:     types.KindObject,
			Children: map[string]*types.Field{
				"host": {Name: "Host", TOMLName: "host", Kind: types.KindString},
				// port отсутствует
			},
		},
	}

	result := Intersect(a, b)

	server, ok := result["server"]
	if !ok {
		t.Fatal("секция 'server' должна быть в результате")
	}

	if len(server.Children) != 1 {
		t.Errorf("len(server.Children) = %d, ожидалось 1", len(server.Children))
	}

	if _, ok := server.Children["host"]; !ok {
		t.Error("поле 'host' должно быть в server.Children")
	}
}

func TestIntersectEmpty(t *testing.T) {
	result := Intersect()
	if result != nil {
		t.Error("Intersect() без аргументов должен возвращать nil")
	}
}

func TestIntersectSingle(t *testing.T) {
	a := map[string]*types.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: types.KindString},
	}

	result := Intersect(a)
	if len(result) != 1 {
		t.Errorf("Intersect с одним аргументом должен вернуть его же")
	}
}

func TestUnion(t *testing.T) {
	a := map[string]*types.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: types.KindString},
		"only_a": {Name: "OnlyA", TOMLName: "only_a", Kind: types.KindString},
	}

	b := map[string]*types.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: types.KindString},
		"only_b": {Name: "OnlyB", TOMLName: "only_b", Kind: types.KindInt},
	}

	result := Union(a, b)

	// Должны быть все поля
	if len(result) != 3 {
		t.Errorf("len(result) = %d, ожидалось 3", len(result))
	}

	if _, ok := result["host"]; !ok {
		t.Error("поле 'host' должно быть в результате")
	}

	if _, ok := result["only_a"]; !ok {
		t.Error("поле 'only_a' должно быть в результате")
	}

	if _, ok := result["only_b"]; !ok {
		t.Error("поле 'only_b' должно быть в результате")
	}
}

func TestUnionNested(t *testing.T) {
	a := map[string]*types.Field{
		"config": {
			Name:     "Config",
			TOMLName: "config",
			Kind:     types.KindObject,
			Children: map[string]*types.Field{
				"host": {Name: "Host", TOMLName: "host", Kind: types.KindString},
			},
		},
	}

	b := map[string]*types.Field{
		"config": {
			Name:     "Config",
			TOMLName: "config",
			Kind:     types.KindObject,
			Children: map[string]*types.Field{
				"port": {Name: "Port", TOMLName: "port", Kind: types.KindInt},
			},
		},
	}

	result := Union(a, b)

	config, ok := result["config"]
	if !ok {
		t.Fatal("секция 'config' должна быть в результате")
	}

	if len(config.Children) != 2 {
		t.Errorf("len(config.Children) = %d, ожидалось 2", len(config.Children))
	}
}

func TestMerge(t *testing.T) {
	a := map[string]*types.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: types.KindString},
	}

	b := map[string]*types.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: types.KindInt}, // другой тип
		"port": {Name: "Port", TOMLName: "port", Kind: types.KindInt},
	}

	result := Merge(a, b)

	// b переопределяет a
	if result["host"].Kind != types.KindInt {
		t.Error("Merge должен переопределять значения более поздними")
	}

	if _, ok := result["port"]; !ok {
		t.Error("поле 'port' должно быть в результате")
	}
}
