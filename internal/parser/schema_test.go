package parser

import (
	"testing"

	"github.com/vovanwin/configgen/internal/model"
)

func TestIntersect(t *testing.T) {
	a := map[string]*model.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: model.KindString},
		"port":   {Name: "Port", TOMLName: "port", Kind: model.KindInt},
		"only_a": {Name: "OnlyA", TOMLName: "only_a", Kind: model.KindString},
	}

	b := map[string]*model.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: model.KindString},
		"port":   {Name: "Port", TOMLName: "port", Kind: model.KindInt},
		"only_b": {Name: "OnlyB", TOMLName: "only_b", Kind: model.KindString},
	}

	result := Intersect(a, b)

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
	a := map[string]*model.Field{
		"value": {Name: "Value", TOMLName: "value", Kind: model.KindString},
	}

	b := map[string]*model.Field{
		"value": {Name: "Value", TOMLName: "value", Kind: model.KindInt},
	}

	result := Intersect(a, b)

	if len(result) != 0 {
		t.Errorf("len(result) = %d, ожидалось 0 (разные типы)", len(result))
	}
}

func TestIntersectNested(t *testing.T) {
	a := map[string]*model.Field{
		"server": {
			Name:     "Server",
			TOMLName: "server",
			Kind:     model.KindObject,
			Children: map[string]*model.Field{
				"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
				"port": {Name: "Port", TOMLName: "port", Kind: model.KindInt},
			},
		},
	}

	b := map[string]*model.Field{
		"server": {
			Name:     "Server",
			TOMLName: "server",
			Kind:     model.KindObject,
			Children: map[string]*model.Field{
				"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
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
	a := map[string]*model.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
	}

	result := Intersect(a)
	if len(result) != 1 {
		t.Errorf("Intersect с одним аргументом должен вернуть его же")
	}
}

func TestUnion(t *testing.T) {
	a := map[string]*model.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: model.KindString},
		"only_a": {Name: "OnlyA", TOMLName: "only_a", Kind: model.KindString},
	}

	b := map[string]*model.Field{
		"host":   {Name: "Host", TOMLName: "host", Kind: model.KindString},
		"only_b": {Name: "OnlyB", TOMLName: "only_b", Kind: model.KindInt},
	}

	result := Union(a, b)

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
	a := map[string]*model.Field{
		"config": {
			Name:     "Config",
			TOMLName: "config",
			Kind:     model.KindObject,
			Children: map[string]*model.Field{
				"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
			},
		},
	}

	b := map[string]*model.Field{
		"config": {
			Name:     "Config",
			TOMLName: "config",
			Kind:     model.KindObject,
			Children: map[string]*model.Field{
				"port": {Name: "Port", TOMLName: "port", Kind: model.KindInt},
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
	a := map[string]*model.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: model.KindString},
	}

	b := map[string]*model.Field{
		"host": {Name: "Host", TOMLName: "host", Kind: model.KindInt},
		"port": {Name: "Port", TOMLName: "port", Kind: model.KindInt},
	}

	result := Merge(a, b)

	if result["host"].Kind != model.KindInt {
		t.Error("Merge должен переопределять значения более поздними")
	}

	if _, ok := result["port"]; !ok {
		t.Error("поле 'port' должно быть в результате")
	}
}
