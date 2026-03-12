package localize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var pluralRe = regexp.MustCompile(`(?s)^\{\s*([^{}\s]+)\s*,\s*plural\s*,\s*(.+)\}\s*$`)

var placeholderRe = regexp.MustCompile(`\{([^{}\s]+)\}`)

func extractOptions(body string) ([]struct{ key, text string }, error) {
	var options []struct{ key, text string }
	i := 0
	n := len(body)

	for i < n {
		// Пропускаем пробелы
		for i < n && (body[i] == ' ' || body[i] == '\t' || body[i] == '\n') {
			i++
		}
		if i >= n {
			break
		}

		// Находим ключ (one, other, =5 и т.д.)
		startKey := i
		for i < n && body[i] != ' ' && body[i] != '{' {
			i++
		}
		key := strings.TrimSpace(body[startKey:i])
		if key == "" {
			return nil, fmt.Errorf("empty key")
		}

		// Пропускаем до {
		for i < n && body[i] != '{' {
			i++
		}
		if i >= n || body[i] != '{' {
			return nil, fmt.Errorf("expected { after key %q", key)
		}
		i++ // skip {

		// Считаем вложенные { }
		depth := 1
		startText := i
		for i < n && depth > 0 {
			switch body[i] {
			case '{':
				depth++
			case '}':
				depth--
			}
			i++
		}
		if depth != 0 {
			return nil, fmt.Errorf("unclosed { in %q", key)
		}

		text := strings.TrimSpace(body[startText : i-1])
		options = append(options, struct{ key, text string }{key, text})
	}

	return options, nil
}

func parseICU(id, icu string) (*Message, error) {
	m := &Message{ID: id}
	icu = strings.TrimSpace(icu)

	// --- 1. НЕ plural ---
	if !pluralRe.MatchString(icu) {
		icu = strings.Trim(icu, `"`)
		m.Other = convertPlaceholders(icu, "")
		return m, nil
	}

	// --- 2. Это plural ---
	sub := pluralRe.FindStringSubmatch(icu)
	if len(sub) != 3 {
		return nil, fmt.Errorf("invalid plural: %q", icu)
	}

	pluralVar := strings.TrimSpace(sub[1])
	body := sub[2]

	opts, err := extractOptions(body)
	if err != nil {
		return nil, err
	}
	if len(opts) == 0 {
		return nil, fmt.Errorf("no plural forms")
	}

	for _, o := range opts {
		text := convertPlaceholders(o.text, pluralVar)

		switch o.key {
		case "zero":
			m.Zero = text
		case "one":
			m.One = text
		case "two":
			m.Two = text
		case "few":
			m.Few = text
		case "many":
			m.Many = text
		case "other":
			m.Other = text
		default:
			if strings.HasPrefix(o.key, "=") {
				n, _ := strconv.Atoi(strings.TrimPrefix(o.key, "="))
				switch n {
				case 0:
					if m.Zero == "" {
						m.Zero = text
					}
				case 1:
					if m.One == "" {
						m.One = text
					}
				case 2:
					if m.Two == "" {
						m.Two = text
					}
				}
			} else {
				return nil, fmt.Errorf("unknown key: %q", o.key)
			}
		}
	}

	return m, nil
}

func convertPlaceholders(s, pluralVar string) string {
	// 1. # → {.pluralVar}
	if pluralVar != "" {
		s = strings.ReplaceAll(s, "#", "{."+pluralVar+"}")
	}

	// 2. {name} → {.name}
	s = placeholderRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := strings.TrimSpace(match[1 : len(match)-1])
		if strings.HasPrefix(inner, ".") {
			return match
		}
		return "{." + inner + "}"
	})

	return s
}
