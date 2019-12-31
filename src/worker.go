package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
	"github.com/retailcrm/api-client-go/errs"
	v5 "github.com/retailcrm/api-client-go/v5"
	v1 "github.com/retailcrm/mg-bot-api-client-go/v1"
	"github.com/retailcrm/mg-bot-helper/src/models"
	"github.com/retailcrm/mg-transport-core/core"
	"golang.org/x/text/language"
)

const (
	CommandPayment  = "/payment"
	CommandDelivery = "/delivery"
	CommandProduct  = "/product"
)

var (
	events      = []string{v1.WsEventMessageNew}
	msgLen      = 2000
	emoji       = []string{"0️⃣ ", "1️⃣ ", "2️⃣ ", "3️⃣ ", "4️⃣ ", "5️⃣ ", "6️⃣ ", "7️⃣ ", "8️⃣ ", "9️⃣ "}
	botCommands = []string{CommandPayment, CommandDelivery, CommandProduct}
)

type Worker struct {
	connection *models.Connection
	mutex      sync.RWMutex
	localizer  *core.Localizer

	sentry *raven.Client
	log    chan LogMessage

	mgClient  *v1.MgClient
	crmClient *v5.Client

	close bool
}

func NewWorker(conn *models.Connection, sentry *raven.Client, logChannel chan LogMessage) *Worker {
	crmClient := v5.New(conn.URL, conn.Key)
	mgClient := v1.New(conn.GateURL, conn.GateToken)
	if app.Config.IsDebug() {
		crmClient.Debug = true
		mgClient.Debug = true
	}

	return &Worker{
		connection: conn,
		sentry:     sentry,
		log:        logChannel,
		localizer:  getLocalizer(conn.Lang),
		mgClient:   mgClient,
		crmClient:  crmClient,
		close:      false,
	}
}

func (w *Worker) UpdateWorker(conn *models.Connection) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	w.localizer = getLocalizer(conn.Lang)
	w.connection = conn
}

func (w *Worker) writeLog(text string, severity logging.Level) {
	w.log <- LogMessage{
		Text:     text,
		Severity: severity,
	}
}

func (w *Worker) sendSentry(err error) {
	tags := map[string]string{
		"crm":        w.connection.URL,
		"active":     strconv.FormatBool(w.connection.Active),
		"lang":       w.connection.Lang,
		"currency":   w.connection.Currency,
		"updated_at": w.connection.UpdatedAt.String(),
	}

	w.writeLog(fmt.Sprintf("ws url: %s\nmgClient: %v\nerr: %v", w.crmClient.URL, w.mgClient, err), logging.ERROR)
	go w.sentry.CaptureError(err, tags)
}

// LogMessage represents log message
type LogMessage struct {
	Text     string
	Severity logging.Level
}

type WorkersManager struct {
	mutex   sync.RWMutex
	log     chan LogMessage
	workers map[string]*Worker
}

func NewWorkersManager() *WorkersManager {
	wm := &WorkersManager{
		workers: map[string]*Worker{},
		log:     make(chan LogMessage),
	}

	go wm.logCollector()

	return wm
}

func (wm *WorkersManager) logCollector() {
	for msg := range wm.log {
		switch msg.Severity {
		case logging.CRITICAL:
			app.Logger().Critical(msg.Text)
		case logging.ERROR:
			app.Logger().Error(msg.Text)
		case logging.WARNING:
			app.Logger().Warning(msg.Text)
		case logging.NOTICE:
			app.Logger().Notice(msg.Text)
		case logging.INFO:
			app.Logger().Info(msg.Text)
		case logging.DEBUG:
			app.Logger().Debug(msg.Text)
		}
	}
}

func (wm *WorkersManager) setWorker(conn *models.Connection) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	if conn.Active {
		worker, ok := wm.workers[conn.ClientID]
		if ok {
			worker.UpdateWorker(conn)
		} else {
			wm.workers[conn.ClientID] = NewWorker(conn, sentry, wm.log)
			go wm.workers[conn.ClientID].UpWS()
		}
	}
}

func (wm *WorkersManager) stopWorker(conn *models.Connection) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	worker, ok := wm.workers[conn.ClientID]
	if ok {
		worker.close = true
		delete(wm.workers, conn.ClientID)
	}
}

