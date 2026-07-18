package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/oapi-codegen/oapi-codegen/v2/examples/petstore-expanded/stdhttp/api"
)

// NewServer creates a fully configured *http.Server with the petstore handler
// and OpenAPI validation middleware. The caller should set Addr before calling
// ListenAndServe, or provide a net.Listener and call Serve.
func NewServer() (*http.Server, error) {
	swagger, err := api.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("error loading swagger spec: %w", err)
	}

	swagger.Servers = nil

	petStore := NewPetStore()

	r := http.NewServeMux()
	errorHandler := func(w http.ResponseWriter, message string, statusCode int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(api.Error{Code: int32(statusCode), Message: message})
	}

	api.HandlerWithOptions(petStore, api.StdHTTPServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			errorHandler(w, err.Error(), http.StatusBadRequest)
		},
	})

	h := middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
		ErrorHandler: errorHandler,
	})(r)

	return &http.Server{Handler: h}, nil
}
