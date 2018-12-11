package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/retailcrm/api-client-go/v5"
)

// GenerateToken function
func GenerateToken() string {
	c := atomic.AddUint32(&tokenCounter, 1)

	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d%d", time.Now().UnixNano(), c))))
}

func getAPIClient(url, key string) (*v5.Client, error, int) {
	client := v5.New(url, key)

	cr, _, e := client.APICredentials()
	if e.RuntimeErr != nil {
		return nil, e.RuntimeErr, http.StatusInternalServerError
	}

	if !cr.Success {
		return nil, errors.New(getLocalizedMessage("incorrect_url_key")), http.StatusBadRequest
	}

	if res := checkCredentials(cr.Credentials); len(res) != 0 {
		return nil,
			errors.New(localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "missing_credentials",
				TemplateData: map[string]interface{}{
					"Credentials": strings.Join(res, ", "),
				},
			})),
			http.StatusBadRequest
	}

	return client, nil, 0
}

func checkCredentials(credential []string) []string {
	rc := make([]string, len(botCredentials))
	copy(rc, botCredentials)

	for _, vc := range credential {
		for kn, vn := range rc {
			if vn == vc {
				if len(rc) == 1 {
					rc = rc[:0]
					break
				}
				rc = append(rc[:kn], rc[kn+1:]...)
			}
		}
	}

	return rc
}
