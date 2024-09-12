// Package gen provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
package gen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// Disk Local disk usage information.
type Disk struct {
	// Free Free disk space in bytes.
	Free int `json:"free"`

	// Name Disk identifier, e.g., "/dev/sda1".
	Name string `json:"name"`

	// Total Total disk space in bytes.
	Total int `json:"total"`

	// Used Used disk space in bytes.
	Used int `json:"used"`
}

// Disks List of local disk usage information.
type Disks = []Disk

// LoadAverage The system load averages for 1, 5, and 15 minutes.
type LoadAverage struct {
	// N15min Load average for the last 15 minutes.
	N15min float32 `json:"15min"`

	// N1min Load average for the last 1 minute.
	N1min float32 `json:"1min"`

	// N5min Load average for the last 5 minutes.
	N5min float32 `json:"5min"`
}

// Memory Memory usage information.
type Memory struct {
	// Free Free memory in bytes.
	Free int `json:"free"`

	// Total Total memory in bytes.
	Total int `json:"total"`

	// Used Used memory in bytes.
	Used int `json:"used"`
}

// Pong defines model for Pong.
type Pong struct {
	Ping string `json:"ping"`
}

// QueueItem defines model for QueueItem.
type QueueItem struct {
	// Body String representation of the body of the queue item.
	Body *string `json:"body,omitempty"`

	// Created Creation timestamp of the queue item.
	Created *time.Time `json:"created,omitempty"`

	// Id Unique identifier of the queue item.
	Id *string `json:"id,omitempty"`

	// Received Number of times the queue item has been received.
	Received *int `json:"received,omitempty"`

	// Timeout Timeout timestamp for the queue item.
	Timeout *time.Time `json:"timeout,omitempty"`

	// Updated Last updated timestamp of the queue item.
	Updated *time.Time `json:"updated,omitempty"`
}

// QueueResponse defines model for QueueResponse.
type QueueResponse struct {
	Items *[]QueueItem `json:"items,omitempty"`

	// TotalItems The total number of queue items.
	TotalItems *int `json:"total_items,omitempty"`
}

