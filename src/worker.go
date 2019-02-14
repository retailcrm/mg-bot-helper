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
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/op/go-logging"
	"github.com/retailcrm/api-client-go/errs"
	"github.com/retailcrm/api-client-go/v5"
	"github.com/retailcrm/mg-bot-api-client-go/v1"
	"golang.org/x/text/language"
)

const (
	CommandPayment  = "/payment"
	CommandDelivery = "/delivery"
	CommandProduct  = "/product"
)

var (
	events         = []string{v1.WsEventMessageNew}
	msgLen         = 2000
	emoji          = []string{"0️⃣ ", "1️⃣ ", "2️⃣ ", "3️⃣ ", "4️⃣ ", "5️⃣ ", "6️⃣ ", "7️⃣ ", "8️⃣ ", "9️⃣ "}
	botCommands    = []string{CommandPayment, CommandDelivery, CommandProduct}
	botCredentials = []string{
		"/api/integration-modules/{code}",
		"/api/integration-modules/{code}/edit",
		"/api/reference/payment-types",
		"/api/reference/delivery-types",
		"/api/store/products",
	}
)

type Worker struct {
	connection *Connection
	mutex      sync.RWMutex
	localizer  *i18n.Localizer

	sentry *raven.Client
	logger *logging.Logger

	mgClient  *v1.MgClient
	crmClient *v5.Client

	close bool
}

func NewWorker(conn *Connection, sentry *raven.Client, logger *logging.Logger) *Worker {
	crmClient := v5.New(conn.APIURL, conn.APIKEY)
	mgClient := v1.New(conn.MGURL, conn.MGToken)
	if config.Debug {
		crmClient.Debug = true
		mgClient.Debug = true
	}

	return &Worker{
		connection: conn,
		sentry:     sentry,
		logger:     logger,
		localizer:  getLang(conn.Lang),
		mgClient:   mgClient,
		crmClient:  crmClient,
		close:      false,
	}
}

func (w *Worker) UpdateWorker(conn *Connection) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	w.localizer = getLang(conn.Lang)
	w.connection = conn
}

func (w *Worker) sendSentry(err error) {
	tags := map[string]string{
		"crm":        w.connection.APIURL,
		"active":     strconv.FormatBool(w.connection.Active),
		"lang":       w.connection.Lang,
		"currency":   w.connection.Currency,
		"updated_at": w.connection.UpdatedAt.String(),
	}

	w.logger.Errorf("ws url: %s\nmgClient: %v\nerr: %v", w.crmClient.URL, w.mgClient, err)
	go w.sentry.CaptureError(err, tags)
}

type WorkersManager struct {
	mutex   sync.RWMutex
	workers map[string]*Worker
}

func NewWorkersManager() *WorkersManager {
	return &WorkersManager{
		workers: map[string]*Worker{},
	}
}

func (wm *WorkersManager) setWorker(conn *Connection) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	if conn.Active {
		worker, ok := wm.workers[conn.ClientID]
		if ok {
			worker.UpdateWorker(conn)
		} else {
			wm.workers[conn.ClientID] = NewWorker(conn, sentry, logger)
			go wm.workers[conn.ClientID].UpWS()
		}
	}
}

func (wm *WorkersManager) stopWorker(conn *Connection) {
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
			if config.Debug {
				w.logger.Debug("stop ws:", w.connection.APIURL)
			}
			return
		}
		ws, _, err := websocket.DefaultDialer.Dial(data, header)
		if err != nil {
			w.sendSentry(err)
			time.Sleep(1000 * time.Millisecond)
			continue ROOT
		}

		if config.Debug {
			w.logger.Info("start ws: ", w.crmClient.URL)
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
				if config.Debug {
					w.logger.Debug("stop ws:", w.connection.APIURL)
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
				msg = w.localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "incorrect_key"})
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
					w.logger.Warningf("MessageSend status: %d\nMessageSend err: %v\nMessageSend data: %v", status, err, d)
					continue
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func checkErrors(err errs.Failure) error {
	if err.RuntimeErr != nil {
		return err.RuntimeErr
	}

	if err.ApiErr != "" {
		return errors.New(err.ApiErr)
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
			return
		}
		for _, v := range res.PaymentTypes {
			if v.Active {
				s = append(s, v.Name)
			}
		}
		if len(s) > 0 {
			resMes = fmt.Sprintf("%s\n\n", w.localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "payment_options"}))
		}
	case CommandDelivery:
		res, _, er := w.crmClient.DeliveryTypes()
		err = checkErrors(er)
		if err != nil {
			return
		}
		for _, v := range res.DeliveryTypes {
			if v.Active {
				s = append(s, v.Name)
			}
		}
		if len(s) > 0 {
			resMes = fmt.Sprintf("%s\n\n", w.localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "delivery_options"}))
		}
	case CommandProduct:
		if params.Filter.Name == "" {
			resMes = w.localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "set_name_or_article"})
			return
		}

		res, _, er := w.crmClient.Products(params)
		err = checkErrors(er)
		if err != nil {
			return
		}

		if len(res.Products) > 0 {
			for _, vp := range res.Products {
				if vp.Active {
					vo := vp.Offers[0]
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
	}

	if len(s) == 0 {
		resMes = w.localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "not_found"})
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

func SetBotCommand(botURL, botToken string) (code int, err error) {
	var client = v1.New(botURL, botToken)

	_, code, err = client.CommandEdit(v1.CommandEditRequest{
		Name:        getTextCommand(CommandPayment),
		Description: getLocalizedMessage("get_payment"),
	})

	_, code, err = client.CommandEdit(v1.CommandEditRequest{
		Name:        getTextCommand(CommandDelivery),
		Description: getLocalizedMessage("get_delivery"),
	})

	_, code, err = client.CommandEdit(v1.CommandEditRequest{
		Name:        getTextCommand(CommandProduct),
		Description: getLocalizedMessage("get_product"),
	})

	return
}

func getTextCommand(command string) string {
	return strings.Replace(command, "/", "", -1)
}

func getLang(lang string) *i18n.Localizer {
	tag, _ := language.MatchStrings(matcher, lang)

	return i18n.NewLocalizer(bundle, tag.String())
}
