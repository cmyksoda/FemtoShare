package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/handlers"
	"github.com/cmyksoda/femtoshare/store/test_sqlite"
)

func TestSettingsPut(t *testing.T) {
	for _, tt := range []struct {
		description string
		payload     string
		settings    femtoshare.Settings
		status      int
	}{
		{
			description: "valid request for 7-day expiration",
			payload: `{
					"defaultExpirationDays": 7
				}`,
			settings: femtoshare.Settings{
				DefaultFileLifetime: femtoshare.NewFileLifetimeInDays(7),
			},
			status: http.StatusOK,
		},
		{
			description: "valid request for never-expiring files",
			payload: `{
					"defaultNeverExpire": true
				}`,
			settings: femtoshare.Settings{
				DefaultFileLifetime: femtoshare.FileLifetimeInfinite,
			},
			status: http.StatusOK,
		},
		{
			description: "rejects invalid expiration days (too low)",
			payload: `{
					"defaultExpirationDays": 0
				}`,
			settings: femtoshare.Settings{},
			status:   http.StatusBadRequest,
		},
		{
			description: "rejects invalid expiration days (not a number)",
			payload: `{
					"defaultExpirationDays": "banana"
				}`,
			settings: femtoshare.Settings{},
			status:   http.StatusBadRequest,
		},
		{
			description: "rejects request with missing expiration days",
			payload: `{
					"someIrrelevantField": "banana"
				}`,
			settings: femtoshare.Settings{},
			status:   http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, nilConversionStateFunc, time.Now)

			req := httptest.NewRequest(
				http.MethodPut,
				"/api/settings",
				strings.NewReader(tt.payload),
			)
			req.Header.Add("Content-Type", "text/json")

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("/api/settings returned wrong status code: got %v want %v",
					got, want)
			}

			if tt.status != http.StatusOK {
				return
			}

			settings, err := dataStore.ReadSettings()
			if err != nil {
				t.Fatalf("failed to retrieve settings from datastore: %v", err)
			}

			if got, want := settings, tt.settings; !reflect.DeepEqual(got, want) {
				t.Fatalf("settings in datastore don't match payload: got %+v, want %+v", got, want)
			}
		})
	}
}
