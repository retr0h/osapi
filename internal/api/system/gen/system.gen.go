// Package gen provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package gen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	externalRef0 "github.com/retr0h/osapi/internal/api/common/gen"
)

const (
	BearerAuthScopes = "BearerAuth.Scopes"
)

// DiskResponse Local disk usage information.
type DiskResponse struct {
	// Free Free disk space in bytes.
	Free int `json:"free"`

	// Name Disk identifier, e.g., "/dev/sda1".
	Name string `json:"name"`

	// Total Total disk space in bytes.
	Total int `json:"total"`

	// Used Used disk space in bytes.
	Used int `json:"used"`
}

// DisksResponse List of local disk usage information.
type DisksResponse = []DiskResponse

// ErrorResponse defines model for ErrorResponse.
type ErrorResponse = externalRef0.ErrorResponse

// HostnameResponse The hostname of the system.
type HostnameResponse struct {
	// Hostname The system's hostname.
	Hostname string `json:"hostname"`
}

// LoadAverageResponse The system load averages for 1, 5, and 15 minutes.
type LoadAverageResponse struct {
	// N15min Load average for the last 15 minutes.
	N15min float32 `json:"15min"`

	// N1min Load average for the last 1 minute.
	N1min float32 `json:"1min"`

	// N5min Load average for the last 5 minutes.
	N5min float32 `json:"5min"`
}

// MemoryResponse Memory usage information.
type MemoryResponse struct {
	// Free Free memory in bytes.
	Free int `json:"free"`

	// Total Total memory in bytes.
	Total int `json:"total"`

	// Used Used memory in bytes.
	Used int `json:"used"`
}

// OSInfoResponse Operating system information.
type OSInfoResponse struct {
	// Distribution The name of the Linux distribution.
	Distribution string `json:"distribution"`

	// Version The version of the Linux distribution.
	Version string `json:"version"`
}

// SystemStatusResponse defines model for SystemStatusResponse.
type SystemStatusResponse struct {
	// Disks List of local disk usage information.
	Disks DisksResponse `json:"disks"`

	// Hostname The hostname of the system.
	Hostname string `json:"hostname"`

	// LoadAverage The system load averages for 1, 5, and 15 minutes.
	LoadAverage LoadAverageResponse `json:"load_average"`

	// Memory Memory usage information.
	Memory MemoryResponse `json:"memory"`

	// OsInfo Operating system information.
	OsInfo OSInfoResponse `json:"os_info"`

	// Uptime The uptime of the system.
	Uptime string `json:"uptime"`
}

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Retrieve system hostname
	// (GET /system/hostname)
	GetSystemHostname(ctx echo.Context) error
	// Retrieve system status
	// (GET /system/status)
	GetSystemStatus(ctx echo.Context) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GetSystemHostname converts echo context to params.
func (w *ServerInterfaceWrapper) GetSystemHostname(ctx echo.Context) error {
	var err error

	ctx.Set(BearerAuthScopes, []string{"read"})

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetSystemHostname(ctx)
	return err
}

// GetSystemStatus converts echo context to params.
func (w *ServerInterfaceWrapper) GetSystemStatus(ctx echo.Context) error {
	var err error

	ctx.Set(BearerAuthScopes, []string{"read"})

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetSystemStatus(ctx)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET(baseURL+"/system/hostname", wrapper.GetSystemHostname)
	router.GET(baseURL+"/system/status", wrapper.GetSystemStatus)

}

type GetSystemHostnameRequestObject struct {
}

type GetSystemHostnameResponseObject interface {
	VisitGetSystemHostnameResponse(w http.ResponseWriter) error
}

type GetSystemHostname200JSONResponse HostnameResponse

func (response GetSystemHostname200JSONResponse) VisitGetSystemHostnameResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GetSystemHostname401JSONResponse externalRef0.ErrorResponse

func (response GetSystemHostname401JSONResponse) VisitGetSystemHostnameResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)

	return json.NewEncoder(w).Encode(response)
}

type GetSystemHostname403JSONResponse externalRef0.ErrorResponse

func (response GetSystemHostname403JSONResponse) VisitGetSystemHostnameResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(403)

	return json.NewEncoder(w).Encode(response)
}

type GetSystemHostname500JSONResponse externalRef0.ErrorResponse

func (response GetSystemHostname500JSONResponse) VisitGetSystemHostnameResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)

	return json.NewEncoder(w).Encode(response)
}

type GetSystemStatusRequestObject struct {
}

type GetSystemStatusResponseObject interface {
	VisitGetSystemStatusResponse(w http.ResponseWriter) error
}

type GetSystemStatus200JSONResponse SystemStatusResponse

func (response GetSystemStatus200JSONResponse) VisitGetSystemStatusResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GetSystemStatus401JSONResponse externalRef0.ErrorResponse

func (response GetSystemStatus401JSONResponse) VisitGetSystemStatusResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(401)

	return json.NewEncoder(w).Encode(response)
}

type GetSystemStatus403JSONResponse externalRef0.ErrorResponse

func (response GetSystemStatus403JSONResponse) VisitGetSystemStatusResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(403)

	return json.NewEncoder(w).Encode(response)
}

type GetSystemStatus500JSONResponse externalRef0.ErrorResponse

func (response GetSystemStatus500JSONResponse) VisitGetSystemStatusResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)

	return json.NewEncoder(w).Encode(response)
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// Retrieve system hostname
	// (GET /system/hostname)
	GetSystemHostname(ctx context.Context, request GetSystemHostnameRequestObject) (GetSystemHostnameResponseObject, error)
	// Retrieve system status
	// (GET /system/status)
	GetSystemStatus(ctx context.Context, request GetSystemStatusRequestObject) (GetSystemStatusResponseObject, error)
}

type StrictHandlerFunc = strictecho.StrictEchoHandlerFunc
type StrictMiddlewareFunc = strictecho.StrictEchoMiddlewareFunc

func NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares}
}

type strictHandler struct {
	ssi         StrictServerInterface
	middlewares []StrictMiddlewareFunc
}

// GetSystemHostname operation middleware
func (sh *strictHandler) GetSystemHostname(ctx echo.Context) error {
	var request GetSystemHostnameRequestObject

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetSystemHostname(ctx.Request().Context(), request.(GetSystemHostnameRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetSystemHostname")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(GetSystemHostnameResponseObject); ok {
		return validResponse.VisitGetSystemHostnameResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}

// GetSystemStatus operation middleware
func (sh *strictHandler) GetSystemStatus(ctx echo.Context) error {
	var request GetSystemStatusRequestObject

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetSystemStatus(ctx.Request().Context(), request.(GetSystemStatusRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetSystemStatus")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(GetSystemStatusResponseObject); ok {
		return validResponse.VisitGetSystemStatusResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}
