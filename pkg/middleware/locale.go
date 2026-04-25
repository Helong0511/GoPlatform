package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const ContextLangKey = "lang"

// LocaleMiddleware 从请求中解析语言，默认简体中文。
// 优先级：X-Lang > Accept-Language > 默认 zh-CN
func LocaleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := strings.TrimSpace(c.GetHeader("X-Lang"))
		if lang == "" {
			lang = parseAcceptLanguage(c.GetHeader("Accept-Language"))
		}
		if lang == "" {
			lang = "zh-CN"
		}
		c.Set(ContextLangKey, normalizeLang(lang))
		c.Next()
	}
}

func parseAcceptLanguage(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	parts := strings.Split(v, ",")
	if len(parts) == 0 {
		return ""
	}
	first := strings.TrimSpace(parts[0])
	first = strings.Split(first, ";")[0]
	return first
}

func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if strings.HasPrefix(lang, "zh") {
		return "zh-CN"
	}
	if strings.HasPrefix(lang, "en") {
		return "en-US"
	}
	return "zh-CN"
}