func (w *Worker) UpWS() {
	data, header, err := w.mgClient.WsMeta(events)
	if err != nil {
		w.sendSentry(err)
		return
	}

ROOT:
	for {
		if w.close {
			if app.Config.IsDebug() {
				w.writeLog(fmt.Sprint("stop ws:", w.connection.URL), logging.DEBUG)
			}
			return
		}
		ws, _, err := websocket.DefaultDialer.Dial(data, header)
		if err != nil {
			w.sendSentry(err)
			time.Sleep(1000 * time.Millisecond)
			continue ROOT
		}

		if app.Config.IsDebug() {
			w.writeLog(fmt.Sprint("start ws: ", w.crmClient.URL), logging.INFO)
		}

		for {
			var wsEvent v1.WsEvent
			err = ws.ReadJSON(&wsEvent)
			if err != nil {
				w.sendSentry(err)
				if websocket.IsUnexpectedCloseError(err) {
					continue ROOT
				}
				continue
			}

			if w.close {
				if app.Config.IsDebug() {
					w.writeLog(fmt.Sprint("stop ws:", w.connection.URL), logging.DEBUG)
				}
				return
			}

			var eventData v1.WsEventMessageNewData
			err = json.Unmarshal(wsEvent.Data, &eventData)
			if err != nil {
				w.sendSentry(err)
				continue
			}

			if eventData.Message.Type != "command" {
				continue
			}

			msg, msgProd, err := w.execCommand(eventData.Message.Content)
			if err != nil {
				w.sendSentry(err)
				msg = w.localizer.GetLocalizedMessage("incorrect_key")
			}

			msgSend := v1.MessageSendRequest{
				Scope:  v1.MessageScopePrivate,
				ChatID: eventData.Message.ChatID,
			}

			if msg != "" {
				msgSend.Type = v1.MsgTypeText
				msgSend.Content = msg
			} else if msgProd.ID != 0 {
				msgSend.Type = v1.MsgTypeProduct
				msgSend.Product = &msgProd
			}

			if msgSend.Type != "" {
				d, status, err := w.mgClient.MessageSend(msgSend)
				if err != nil {
					w.writeLog(fmt.Sprintf("MessageSend status: %d\nMessageSend err: %v\nMessageSend data: %v", status, err, d), logging.WARNING)
					continue
				}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func checkErrors(err *errs.Failure) error {
	if err != nil && err.Error() != "" {
		return errors.New(err.Error())
	}

	if err != nil && (err.ApiError() != "" || len(err.ApiErrors()) > 0) {
		if err.ApiError() != "" {
			return errors.New(err.ApiError())
		} else {
			var errStr []string
			for key, value := range err.ApiErrors() {
				errStr = append(errStr, key+": "+value)
			}

			return errors.New(strings.Join(errStr, ", "))
		}
	}

	return nil
}

func parseCommand(ci string) (co string, params v5.ProductsRequest, err error) {
	s := strings.Split(ci, " ")

	for _, cmd := range botCommands {
		if s[0] == cmd {
			if len(s) > 1 && cmd == CommandProduct {
				params.Filter = v5.ProductsFilter{
					Name: ci[len(CommandProduct)+1:],
				}
			}
			co = s[0]
			break
		}
	}

	return
}

func (w *Worker) execCommand(message string) (resMes string, msgProd v1.MessageProduct, err error) {
	var s []string

	command, params, err := parseCommand(message)
	if err != nil {
		return
	}

	switch command {
	case CommandPayment:
		res, _, er := w.crmClient.PaymentTypes()
		err = checkErrors(er)
		if err != nil {
			w.writeLog(fmt.Sprintf("%s - Cannot retrieve payment types, error: %s", w.crmClient.URL, err.Error()), logging.ERROR)
			return
		}
		for _, v := range res.PaymentTypes {
			if v.Active {
				s = append(s, v.Name)
			}
		}
		if len(s) > 0 {
			resMes = fmt.Sprintf("%s\n\n", w.localizer.GetLocalizedMessage("payment_options"))
		}
	case CommandDelivery:
		res, _, er := w.crmClient.DeliveryTypes()
		err = checkErrors(er)
		if err != nil {
			w.writeLog(fmt.Sprintf("%s - Cannot retrieve delivery types, error: %s", w.crmClient.URL, err.Error()), logging.ERROR)
			return
		}
		for _, v := range res.DeliveryTypes {
			if v.Active {
				s = append(s, v.Name)
			}
		}
		if len(s) > 0 {
			resMes = fmt.Sprintf("%s\n\n", w.localizer.GetLocalizedMessage("delivery_options"))
		}
	case CommandProduct:
		if params.Filter.Name == "" {
			resMes = w.localizer.GetLocalizedMessage("set_name_or_article")
			return
		}

		res, _, er := w.crmClient.Products(params)
		err = checkErrors(er)
		if err != nil {
			w.writeLog(fmt.Sprintf("%s - Cannot retrieve product, error: %s", w.crmClient.URL, err.Error()), logging.ERROR)
			return
		}

		if len(res.Products) > 0 {
			for _, vp := range res.Products {
				if vp.Active {
					vo := searchOffer(vp.Offers, params.Filter.Name)
					msgProd = v1.MessageProduct{
						ID:      uint64(vo.ID),
						Name:    vo.Name,
						Article: vo.Article,
						Url:     vp.URL,
						Img:     vp.ImageURL,
						Cost: &v1.MessageOrderCost{
							Value:    vo.Price,
							Currency: w.connection.Currency,
						},
					}

					if vp.Quantity > 0 {
						msgProd.Quantity = &v1.MessageOrderQuantity{
							Value: vp.Quantity,
							Unit:  vo.Unit.Sym,
						}
					}

					if len(vo.Images) > 0 {
						msgProd.Img = vo.Images[0]
					}

					return
				}
			}

		}
	default:
		return
	}

	if len(s) == 0 {
		resMes = w.localizer.GetLocalizedMessage("not_found")
		return
	}

	if len(s) > 1 {
		for k, v := range s {
			var a string
			for _, iv := range strings.Split(strconv.Itoa(k+1), "") {
				t, _ := strconv.Atoi(iv)
				a += emoji[t]
			}
			s[k] = fmt.Sprintf("%v %v", a, v)
		}
	}

	str := strings.Join(s, "\n")
	resMes += str

	if len(resMes) > msgLen {
		resMes = resMes[:msgLen]
	}

	return
}

func searchOffer(offers []v5.Offer, filter string) (offer v5.Offer) {
	for _, o := range offers {
		if o.Article == filter {
			offer = o
		}
	}

	if offer.ID == 0 {
		for _, o := range offers {
			if o.Name == filter {
				offer = o
			}
		}
	}

	if offer.ID == 0 {
		offer = offers[0]
	}

	return
}

func SetBotCommand(botURL, botToken string) (code int, err error) {
	var client = v1.New(botURL, botToken)

	_, code, err = client.CommandEdit(v1.CommandEditRequest{
		Name:        getTextCommand(CommandPayment),
		Description: app.GetLocalizedMessage("get_payment"),
	})

	_, code, err = client.CommandEdit(v1.CommandEditRequest{
		Name:        getTextCommand(CommandDelivery),
		Description: app.GetLocalizedMessage("get_delivery"),
	})

	_, code, err = client.CommandEdit(v1.CommandEditRequest{
		Name:        getTextCommand(CommandProduct),
		Description: app.GetLocalizedMessage("get_product"),
	})

	return
}

func getTextCommand(command string) string {
	return strings.Replace(command, "/", "", -1)
}

func getLocalizer(locale string) *core.Localizer {
	var localizer *core.Localizer

	if app.TranslationsBox != nil {
		localizer = core.NewLocalizerFS(language.English, core.DefaultLocalizerBundle(),
			core.DefaultLocalizerMatcher(), app.TranslationsBox)
	}

	if app.TranslationsPath != "" {
		localizer = core.NewLocalizer(language.English, core.DefaultLocalizerBundle(),
			core.DefaultLocalizerMatcher(), app.TranslationsPath)
	}

	if localizer == nil {
		panic("cannot initialize localizer")
	}

	localizer.SetLocale(locale)
	return localizer
}
