package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func loadDotEnvFiles(paths ...string) {
	protected := currentEnvironment()
	loaded := make(map[string]struct{})

	for _, path := range paths {
		loadDotEnvFile(path, protected, loaded)
	}
}

func currentEnvironment() map[string]struct{} {
	protected := make(map[string]struct{})

	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok && key != "" {
			protected[key] = struct{}{}
		}
	}

	return protected
}

func loadDotEnvFile(path string, protected map[string]struct{}, loaded map[string]struct{}) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := parseDotEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := protected[key]; exists {
			continue
		}

		if _, exists := loaded[key]; exists {
			_ = os.Unsetenv(key)
		}
		_ = os.Setenv(key, value)
		loaded[key] = struct{}{}
	}
}

func parseDotEnvLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}

	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}

	key, rawValue, ok := strings.Cut(trimmed, "=")
	if !ok {
		return "", "", false
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}

	value := strings.TrimSpace(rawValue)
	if value == "" {
		return key, "", true
	}

	value = stripInlineComment(value)

	if strings.HasPrefix(value, "\"") {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			return key, unquoted, true
		}
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return key, value[1 : len(value)-1], true
	}

	return key, value, true
}

func stripInlineComment(value string) string {
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i, r := range value {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && inDoubleQuote:
			escaped = true
		case r == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case r == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case r == '#' && !inSingleQuote && !inDoubleQuote:
			if i == 0 || value[i-1] == ' ' || value[i-1] == '\t' {
				return strings.TrimSpace(value[:i])
			}
		}
	}

	return value
}
