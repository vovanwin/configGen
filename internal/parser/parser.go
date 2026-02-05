package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/vovanwin/configgen/pkg/types"
)

// commentMap хранит комментарии для ключей (section.key -> comment)
type commentMap map[string]string

// ParseFile читает TOML файл и возвращает map[string]*types.Field с деревом полей
func ParseFile(path string) (map[string]*types.Field, error) {
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

	return buildFieldsWithComments(root, comments, "")
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
func buildFieldsWithComments(node map[string]any, comments commentMap, prefix string) (map[string]*types.Field, error) {
	res := make(map[string]*types.Field)

	for k, v := range node {
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
func detectFieldWithComment(key string, val any, comments commentMap, fullKey string) (*types.Field, error) {
	comment := comments[fullKey]

	switch v := val.(type) {
	case string:
		// Проверяем похоже ли на duration
		if _, err := time.ParseDuration(v); err == nil && containsDurationSuffix(v) {
			return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindDuration, Comment: comment}, nil
		}
		return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindString, Comment: comment}, nil

	case int, int64:
		return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindInt, Comment: comment}, nil

	case float32, float64:
		return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindFloat, Comment: comment}, nil

	case bool:
		return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindBool, Comment: comment}, nil

	case []any:
		if len(v) == 0 {
			return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindSlice, ItemKind: types.KindString, Comment: comment}, nil
		}
		itemKind := detectSimpleKind(v[0])
		return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindSlice, ItemKind: itemKind, Comment: comment}, nil

	case map[string]any:
		children := make(map[string]*types.Field)
		for kk, vv := range v {
			childKey := fullKey + "." + kk
			cf, err := detectFieldWithComment(kk, vv, comments, childKey)
			if err != nil {
				return nil, err
			}
			children[kk] = cf
		}
		return &types.Field{Name: ToGoName(key), TOMLName: key, Kind: types.KindObject, Children: children, Comment: comment}, nil

	default:
		return nil, fmt.Errorf("неподдерживаемый тип для ключа %s: %T", key, val)
	}
}

// detectSimpleKind определяет Kind для простых типов
func detectSimpleKind(val any) types.Kind {
	switch val.(type) {
	case string:
		return types.KindString
	case int, int64:
		return types.KindInt
	case float32, float64:
		return types.KindFloat
	case bool:
		return types.KindBool
	default:
		return types.KindString
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
