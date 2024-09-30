// Package gen provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package gen

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime"
)

// DNSConfig defines model for DNSConfig.
type DNSConfig struct {
	// SearchDomains List of search domains.
	SearchDomains *[]string `json:"search_domains,omitempty"`

	// Servers List of configured DNS servers.
	Servers *[]string `json:"servers,omitempty"`
}

// DNSConfigUpdate defines model for DNSConfigUpdate.
type DNSConfigUpdate struct {
	// SearchDomains New list of search domains to configure.
	SearchDomains *[]string `json:"search_domains,omitempty"`

	// Servers New list of DNS servers to configure.
	Servers *[]string `json:"servers,omitempty"`
}

// PingResponse defines model for PingResponse.
type PingResponse struct {
	// AvgRtt Average round-trip time as a string in Go's time.Duration format.
	AvgRtt *string `json:"avg_rtt,omitempty"`

	// MaxRtt Maximum round-trip time as a string in Go's time.Duration format.
	MaxRtt *string `json:"max_rtt,omitempty"`

	// MinRtt Minimum round-trip time as a string in Go's time.Duration format.
	MinRtt *string `json:"min_rtt,omitempty"`

	// PacketLoss Percentage of packet loss.
	PacketLoss *float64 `json:"packet_loss,omitempty"`

	// PacketsReceived Number of packets received.
	PacketsReceived *int `json:"packets_received,omitempty"`

	// PacketsSent Number of packets sent.
	PacketsSent *int `json:"packets_sent,omitempty"`
}

// NetworkErrorResponse defines model for network.ErrorResponse.
type NetworkErrorResponse struct {
	// Code The error code.
	Code int `json:"code"`

	// Details Additional details about the error, specifying which component failed.
	Details *string `json:"details,omitempty"`

	// Error A description of the error that occurred.
	Error string `json:"error"`
}

// PostNetworkPingJSONBody defines parameters for PostNetworkPing.
type PostNetworkPingJSONBody struct {
	// Address The IP address or hostname of the server to ping.
	Address string `json:"address"`
}

// PutNetworkDNSJSONRequestBody defines body for PutNetworkDNS for application/json ContentType.
type PutNetworkDNSJSONRequestBody = DNSConfigUpdate

// PostNetworkPingJSONRequestBody defines body for PostNetworkPing for application/json ContentType.
type PostNetworkPingJSONRequestBody PostNetworkPingJSONBody

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// List DNS servers
	// (GET /network/dns)
	GetNetworkDNS(ctx echo.Context) error
	// Update DNS servers
	// (PUT /network/dns)
	PutNetworkDNS(ctx echo.Context) error
	// Delete a DNS server
	// (DELETE /network/dns/{serverId})
	DeleteNetworkDNSServerID(ctx echo.Context, serverId string) error
	// Ping a remote server
	// (POST /network/ping)
	PostNetworkPing(ctx echo.Context) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GetNetworkDNS converts echo context to params.
func (w *ServerInterfaceWrapper) GetNetworkDNS(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetNetworkDNS(ctx)
	return err
}

// PutNetworkDNS converts echo context to params.
func (w *ServerInterfaceWrapper) PutNetworkDNS(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.PutNetworkDNS(ctx)
	return err
}

// DeleteNetworkDNSServerID converts echo context to params.
func (w *ServerInterfaceWrapper) DeleteNetworkDNSServerID(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "serverId" -------------
	var serverId string

	err = runtime.BindStyledParameterWithOptions("simple", "serverId", ctx.Param("serverId"), &serverId, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter serverId: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.DeleteNetworkDNSServerID(ctx, serverId)
	return err
}

// PostNetworkPing converts echo context to params.
func (w *ServerInterfaceWrapper) PostNetworkPing(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.PostNetworkPing(ctx)
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

	router.GET(baseURL+"/network/dns", wrapper.GetNetworkDNS)
	router.PUT(baseURL+"/network/dns", wrapper.PutNetworkDNS)
	router.DELETE(baseURL+"/network/dns/:serverId", wrapper.DeleteNetworkDNSServerID)
	router.POST(baseURL+"/network/ping", wrapper.PostNetworkPing)

}