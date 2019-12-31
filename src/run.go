package main

import (
	"html/template"
	"os"
	"os/signal"
	"syscall"

	"github.com/getsentry/raven-go"
	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/retailcrm/mg-bot-helper/src/models"
	"github.com/retailcrm/mg-transport-core/core"
	cors "github.com/rs/cors/wrapper/gin"
)

func init() {
	parser.AddCommand("run",
		"Run bot",
		"Run bot.",
		&RunCommand{},
	)
}

var sentry *raven.Client

// RunCommand struct
type RunCommand struct{}

// Execute command
func (x *RunCommand) Execute(args []string) error {
	initialize(true)
	go start()

	c := make(chan os.Signal, 1)
	signal.Notify(c)
	for sig := range c {
		switch sig {
		case os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM:
			app.DB.Close()
			return nil
		default:
		}
	}

	return nil
}

func initialize(enableCSRFVerification bool) {
	app.TaggedTypes = getSentryTaggedTypes()
	secret, _ := core.GetEntitySHA1(app.Config.GetTransportInfo())
	app.InitCSRF(
		secret,
		func(c *gin.Context, r core.CSRFErrorReason) {
			app.Logger().Errorf("[%s]: wrong csrf token: %s", c.Request.RemoteAddr, core.GetCSRFErrorMessage(r))
			c.AbortWithStatusJSON(app.BadRequestLocalized("incorrect_csrf_token"))
			return
		},
		core.DefaultCSRFTokenGetter,
	)

	app.ConfigureRouter(func(r *gin.Engine) {
		r.StaticFS("/static", static)

		if r.HTMLRender == nil {
			r.HTMLRender = app.CreateRendererFS(templates, insertTemplates, template.FuncMap{})
		}

		r.Use(cors.New(cors.Options{
			AllowedOrigins:     []string{"https://" + app.Config.GetHTTPConfig().Host},
			AllowedMethods:     []string{"HEAD", "GET", "POST", "PUT", "DELETE"},
			MaxAge:             60 * 5,
			AllowCredentials:   true,
			OptionsPassthrough: false,
			Debug:              false,
		}))
		r.Use(secure.New(secure.Config{
			STSSeconds:            315360000,
			IsDevelopment:         false,
			STSIncludeSubdomains:  true,
			FrameDeny:             true,
			ContentTypeNosniff:    true,
			BrowserXssFilter:      true,
			IENoOpen:              true,
			SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
			ContentSecurityPolicy: "default-src 'self' 'unsafe-eval' 'unsafe-inline' *.facebook.net *.facebook.com; object-src 'none'",
		}))

		if enableCSRFVerification {
			r.Use(app.GenerateCSRFMiddleware())
		}

		initRouting(r, enableCSRFVerification)
	})
	app.BuildHTTPClient(true)
}

func start() {
	startWS()
	app.Run()
}

func startWS() {
	res := models.GetActiveConnection()
	if len(res) > 0 {
		for _, v := range res {
			wm.setWorker(v)
		}
	}
}

func getSentryTaggedTypes() core.SentryTaggedTypes {
	return core.SentryTaggedTypes{
		core.NewTaggedStruct(models.Connection{}, "connection", core.SentryTags{
			"crm":      "URL",
			"clientID": "ClientID",
		}),
	}
}

func insertTemplates(r *core.Renderer) {
	r.Push("home", "layout.html", "home.html")
	r.Push("form", "layout.html", "form.html")
}
