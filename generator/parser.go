package generator

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Field представляет поле конфига
type Field struct {
	Name     string            // Go имя (CamelCase)
	Key      string            // Полный ключ (server.host)
	TOMLName string            // Имя в TOML
	Type     string            // Go тип
	Comment  string            // Комментарий
	Children map[string]*Field // Вложенные поля
}

// ParseFile парсит TOML файл
func ParseFile(path string) (map[string]*Field, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}

	// Извлекаем комментарии
	comments, err := extractComments(path)
	if err != nil {
		return nil, err
	}

	var root map[string]any
	if _, err := toml.Decode(string(data), &root); err != nil {
		return nil, fmt.Errorf("парсинг %s: %w", path, err)
	}

	return buildFields(root, comments, ""), nil
}

func buildFields(data map[string]any, comments map[string]string, prefix string) map[string]*Field {
	fields := make(map[string]*Field)

	for key, val := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		field := &Field{
			Name:     toCamelCase(key),
			Key:      fullKey,
			TOMLName: key,
			Comment:  comments[fullKey],
		}

		switch v := val.(type) {
		case map[string]any:
			field.Type = toCamelCase(key)
			field.Children = buildFields(v, comments, fullKey)
		case string:
			if isDuration(v) {
				field.Type = "time.Duration"
			} else {
				field.Type = "string"
			}
		case int, int64:
			field.Type = "int"
		case float64:
			field.Type = "float64"
		case bool:
			field.Type = "bool"
		case []any:
			field.Type = detectSliceType(v)
		default:
			field.Type = "any"
		}

		fields[key] = field
	}

	return fields
}

func extractComments(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	comments := make(map[string]string)
	scanner := bufio.NewScanner(file)

	var section string
	var pending []string

	sectionRe := regexp.MustCompile(`^\s*\[([^\]]+)\]`)
	keyRe := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=`)
	commentRe := regexp.MustCompile(`^\s*#\s*(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		if m := sectionRe.FindStringSubmatch(line); m != nil {
			if len(pending) > 0 {
				comments[m[1]] = strings.Join(pending, "\n")
				pending = nil
			}
			section = m[1]
			continue
		}

		if m := commentRe.FindStringSubmatch(line); m != nil {
			if c := strings.TrimSpace(m[1]); c != "" {
				pending = append(pending, c)
			}
			continue
		}

		if m := keyRe.FindStringSubmatch(line); m != nil {
			key := m[1]
			if section != "" {
				key = section + "." + key
			}
			if len(pending) > 0 {
				comments[key] = strings.Join(pending, "\n")
				pending = nil
			}
			continue
		}

		if strings.TrimSpace(line) == "" {
			pending = nil
		}
	}

	return comments, scanner.Err()
}

func toCamelCase(s string) string {
	var out []rune
	upper := true
	for _, r := range s {
		if r == '_' || r == '-' {
			upper = true
			continue
		}
		if upper && r >= 'a' && r <= 'z' {
			r -= 32
		}
		upper = false
		out = append(out, r)
	}
	return string(out)
}

func isDuration(s string) bool {
	if _, err := time.ParseDuration(s); err == nil {
		for _, suffix := range []string{"ns", "us", "µs", "ms", "s", "m", "h"} {
			if strings.HasSuffix(s, suffix) && len(s) > len(suffix) {
				return true
			}
		}
	}
	return false
}

func detectSliceType(arr []any) string {
	if len(arr) == 0 {
		return "[]any"
	}
	switch arr[0].(type) {
	case string:
		return "[]string"
	case int, int64:
		return "[]int"
	case float64:
		return "[]float64"
	case bool:
		return "[]bool"
	default:
		return "[]any"
	}
}
