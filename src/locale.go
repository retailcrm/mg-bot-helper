package main

import (
	"html/template"
	"io/ioutil"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

var (
	localizer *i18n.Localizer
	bundle    = &i18n.Bundle{DefaultLanguage: language.English}
	matcher   = language.NewMatcher([]language.Tag{
		language.English,
		language.Russian,
		language.Spanish,
	})
)

func loadTranslateFile() {
	bundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)
	files, err := ioutil.ReadDir("translate")
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			bundle.MustLoadMessageFile("translate/" + f.Name())
		}
	}
}

func setLocale(al string) {
	tag, _ := language.MatchStrings(matcher, al)
	localizer = i18n.NewLocalizer(bundle, tag.String())
}

func getLocalizedMessage(messageID string) string {
	return localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: messageID})
}

func getLocale() map[string]interface{} {
	return map[string]interface{}{
		"Version":       config.Version,
		"ButtonSave":    getLocalizedMessage("button_save"),
		"ApiKey":        getLocalizedMessage("api_key"),
		"TabSettings":   getLocalizedMessage("tab_settings"),
		"TabBots":       getLocalizedMessage("tab_bots"),
		"TableUrl":      getLocalizedMessage("table_url"),
		"TableActivity": getLocalizedMessage("table_activity"),
		"Title":         getLocalizedMessage("title"),
		"Language":      getLocalizedMessage("language"),
		"CRMLink":       template.HTML(getLocalizedMessage("crm_link")),
		"DocLink":       template.HTML(getLocalizedMessage("doc_link")),
	}
}
