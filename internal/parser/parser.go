package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/vovanwin/configgen/internal/model"
)

// commentMap хранит комментарии для ключей (section.key -> comment)
type commentMap map[string]string

// ParseFile читает TOML файл и возвращает map[string]*model.Field с деревом полей
func ParseFile(path string) (map[string]*model.Field, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение файла %s: %w", path, err)
	}

	// Извлекаем комментарии из файла
	comments, err := extractComments(path)
	if err != nil {
		return nil, fmt.Errorf("извлечение комментариев %s: %w", path, err)
	}

	var root map[string]any
	if _, err := toml.Decode(string(b), &root); err != nil {
		return nil, fmt.Errorf("декодирование toml %s: %w", path, err)
	}

	fields, err := buildFieldsWithComments(root, comments, "")
	if err != nil {
		return nil, err
	}
	AssignStructNames(fields, "")
	return fields, nil
}

// extractComments парсит TOML файл и извлекает комментарии перед каждым ключом
func extractComments(path string) (commentMap, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	comments := make(commentMap)
	scanner := bufio.NewScanner(file)

	var currentSection string
	var pendingComments []string

	// Регулярки для парсинга
	sectionRe := regexp.MustCompile(`^\s*\[([^\]]+)\]\s*$`)
	keyRe := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=`)
	commentRe := regexp.MustCompile(`^\s*#\s*(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Проверяем секцию [section]
		if match := sectionRe.FindStringSubmatch(line); match != nil {
			currentSection = match[1]
			// Сохраняем комментарий для секции если есть
			if len(pendingComments) > 0 {
				comments[currentSection] = strings.Join(pendingComments, "\n")
				pendingComments = nil
			}
			continue
		}

		// Проверяем комментарий
		if match := commentRe.FindStringSubmatch(line); match != nil {
			comment := strings.TrimSpace(match[1])
			if comment != "" {
				pendingComments = append(pendingComments, comment)
			}
			continue
		}

		// Проверяем ключ
		if match := keyRe.FindStringSubmatch(line); match != nil {
			key := match[1]
			fullKey := key
			if currentSection != "" {
				fullKey = currentSection + "." + key
			}
			if len(pendingComments) > 0 {
				comments[fullKey] = strings.Join(pendingComments, "\n")
				pendingComments = nil
			}
			continue
		}

		// Пустая строка сбрасывает накопленные комментарии
		if strings.TrimSpace(line) == "" {
			pendingComments = nil
		}
	}

	return comments, scanner.Err()
}

// buildFieldsWithComments строит дерево полей из распарсенного TOML с комментариями
func buildFieldsWithComments(node map[string]any, comments commentMap, prefix string) (map[string]*model.Field, error) {
	res := make(map[string]*model.Field)

	for k, v := range node {
		// Пропускаем секцию [flags] — она обрабатывается отдельно в loader
		if prefix == "" && k == "flags" {
			continue
		}

		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		f, err := detectFieldWithComment(k, v, comments, fullKey)
		if err != nil {
			return nil, err
		}
		res[k] = f
	}
	return res, nil
}

// detectFieldWithComment определяет тип поля и создает Field структуру с комментарием
func detectFieldWithComment(key string, val any, comments commentMap, fullKey string) (*model.Field, error) {
	comment := comments[fullKey]

	switch v := val.(type) {
	case string:
		// Проверяем похоже ли на duration
		if _, err := time.ParseDuration(v); err == nil && containsDurationSuffix(v) {
			return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindDuration, Comment: comment}, nil
		}
		return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindString, Comment: comment}, nil

	case int, int64:
		return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindInt, Comment: comment}, nil

	case float32, float64:
		return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindFloat, Comment: comment}, nil

	case bool:
		return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindBool, Comment: comment}, nil

	case []any:
		if len(v) == 0 {
			return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindSlice, ItemKind: model.KindString, Comment: comment}, nil
		}
		itemKind := detectSimpleKind(v[0])
		return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindSlice, ItemKind: itemKind, Comment: comment}, nil

	case map[string]any:
		children := make(map[string]*model.Field)
		for kk, vv := range v {
			childKey := fullKey + "." + kk
			cf, err := detectFieldWithComment(kk, vv, comments, childKey)
			if err != nil {
				return nil, err
			}
			children[kk] = cf
		}
		return &model.Field{Name: ToGoName(key), TOMLName: key, Kind: model.KindObject, Children: children, Comment: comment}, nil

	default:
		return nil, fmt.Errorf("неподдерживаемый тип для ключа %s: %T", key, val)
	}
}

// detectSimpleKind определяет Kind для простых типов
func detectSimpleKind(val any) model.Kind {
	switch val.(type) {
	case string:
		return model.KindString
	case int, int64:
		return model.KindInt
	case float32, float64:
		return model.KindFloat
	case bool:
		return model.KindBool
	default:
		return model.KindString
	}
}

// containsDurationSuffix проверяет наличие суффикса времени
func containsDurationSuffix(s string) bool {
	suffixes := []string{"ns", "us", "µs", "ms", "s", "m", "h"}
	for _, suffix := range suffixes {
		if len(s) > len(suffix) && s[len(s)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

// AssignStructNames рекурсивно проставляет StructName для всех KindObject полей
func AssignStructNames(fields map[string]*model.Field, prefix string) {
	for _, f := range fields {
		if f.Kind == model.KindObject {
			f.StructName = prefix + ToGoName(f.TOMLName)
			AssignStructNames(f.Children, f.StructName)
		}
	}
}

// ToGoName конвертирует snake_case в CamelCase
func ToGoName(s string) string {
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
