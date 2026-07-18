package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oapi-codegen/oapi-codegen/v2/examples/petstore-expanded/stdhttp/api"
)

func TestFindPetsReturnsEmptyArray(t *testing.T) {
	if pets := NewStore().FindPets(nil, nil); pets == nil {
		t.Fatal("FindPets() returned nil; want an empty array")
	}
}

func TestValidationErrorUsesErrorSchema(t *testing.T) {
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	response := httptest.NewRecorder()
	srv.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/pets?limit=invalid", nil))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusBadRequest)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q; want application/json", contentType)
	}

	var responseError api.Error
	if err := json.NewDecoder(response.Body).Decode(&responseError); err != nil {
		t.Fatalf("decode Error response: %v", err)
	}
	if responseError.Code != http.StatusBadRequest || responseError.Message == "" {
		t.Fatalf("Error response = %#v; want code 400 and non-empty message", responseError)
	}
}
