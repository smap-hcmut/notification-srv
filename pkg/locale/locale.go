package locale

import (
	"context"
	"strings"
)

func ParseLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	switch lang {
	case EN, "english":
		return EN
	case VI, "vietnamese", "việt nam":
		return VI
	case JA, "japanese":
		return JA
	default:
		return DefaultLang
	}
}

func IsValidLang(lang string) bool {
	lang = strings.TrimSpace(strings.ToLower(lang))
	for _, supported := range LangList {
		if lang == supported {
			return true
		}
	}
	return false
}

func GetLang(ctx context.Context) string {
	lang, ok := GetLocaleFromContext(ctx)
	if !ok {
		return DefaultLang
	}
	return lang
}

func SetLocaleToContext(ctx context.Context, lang string) context.Context {
	if !IsValidLang(lang) {
		lang = DefaultLang
	}
	return context.WithValue(ctx, Locale{}, lang)
}

func GetLocaleFromContext(ctx context.Context) (string, bool) {
	lang, ok := ctx.Value(Locale{}).(string)
	if !ok || lang == "" {
		return "", false
	}
	return lang, true
}
