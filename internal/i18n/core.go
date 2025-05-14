package i18n

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
	"github.com/mcp-ecosystem/mcp-gateway/internal/common/cnst"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	translatorOnce sync.Once
	translator     *I18n
	defaultLang    = cnst.LangEN
)

// SetDefaultLanguage sets the default language for error messages
func SetDefaultLanguage(lang string) {
	defaultLang = lang
}

// InitTranslator initializes the global translator
func InitTranslator(translationsPath string) error {
	var initErr error
	translatorOnce.Do(func() {
		translator = NewI18n(language.Chinese)
		initErr = translator.LoadTranslations(translationsPath)
	})
	return initErr
}

// GetTranslator returns the global translator
func GetTranslator() *I18n {
	if translator == nil {
		// Initialize with default path if not already initialized
		_ = InitTranslator("configs/i18n")
	}
	return translator
}

// I18n manages internationalization and translations
type I18n struct {
	bundle      *i18n.Bundle
	defaultLang language.Tag
}

// NewI18n creates a new I18n instance with the specified default language
func NewI18n(defaultLang language.Tag) *I18n {
	bundle := i18n.NewBundle(defaultLang)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	return &I18n{
		bundle:      bundle,
		defaultLang: defaultLang,
	}
}

// LoadTranslations loads translation files from the specified directory
func (i *I18n) LoadTranslations(translationsDir string) error {
	files, err := os.ReadDir(translationsDir)
	if err != nil {
		return fmt.Errorf("failed to read translations directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(translationsDir, file.Name())
		i.bundle.MustLoadMessageFile(filePath)
	}

	return nil
}

// Translate returns a localized string for the given message ID and language
func (i *I18n) Translate(msgID string, lang string, templateData map[string]interface{}) string {
	tag := language.Make(lang)
	localizer := i18n.NewLocalizer(i.bundle, tag.String(), i.defaultLang.String())

	lc := &i18n.LocalizeConfig{
		MessageID: msgID,
	}

	if len(templateData) > 0 {
		lc.TemplateData = templateData
	}

	msg, err := localizer.Localize(lc)
	if err != nil {
		return msgID // Return original message ID if translation fails
	}

	return msg
}

// TranslateContext returns a localized string using the Gin context's language preference
func (i *I18n) TranslateContext(c *gin.Context, msgID string, templateData map[string]interface{}) string {
	defaultLanguage := "zh"

	lang, exists := c.Get(cnst.XLang)
	if !exists || lang == "" {
		lang = defaultLanguage
	}

	langStr, ok := lang.(string)
	if !ok {
		langStr = defaultLanguage
	}

	return i.Translate(msgID, langStr, templateData)
}

// getLanguageFromRequest extracts language preference from HTTP headers
func getLanguageFromRequest(r *http.Request) string {
	// Try X-Lang header first
	lang := r.Header.Get(cnst.XLang)
	if lang != "" {
		return normalizeLang(lang)
	}

	// Then try Accept-Language
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		langs := strings.Split(acceptLang, ",")
		if len(langs) > 0 {
			firstLang := strings.TrimSpace(strings.Split(langs[0], ";")[0])
			return normalizeLang(firstLang)
		}
	}

	return defaultLang
}

// normalizeLang standardizes language codes
func normalizeLang(lang string) string {
	langCode := strings.Split(lang, "-")[0]
	langCode = strings.ToLower(langCode)

	supportedLangs := []string{"en", "zh"}
	for _, supported := range supportedLangs {
		if langCode == supported {
			return langCode
		}
	}

	return defaultLang
}

// TranslateMessage translates a message ID using the context's language preference
func TranslateMessage(c *gin.Context, msgID string, data map[string]interface{}) string {
	lang, exists := c.Get(cnst.XLang)
	if !exists || lang == "" {
		lang = defaultLang
	}

	langStr, ok := lang.(string)
	if !ok {
		langStr = defaultLang
	}

	t := GetTranslator()
	if t != nil {
		return t.Translate(msgID, langStr, data)
	}
	return msgID
}

// TranslateMessageGin is an alias for TranslateMessage with the same parameter order
func TranslateMessageGin(c *gin.Context, msgID string, data map[string]interface{}) string {
	return TranslateMessage(c, msgID, data)
}

// DebugLoadedMessages prints out debugging information about loaded messages
func (i *I18n) DebugLoadedMessages() {
	// Debug function kept but implementation removed
}
