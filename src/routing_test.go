package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

var (
	router   *gin.Engine
	crmUrl   = "https://test.retailcrm.ru"
	clientID = "09385039f039irf039fkj309fj30jf3"
)

func init() {
	os.Chdir("../")
	config = LoadConfig("config_test.yml")
	orm = NewDb(config)
	logger = newLogger()
	router = setup()
}

func TestMain(m *testing.M) {
	c := Connection{
		ID:       1,
		ClientID: clientID,
		APIKEY:   "ii32if32iuf23iufn2uifnr23inf",
		APIURL:   crmUrl,
		MGURL:    "https://test.retailcrm.pro",
		MGToken:  "988730985u23r390rf8j3984jf32904fj",
		Active:   true,
	}

	orm.DB.Delete(Connection{}, "id > ?", 0)

	c.createConnection()
	retCode := m.Run()
	orm.DB.Delete(Connection{}, "id > ?", 0)
	os.Exit(retCode)
}

func TestRouting_connectHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		fmt.Sprintf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK))
}

func TestRouting_settingsHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/settings/"+clientID, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		fmt.Sprintf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK))
}

func TestRouting_saveHandler(t *testing.T) {
	defer gock.Off()

	gock.New(crmUrl).
		Get("/api/credentials").
		Reply(200).
		BodyString(`{"success": true, "credentials": ["/api/integration-modules/{code}", "/api/integration-modules/{code}/edit", "/api/reference/payment-types", "/api/reference/delivery-types", "/api/store/products"]}`)

	req, err := http.NewRequest("POST", "/save/",
		strings.NewReader(fmt.Sprintf(
			`{"clientId": "%s",
			"api_url": "%s",
			"api_key": "test"}`,
			clientID,
			crmUrl,
		)),
	)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		fmt.Sprintf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK))
}

func TestRouting_activityHandler(t *testing.T) {
	startWS()

	if _, ok := wm.workers[clientID]; !ok {
		t.Fatal("worker don`t start")
	}

	data := []url.Values{
		{
			"clientId":  {clientID},
			"activity":  {`{"active": false, "freeze": false}`},
			"systemUrl": {crmUrl},
		},
		{
			"clientId":  {clientID},
			"activity":  {`{"active": true, "freeze": false}`},
			"systemUrl": {crmUrl},
		},
		{
			"clientId":  {clientID},
			"activity":  {`{"active": true, "freeze": false}`},
			"systemUrl": {"http://change.retailcrm.ru"},
		},
	}

	for _, v := range data {

		req, err := http.NewRequest("POST", "/actions/activity", strings.NewReader(v.Encode()))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code,
			fmt.Sprintf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK))

		activity := make(map[string]bool)
		err = json.Unmarshal([]byte(v.Get("activity")), &activity)
		if err != nil {
			t.Fatal(err)
		}

		w, ok := wm.workers[clientID]

		if ok != (activity["active"] && !activity["freeze"]) {
			t.Error("worker don`t stop")
		}

		if ok && w.connection.APIURL != v.Get("systemUrl") {
			t.Error("fail update systemUrl")
		}
	}
}

func TestTranslate(t *testing.T) {
	files, err := ioutil.ReadDir("translate")
	if err != nil {
		t.Fatal(err)
	}

	m := make(map[int]map[string]string)
	i := 0

	for _, f := range files {
		mt := make(map[string]string)
		if !f.IsDir() {
			yamlFile, err := ioutil.ReadFile("translate/" + f.Name())
			if err != nil {
				t.Fatal(err)
			}

			err = yaml.Unmarshal(yamlFile, &mt)
			if err != nil {
				t.Fatal(err)
			}

			m[i] = mt
			i++
		}
	}

	for k, v := range m {
		for kv := range v {
			if len(m) > k+1 {
				if _, ok := m[k+1][kv]; !ok {
					t.Error(kv)
				}
			}
		}
	}
}
