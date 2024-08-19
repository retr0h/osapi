// Package client provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrorResponse defines model for ErrorResponse.
type ErrorResponse struct {
	// Code The error code.
	Code int `json:"code"`

	// Details Additional details about the error, specifying which component failed.
	Details *string `json:"details,omitempty"`

	// Error A description of the error that occurred.
	Error string `json:"error"`
}

// Pong defines model for Pong.
type Pong struct {
	Ping string `json:"ping"`
}

// SystemStatus defines model for SystemStatus.
type SystemStatus struct {
	// Disk Disk usage information.
	Disk struct {
		// Free Free disk space in bytes.
		Free int `json:"free"`

		// Total Total disk space in bytes.
		Total int `json:"total"`

		// Used Used disk space in bytes.
		Used int `json:"used"`
	} `json:"disk"`

	// Hostname The hostname of the system.
	Hostname string `json:"hostname"`

	// LoadAverage The system load averages for 1, 5, and 15 minutes.
	LoadAverage struct {
		// N15min Load average for the last 15 minutes.
		N15min float32 `json:"15min"`

		// N1min Load average for the last 1 minute.
		N1min float32 `json:"1min"`

		// N5min Load average for the last 5 minutes.
		N5min float32 `json:"5min"`
	} `json:"load_average"`

	// Memory Memory usage information.
	Memory struct {
		// Free Free memory in bytes.
		Free int `json:"free"`

		// Total Total memory in bytes.
		Total int `json:"total"`

		// Used Used memory in bytes.
		Used int `json:"used"`
	} `json:"memory"`

	// Uptime The uptime of the system.
	Uptime string `json:"uptime"`
}

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// GetPing request
	GetPing(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetSystemStatus request
	GetSystemStatus(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) GetPing(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetPingRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetSystemStatus(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetSystemStatusRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetPingRequest generates requests for GetPing
func NewGetPingRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/ping")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetSystemStatusRequest generates requests for GetSystemStatus
func NewGetSystemStatusRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/system/status")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// GetPingWithResponse request
	GetPingWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetPingResponse, error)

	// GetSystemStatusWithResponse request
	GetSystemStatusWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetSystemStatusResponse, error)
}

type GetPingResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *Pong
}

// Status returns HTTPResponse.Status
func (r GetPingResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetPingResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetSystemStatusResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *SystemStatus
	JSON500      *ErrorResponse
}

// Status returns HTTPResponse.Status
func (r GetSystemStatusResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetSystemStatusResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// GetPingWithResponse request returning *GetPingResponse
func (c *ClientWithResponses) GetPingWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetPingResponse, error) {
	rsp, err := c.GetPing(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetPingResponse(rsp)
}

// GetSystemStatusWithResponse request returning *GetSystemStatusResponse
func (c *ClientWithResponses) GetSystemStatusWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetSystemStatusResponse, error) {
	rsp, err := c.GetSystemStatus(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSystemStatusResponse(rsp)
}

// ParseGetPingResponse parses an HTTP response from a GetPingWithResponse call
func ParseGetPingResponse(rsp *http.Response) (*GetPingResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetPingResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest Pong
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	}

	return response, nil
}

// ParseGetSystemStatusResponse parses an HTTP response from a GetSystemStatusWithResponse call
func ParseGetSystemStatusResponse(rsp *http.Response) (*GetSystemStatusResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetSystemStatusResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest SystemStatus
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest ErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}