// SystemStatus defines model for SystemStatus.
type SystemStatus struct {
	// Disks List of local disk usage information.
	Disks Disks `json:"disks"`

	// Hostname The hostname of the system.
	Hostname string `json:"hostname"`

	// LoadAverage The system load averages for 1, 5, and 15 minutes.
	LoadAverage LoadAverage `json:"load_average"`

	// Memory Memory usage information.
	Memory Memory `json:"memory"`

	// Uptime The uptime of the system.
	Uptime string `json:"uptime"`
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

// QueueErrorResponse defines model for queue.ErrorResponse.
type QueueErrorResponse struct {
	// Code The error code.
	Code int `json:"code"`

	// Details Additional details about the error, specifying which component failed.
	Details *string `json:"details,omitempty"`

	// Error A description of the error that occurred.
	Error string `json:"error"`
}

// SystemErrorResponse defines model for system.ErrorResponse.
type SystemErrorResponse struct {
	// Code The error code.
	Code int `json:"code"`

	// Details Additional details about the error, specifying which component failed.
	Details *string `json:"details,omitempty"`

	// Error A description of the error that occurred.
	Error string `json:"error"`
}

// GetQueueParams defines parameters for GetQueue.
type GetQueueParams struct {
	Limit  *int `form:"limit,omitempty" json:"limit,omitempty"`
	Offset *int `form:"offset,omitempty" json:"offset,omitempty"`
}

// PutNetworkDNSJSONRequestBody defines body for PutNetworkDNS for application/json ContentType.
type PutNetworkDNSJSONRequestBody = DNSConfigUpdate

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
	// GetNetworkDNS request
	GetNetworkDNS(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// PutNetworkDNSWithBody request with any body
	PutNetworkDNSWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	PutNetworkDNS(ctx context.Context, body PutNetworkDNSJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// DeleteNetworkDNSServerID request
	DeleteNetworkDNSServerID(ctx context.Context, serverId string, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetPing request
	GetPing(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetQueue request
	GetQueue(ctx context.Context, params *GetQueueParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetQueueID request
	GetQueueID(ctx context.Context, id string, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetSystemStatus request
	GetSystemStatus(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) GetNetworkDNS(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetNetworkDNSRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) PutNetworkDNSWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewPutNetworkDNSRequestWithBody(c.Server, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) PutNetworkDNS(ctx context.Context, body PutNetworkDNSJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewPutNetworkDNSRequest(c.Server, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) DeleteNetworkDNSServerID(ctx context.Context, serverId string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewDeleteNetworkDNSServerIDRequest(c.Server, serverId)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
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

func (c *Client) GetQueue(ctx context.Context, params *GetQueueParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetQueueRequest(c.Server, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetQueueID(ctx context.Context, id string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetQueueIDRequest(c.Server, id)
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

// NewGetNetworkDNSRequest generates requests for GetNetworkDNS
func NewGetNetworkDNSRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/network/dns")
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

// NewPutNetworkDNSRequest calls the generic PutNetworkDNS builder with application/json body
func NewPutNetworkDNSRequest(server string, body PutNetworkDNSJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewPutNetworkDNSRequestWithBody(server, "application/json", bodyReader)
}

// NewPutNetworkDNSRequestWithBody generates requests for PutNetworkDNS with any type of body
func NewPutNetworkDNSRequestWithBody(server string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/network/dns")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewDeleteNetworkDNSServerIDRequest generates requests for DeleteNetworkDNSServerID
func NewDeleteNetworkDNSServerIDRequest(server string, serverId string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "serverId", runtime.ParamLocationPath, serverId)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/network/dns/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
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

// NewGetQueueRequest generates requests for GetQueue
func NewGetQueueRequest(server string, params *GetQueueParams) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/queue")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	if params != nil {
		queryValues := queryURL.Query()

		if params.Limit != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "limit", runtime.ParamLocationQuery, *params.Limit); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		if params.Offset != nil {

			if queryFrag, err := runtime.StyleParamWithLocation("form", true, "offset", runtime.ParamLocationQuery, *params.Offset); err != nil {
				return nil, err
			} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
				return nil, err
			} else {
				for k, v := range parsed {
					for _, v2 := range v {
						queryValues.Add(k, v2)
					}
				}
			}

		}

		queryURL.RawQuery = queryValues.Encode()
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetQueueIDRequest generates requests for GetQueueID
func NewGetQueueIDRequest(server string, id string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "id", runtime.ParamLocationPath, id)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/queue/%s", pathParam0)
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
	// GetNetworkDNSWithResponse request
	GetNetworkDNSWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetNetworkDNSResponse, error)

	// PutNetworkDNSWithBodyWithResponse request with any body
	PutNetworkDNSWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*PutNetworkDNSResponse, error)

	PutNetworkDNSWithResponse(ctx context.Context, body PutNetworkDNSJSONRequestBody, reqEditors ...RequestEditorFn) (*PutNetworkDNSResponse, error)

	// DeleteNetworkDNSServerIDWithResponse request
	DeleteNetworkDNSServerIDWithResponse(ctx context.Context, serverId string, reqEditors ...RequestEditorFn) (*DeleteNetworkDNSServerIDResponse, error)

	// GetPingWithResponse request
	GetPingWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetPingResponse, error)

	// GetQueueWithResponse request
	GetQueueWithResponse(ctx context.Context, params *GetQueueParams, reqEditors ...RequestEditorFn) (*GetQueueResponse, error)

	// GetQueueIDWithResponse request
	GetQueueIDWithResponse(ctx context.Context, id string, reqEditors ...RequestEditorFn) (*GetQueueIDResponse, error)

	// GetSystemStatusWithResponse request
	GetSystemStatusWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetSystemStatusResponse, error)
}

type GetNetworkDNSResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *DNSConfig
	JSON500      *NetworkErrorResponse
}

// Status returns HTTPResponse.Status
func (r GetNetworkDNSResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetNetworkDNSResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type PutNetworkDNSResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON400      *NetworkErrorResponse
	JSON500      *NetworkErrorResponse
}

// Status returns HTTPResponse.Status
func (r PutNetworkDNSResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r PutNetworkDNSResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type DeleteNetworkDNSServerIDResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON404      *NetworkErrorResponse
	JSON500      *NetworkErrorResponse
}

// Status returns HTTPResponse.Status
func (r DeleteNetworkDNSServerIDResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r DeleteNetworkDNSServerIDResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
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

type GetQueueResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *QueueResponse
	JSON500      *QueueErrorResponse
}

// Status returns HTTPResponse.Status
func (r GetQueueResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetQueueResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetQueueIDResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *QueueItem
	JSON500      *QueueErrorResponse
}

// Status returns HTTPResponse.Status
func (r GetQueueIDResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetQueueIDResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetSystemStatusResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *SystemStatus
	JSON500      *SystemErrorResponse
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

// GetNetworkDNSWithResponse request returning *GetNetworkDNSResponse
func (c *ClientWithResponses) GetNetworkDNSWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetNetworkDNSResponse, error) {
	rsp, err := c.GetNetworkDNS(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetNetworkDNSResponse(rsp)
}

// PutNetworkDNSWithBodyWithResponse request with arbitrary body returning *PutNetworkDNSResponse
func (c *ClientWithResponses) PutNetworkDNSWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*PutNetworkDNSResponse, error) {
	rsp, err := c.PutNetworkDNSWithBody(ctx, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParsePutNetworkDNSResponse(rsp)
}

func (c *ClientWithResponses) PutNetworkDNSWithResponse(ctx context.Context, body PutNetworkDNSJSONRequestBody, reqEditors ...RequestEditorFn) (*PutNetworkDNSResponse, error) {
	rsp, err := c.PutNetworkDNS(ctx, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParsePutNetworkDNSResponse(rsp)
}

// DeleteNetworkDNSServerIDWithResponse request returning *DeleteNetworkDNSServerIDResponse
func (c *ClientWithResponses) DeleteNetworkDNSServerIDWithResponse(ctx context.Context, serverId string, reqEditors ...RequestEditorFn) (*DeleteNetworkDNSServerIDResponse, error) {
	rsp, err := c.DeleteNetworkDNSServerID(ctx, serverId, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseDeleteNetworkDNSServerIDResponse(rsp)
}

// GetPingWithResponse request returning *GetPingResponse
func (c *ClientWithResponses) GetPingWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetPingResponse, error) {
	rsp, err := c.GetPing(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetPingResponse(rsp)
}

// GetQueueWithResponse request returning *GetQueueResponse
func (c *ClientWithResponses) GetQueueWithResponse(ctx context.Context, params *GetQueueParams, reqEditors ...RequestEditorFn) (*GetQueueResponse, error) {
	rsp, err := c.GetQueue(ctx, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetQueueResponse(rsp)
}

// GetQueueIDWithResponse request returning *GetQueueIDResponse
func (c *ClientWithResponses) GetQueueIDWithResponse(ctx context.Context, id string, reqEditors ...RequestEditorFn) (*GetQueueIDResponse, error) {
	rsp, err := c.GetQueueID(ctx, id, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetQueueIDResponse(rsp)
}

// GetSystemStatusWithResponse request returning *GetSystemStatusResponse
func (c *ClientWithResponses) GetSystemStatusWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetSystemStatusResponse, error) {
	rsp, err := c.GetSystemStatus(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetSystemStatusResponse(rsp)
}

// ParseGetNetworkDNSResponse parses an HTTP response from a GetNetworkDNSWithResponse call
func ParseGetNetworkDNSResponse(rsp *http.Response) (*GetNetworkDNSResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetNetworkDNSResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest DNSConfig
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest NetworkErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParsePutNetworkDNSResponse parses an HTTP response from a PutNetworkDNSWithResponse call
func ParsePutNetworkDNSResponse(rsp *http.Response) (*PutNetworkDNSResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &PutNetworkDNSResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest NetworkErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest NetworkErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseDeleteNetworkDNSServerIDResponse parses an HTTP response from a DeleteNetworkDNSServerIDWithResponse call
func ParseDeleteNetworkDNSServerIDResponse(rsp *http.Response) (*DeleteNetworkDNSServerIDResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &DeleteNetworkDNSServerIDResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 404:
		var dest NetworkErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON404 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest NetworkErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
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

// ParseGetQueueResponse parses an HTTP response from a GetQueueWithResponse call
func ParseGetQueueResponse(rsp *http.Response) (*GetQueueResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetQueueResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest QueueResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest QueueErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}

// ParseGetQueueIDResponse parses an HTTP response from a GetQueueIDWithResponse call
func ParseGetQueueIDResponse(rsp *http.Response) (*GetQueueIDResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetQueueIDResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest QueueItem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 500:
		var dest QueueErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

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
		var dest SystemErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON500 = &dest

	}

	return response, nil
}
