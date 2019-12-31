package main

import (
	"fmt"

	"github.com/retailcrm/api-client-go/v5"
	"github.com/retailcrm/mg-bot-helper/src/models"
)

func getIntegrationModule(clientId string) v5.IntegrationModule {
	return v5.IntegrationModule{
		Code:            models.GetConfig().BotInfo.Code,
		IntegrationCode: models.GetConfig().BotInfo.Code,
		Active:          true,
		Name:            models.GetConfig().BotInfo.Name,
		ClientID:        clientId,
		Logo: fmt.Sprintf(
			"https://%s%s",
			models.GetConfig().HTTPServer.Host,
			models.GetConfig().BotInfo.LogoPath,
		),
		BaseURL: fmt.Sprintf(
			"https://%s",
			models.GetConfig().HTTPServer.Host,
		),
		AccountURL: fmt.Sprintf(
			"https://%s/settings/%s",
			models.GetConfig().HTTPServer.Host,
			clientId,
		),
		Actions: map[string]string{"activity": "/actions/activity"},
		Integrations: &v5.Integrations{
			MgBot: &v5.MgBot{},
		},
	}
}
