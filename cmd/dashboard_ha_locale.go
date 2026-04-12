package cmd

import (
	"context"
	"strings"
)

func (c *haWSClient) GetPreferredLanguage(_ context.Context) string {
	var userData map[string]any
	if err := c.call(map[string]any{"type": "frontend/get_user_data"}, &userData); err == nil {
		if language := extractLanguageValue(userData); language != "" {
			return language
		}
	}

	var config map[string]any
	if err := c.call(map[string]any{"type": "config"}, &config); err == nil {
		if language := extractLanguageValue(config); language != "" {
			return language
		}
	}

	return ""
}

func extractLanguageValue(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}

	keys := []string{
		"language",
		"locale",
		"ui_language",
		"frontend_language",
	}
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if language := normalizeLanguageValue(value); language != "" {
				return language
			}
		}
	}

	for _, key := range []string{"data", "user_data", "preferences"} {
		nested, ok := payload[key].(map[string]any)
		if !ok {
			continue
		}
		if language := extractLanguageValue(nested); language != "" {
			return language
		}
	}

	return ""
}

func normalizeLanguageValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return strings.ToLower(strings.ReplaceAll(text, "_", "-"))
}
