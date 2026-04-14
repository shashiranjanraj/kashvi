package router_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shashiranjanraj/kashvi/pkg/apperror"
	"github.com/shashiranjanraj/kashvi/pkg/router"
)

type envelope map[string]any

func TestWrap(t *testing.T) {
	tests := []struct {
		name           string
		handlerError   error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "NoError",
			handlerError:   nil,
			expectedStatus: http.StatusOK,
			expectedMsg:    "",
		},
		{
			name:           "AppError_NotFound",
			handlerError:   apperror.NotFound("User not found"),
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "User not found",
		},
		{
			name:           "AppError_BadRequest",
			handlerError:   apperror.BadRequest("Invalid input"),
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid input",
		},
		{
			name:           "GenericError",
			handlerError:   errors.New("some random db error"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := router.Wrap(func(w http.ResponseWriter, r *http.Request) error {
				if tt.handlerError == nil {
					w.WriteHeader(http.StatusOK)
					return nil
				}
				return tt.handlerError
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d; got %d", tt.expectedStatus, rr.Code)
			}

			if tt.handlerError != nil {
				var response map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to parse response JSON: %v", err)
				}
				if msg, ok := response["message"].(string); !ok || msg != tt.expectedMsg {
					t.Errorf("expected message %q; got %q", tt.expectedMsg, msg)
				}
			}
		})
	}
}
