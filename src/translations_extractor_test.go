package main

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/retailcrm/mg-transport-core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TranslationsExtractorTest will compare correctness between translations. It uses TranslationExtractor.
// TranslationExtractor will load translations data from files or from box, and then it will be used to
// compare every translation file keys to keys from all other translation files. If there is any
// difference - test will fail.
type TranslationsExtractorTest struct {
	suite.Suite
	extractor *core.TranslationsExtractor
	locales   []string
}

// Test_Translations suite runner
func Test_Translations(t *testing.T) {
	suite.Run(t, &TranslationsExtractorTest{
		extractor: core.NewTranslationsExtractor("translate.{}.yml"),
		locales:   []string{"en", "es", "ru"},
	})
}

// getError returns error message from text, or empty string if error is nil
func (t *TranslationsExtractorTest) getError(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

func (t *TranslationsExtractorTest) SetupSuite() {
	configPath := path.Clean("./../config_test.yml")
	info, err := os.Stat(configPath)
	if configPath == "/" || configPath == "." || err != nil || info.IsDir() {
		configPath = path.Clean("./config_test.yml")
	}

	require.False(t.T(), configPath == "/" || configPath == ".", "config_test.yml not found")

	initVariables(configPath)

	require.NotNil(t.T(), app, "app must be initialized to test translations")
	require.False(t.T(), app.TranslationsPath == "" && app.TranslationsBox == nil,
		"translations path or translations box must be initialized in app")

	t.extractor.TranslationsPath = app.TranslationsPath
	t.extractor.TranslationsBox = app.TranslationsBox
}

func (t *TranslationsExtractorTest) Test_Locales() {
	checked := map[string]string{}
	localeData := map[string][]string{}

	for _, locale := range t.locales {
		data, err := t.extractor.LoadLocaleKeys(locale)
		require.NoError(t.T(), err, fmt.Sprintf("error while loading locale `%s`: %s", locale, t.getError(err)))
		localeData[locale] = data
	}

	for _, comparableLocale := range t.locales {
		for _, innerLocale := range t.locales {
			if comparableLocale == innerLocale {
				continue
			}

			if checkedLocale, ok := checked[comparableLocale]; ok && checkedLocale == innerLocale {
				continue
			}

			diff := cmp.Diff(localeData[comparableLocale], localeData[innerLocale])
			assert.Empty(t.T(), diff,
				fmt.Sprintf("non-empty diff between `%s` and `%s`", comparableLocale, innerLocale))
			checked[innerLocale] = comparableLocale
		}
	}
}
