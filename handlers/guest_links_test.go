package handlers_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cmyksoda/femtoshare/femtoshare"
	"github.com/cmyksoda/femtoshare/handlers"
	"github.com/cmyksoda/femtoshare/store"
	"github.com/cmyksoda/femtoshare/store/test_sqlite"
)

func TestGuestLinksPost(t *testing.T) {
	for _, tt := range []struct {
		description string
		payload     string
		currentTime time.Time
		expected    femtoshare.GuestLink
		status      int
	}{
		{
			description: "minimally populated request",
			payload: `{
					"label": null,
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: femtoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           femtoshare.GuestLinkLabel(""),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
			},
			status: http.StatusOK,
		},
		{
			description: "fully populated request",
			payload: `{
					"label": "For my good pal, Maurice",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"876000h0m0s",
					"maxFileBytes": 1048576,
					"maxFileUploads": 1
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: femtoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           femtoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads:  makeGuestUploadCountLimit(1),
			},
			status: http.StatusOK,
		},
		{
			description: "guest file expires in 1 day",
			payload: `{
					"label": "For my good pal, Maurice",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"24h0m0s",
					"maxFileBytes": 1048576,
					"maxFileUploads": 1
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: femtoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           femtoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.NewFileLifetimeInDays(1),
				MaxFileBytes:    makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads:  makeGuestUploadCountLimit(1),
			},
			status: http.StatusOK,
		},
		{
			description: "guest file expires in 30 days",
			payload: `{
					"label": "For my good pal, Maurice",
					"urlExpirationTime":"2030-01-02T03:04:25Z",
					"fileLifetime":"720h0m0s",
					"maxFileBytes": 1048576,
					"maxFileUploads": 1
				}`,
			currentTime: mustParseTime("2024-01-01T00:00:00Z"),
			expected: femtoshare.GuestLink{
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				Label:           femtoshare.GuestLinkLabel("For my good pal, Maurice"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.NewFileLifetimeInDays(30),
				MaxFileBytes:    makeGuestUploadMaxFileBytes(1048576),
				MaxFileUploads:  makeGuestUploadCountLimit(1),
			},
			status: http.StatusOK,
		},
		{
			description: "empty string",
			payload:     "",
			status:      http.StatusBadRequest,
		},
		{
			description: "empty payload",
			payload:     "{}",
			status:      http.StatusBadRequest,
		},
		{
			description: "invalid label field (non-string)",
			payload: `{
					"label": 5,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "invalid label field (too long)",
			payload: fmt.Sprintf(`{
					"label": "%s",
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": null
				}`, strings.Repeat("A", 201)),
			status: http.StatusBadRequest,
		},
		{
			description: "missing urlExpirationTime field",
			payload: `{
					"label": null,
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "invalid expirationTime field",
			payload: `{
					"label": null,
					"urlExpirationTime": 25,
					"maxFileBytes": null,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "negative maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": -5,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "decimal maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 1.5,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "too low a maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 1,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "zero maxFileBytes field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": 0,
					"maxFileUploads": null
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "negative maxFileUploads field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": -5
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "decimal maxFileUploads field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": 1.5
				}`,
			status: http.StatusBadRequest,
		},
		{
			description: "zero maxFileUploads field",
			payload: `{
					"label": null,
					"urlExpirationTime":"2025-01-01T00:00:00Z",
					"maxFileBytes": null,
					"maxFileUploads": 0
				}`,
			status: http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			now := tt.currentTime
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, nilConversionStateFunc, func() time.Time { return now })

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/guest-links",
				strings.NewReader(tt.payload),
			)
			req.Header.Add("Content-Type", "text/json")

			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			if tt.status != http.StatusOK {
				return
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("failed to read response body")
			}

			var response handlers.GuestLinkPostResponse
			err = json.Unmarshal(body, &response)
			if err != nil {
				t.Fatalf("response is not valid JSON: %v", body)
			}

			gl, err := dataStore.GetGuestLink(femtoshare.GuestLinkID(response.ID))
			if err != nil {
				t.Fatalf("failed to retrieve guest link from datastore: %v", err)
			}

			// Copy the ID, which we can't predict in advance.
			tt.expected.ID = femtoshare.GuestLinkID(response.ID)

			if got, want := gl, tt.expected; !reflect.DeepEqual(got, want) {
				t.Fatalf("guestLink=%+v, want=%+v", got, want)
			}
		})
	}
}

func makeGuestUploadMaxFileBytes(i uint64) femtoshare.GuestUploadMaxFileBytes {
	return femtoshare.GuestUploadMaxFileBytes(&i)
}

func makeGuestUploadCountLimit(i int) femtoshare.GuestUploadCountLimit {
	return femtoshare.GuestUploadCountLimit(&i)
}

func TestDeleteExistingGuestLink(t *testing.T) {
	dataStore := test_sqlite.New(t)
	dataStore.InsertGuestLink(femtoshare.GuestLink{
		ID:         femtoshare.GuestLinkID("abcdefgh23456789"),
		Created:    mustParseTime("2025-05-25T00:00:00Z"),
		UrlExpires: mustParseExpirationTime("2030-01-02T03:04:25Z"),
	})
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, nilConversionStateFunc, time.Now)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/guest-links/abcdefgh23456789",
		nil,
	)

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}

	_, err := dataStore.GetGuestLink(femtoshare.GuestLinkID("dummy-guest-link-id"))
	if _, ok := err.(store.GuestLinkNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted, got: %v", "abcdefgh23456789", err)
	}
}

func TestDeleteNonExistentGuestLink(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, nilConversionStateFunc, time.Now)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/guest-links/abcdefgh23456789",
		nil,
	)

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
}

func TestDeleteInvalidGuestLink(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, nilConversionStateFunc, time.Now)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/guest-links/i-am-an-invalid-link",
		nil,
	)

	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()

	if got, want := res.StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("status=%d, want=%d", got, want)
	}
}

func TestEnableDisableGuestLink(t *testing.T) {
	for _, tt := range []struct {
		description      string
		requestRoute     string
		guestLinkInStore femtoshare.GuestLink
		expected         femtoshare.GuestLink
		status           int
	}{
		{
			description:  "disabling active guest link succeeds",
			requestRoute: "/api/guest-links/abcdefgh23456789/disable",
			guestLinkInStore: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			expected: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "enabling inactive guest link succeeds",
			requestRoute: "/api/guest-links/abcdefgh23456789/enable",
			guestLinkInStore: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			expected: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "disabling guest link succeeds but has no effect when guest link is not disabled",
			requestRoute: "/api/guest-links/abcdefgh23456789/disable",
			guestLinkInStore: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			expected: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      true,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "enabling guest link succeeds but has no effect when guest link is not enabled",
			requestRoute: "/api/guest-links/abcdefgh23456789/enable",
			guestLinkInStore: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			expected: femtoshare.GuestLink{
				ID:              femtoshare.GuestLinkID("abcdefgh23456789"),
				Created:         mustParseTime("2024-01-01T00:00:00Z"),
				UrlExpires:      mustParseExpirationTime("2030-01-02T03:04:25Z"),
				MaxFileLifetime: femtoshare.FileLifetimeInfinite,
				MaxFileBytes:    femtoshare.GuestUploadUnlimitedFileSize,
				MaxFileUploads:  femtoshare.GuestUploadUnlimitedFileUploads,
				IsDisabled:      false,
			},
			status: http.StatusNoContent,
		},
		{
			description:  "disable a non-existent guest link",
			requestRoute: "/api/guest-links/abcdefgh23456789/disable",
			status:       http.StatusNotFound,
		}, {
			description:  "enable a non-existent guest link",
			requestRoute: "/api/guest-links/abcdefgh23456789/enable",
			status:       http.StatusNotFound,
		},
		{
			description:  "disable a guest link with an invalid ID",
			requestRoute: "/api/guest-links/i-am-an-invalid-link/disable",
			status:       http.StatusBadRequest,
		},
		{
			description:  "enable a guest link with an invalid ID",
			requestRoute: "/api/guest-links/i-am-an-invalid-link/enable",
			status:       http.StatusBadRequest,
		},
	} {
		t.Run(tt.description, func(t *testing.T) {
			dataStore := test_sqlite.New(t)

			if !tt.guestLinkInStore.Empty() {
				if err := dataStore.InsertGuestLink(tt.guestLinkInStore); err != nil {
					t.Fatalf("failed to insert dummy guest link: %v", err)
				}
			}

			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, nilConversionStateFunc, time.Now)

			req := httptest.NewRequest(http.MethodPut, tt.requestRoute, nil)
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}

			if tt.status != http.StatusNoContent {
				return
			}

			gl, err := dataStore.GetGuestLink(tt.guestLinkInStore.ID)
			if err != nil {
				t.Fatalf("failed to getGuestLink : %s", tt.guestLinkInStore.ID)
			}

			if got, want := gl, tt.expected; !reflect.DeepEqual(got, want) {
				t.Errorf("guestLink=%+v, want=%+v", got, want)
			}
		})
	}
}
