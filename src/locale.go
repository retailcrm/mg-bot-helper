package main

import (
	"html/template"
)

func old_getLocale() map[string]interface{} {
	return map[string]interface{}{
		"Version":       "version here",
		"ButtonSave":    app.GetLocalizedMessage("button_save"),
		"ApiKey":        app.GetLocalizedMessage("api_key"),
		"TabSettings":   app.GetLocalizedMessage("tab_settings"),
		"TabBots":       app.GetLocalizedMessage("tab_bots"),
		"TableUrl":      app.GetLocalizedMessage("table_url"),
		"TableActivity": app.GetLocalizedMessage("table_activity"),
		"Title":         app.GetLocalizedMessage("title"),
		"Language":      app.GetLocalizedMessage("language"),
		"CRMLink":       template.HTML(app.GetLocalizedMessage("crm_link")),
		"DocLink":       template.HTML(app.GetLocalizedMessage("doc_link")),
	}
}
