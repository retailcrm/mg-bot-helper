package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/retailcrm/api-client-go/v5"
	"github.com/retailcrm/mg-bot-helper/src/models"
	"github.com/retailcrm/mg-transport-core/core"
)

func initRouting(r *gin.Engine, enableCSRFValidation bool) {
	csrf := func(c *gin.Context) {}
	if enableCSRFValidation {
		csrf = app.VerifyCSRFMiddleware(core.DefaultIgnoredMethods)
	}

	r.GET("/", checkAccountForRequest(), connectHandler)
	r.Any("/settings/:uid", settingsHandler)
	r.POST("/save/", csrf, checkConnectionForRequest(), saveHandler)
	r.POST("/create/", csrf, checkConnectionForRequest(), createHandler)
	r.POST("/bot-settings/", botSettingsHandler)
	r.POST("/actions/activity", activityHandler)
}

func checkAccountForRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		ra := app.RemoveTrailingSlash(c.Query("account"))
		p := models.Connection{
			Connection: core.Connection{
				URL: ra,
			},
		}

		c.Set("account", p)
	}
}

func checkConnectionForRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		var conn models.Connection

		if err := c.ShouldBindJSON(&conn); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": app.GetLocalizedMessage("incorrect_url_key")})
			return
		}
		conn.NormalizeApiUrl()

		c.Set("connection", conn)
	}
}

func connectHandler(c *gin.Context) {
	res := struct {
		Conn      models.Connection
		TokenCSRF string
		Year      int
	}{
		c.MustGet("account").(models.Connection),
		app.GetCSRFToken(c),
		time.Now().Year(),
	}

	c.HTML(http.StatusOK, "home", &res)
}

func botSettingsHandler(c *gin.Context) {
	jm := map[string]string{}

	if err := c.ShouldBindJSON(&jm); err != nil {
		c.Error(err)
		return
	}

	conn := models.GetConnection(jm["client_id"])
	conn.Lang = jm["lang"]
	conn.Currency = jm["currency"]

	err := conn.SaveConnection()
	if err != nil {
		c.Error(err)
		return
	}

	wm.setWorker(conn)

	c.JSON(http.StatusOK, gin.H{"msg": app.GetLocalizedMessage("successful")})
}

func settingsHandler(c *gin.Context) {
	uid := c.Param("uid")
	p := models.GetConnection(uid)
	if p.ID == 0 {
		c.Redirect(http.StatusFound, "/")
		return
	}

	res := struct {
		Conn         *models.Connection
		TokenCSRF    string
		Year         int
		LangCode     []string
		CurrencyCode map[string]string
	}{
		p,
		app.GetCSRFToken(c),
		time.Now().Year(),
		[]string{"en", "ru", "es"},
		currency,
	}

	c.HTML(200, "form", res)
}

func saveHandler(c *gin.Context) {
	conn := c.MustGet("connection").(models.Connection)

	_, code, err := app.GetAPIClient(conn.URL, conn.Key)
	if err != nil {
		if code == http.StatusInternalServerError {
			c.Error(err)
		} else {
			c.JSON(code, gin.H{"error": err.Error()})
		}
		return
	}

	err = conn.SaveConnection()
	if err != nil {
		c.Error(err)
		return
	}

	wm.setWorker(&conn)

	c.JSON(http.StatusOK, gin.H{"msg": app.GetLocalizedMessage("successful")})
}

func createHandler(c *gin.Context) {
	conn := c.MustGet("connection").(models.Connection)

	cl := models.GetConnectionByURL(conn.URL)
	if cl.ID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": app.GetLocalizedMessage("connection_already_created")})
		return
	}

	client, code, err := app.GetAPIClient(conn.URL, conn.Key)
	if err != nil {
		if code == http.StatusInternalServerError {
			c.Error(err)
		} else {
			c.JSON(code, gin.H{"error": err.Error()})
		}
		return
	}

	conn.ClientID = app.GenerateToken()

	data, status, e := client.IntegrationModuleEdit(getIntegrationModule(conn.ClientID))
	if e != nil && e.Error() != "" {
		c.Error(e)
		return
	}

	if status >= http.StatusBadRequest && e != nil {
		c.JSON(app.BadRequestLocalized("error_activity_mg"))
		app.Logger().Error(conn.URL, status, e.ApiError(), e.ApiErrors(), data)
		return
	}

	conn.GateURL = data.Info.MgBotInfo.EndpointUrl
	conn.GateToken = data.Info.MgBotInfo.Token
	conn.Active = true
	conn.Lang = "ru"
	conn.Currency = currency["Российский рубль"]

	bj, _ := json.Marshal(botCommands)
	conn.Commands.RawMessage = bj

	code, err = SetBotCommand(conn.GateURL, conn.GateToken)
	if err != nil {
		c.JSON(code, gin.H{"error": app.GetLocalizedMessage("error_activity_mg")})
		app.Logger().Error(conn.URL, code, err)
		return
	}

	err = conn.CreateConnection()
	if err != nil {
		c.Error(err)
		return
	}

	wm.setWorker(&conn)

	c.JSON(
		http.StatusCreated,
		gin.H{
			"url":     "/settings/" + conn.ClientID,
			"message": app.GetLocalizedMessage("successful"),
		},
	)
}

func activityHandler(c *gin.Context) {
	var (
		activity  v5.Activity
		systemUrl = c.PostForm("systemUrl")
		clientId  = c.PostForm("clientId")
	)

	conn := models.GetConnection(clientId)
	if conn.ID == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"error":   "Wrong data",
			},
		)
		return
	}

	err := json.Unmarshal([]byte(c.PostForm("activity")), &activity)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{
				"success": false,
				"error":   "Wrong data",
			},
		)
		return
	}

	conn.Active = activity.Active && !activity.Freeze

	if systemUrl != "" {
		conn.URL = systemUrl
	}
	conn.NormalizeApiUrl()

	if err := conn.SetConnectionActivity(); err != nil {
		c.Error(err)
		return
	}

	if !conn.Active {
		wm.stopWorker(conn)
	} else {
		wm.setWorker(conn)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
