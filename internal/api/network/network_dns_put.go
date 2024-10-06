// Copyright (c) 2024 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package network

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/network/gen"
	"github.com/retr0h/osapi/internal/task"
)

// CreateAndMarshalChangeDNSActionFunc is a function variable that allows swapping the actual
// task.CreateAndMarshalChangeDNSAction function for a mock function during testing.
//
// This is a temporary solution to facilitate testing by allowing the function to be
// replaced with a mock implementation. In the future, this may be replaced by a
// better solution using dependency injection or test mocks.
var CreateAndMarshalChangeDNSActionFunc = task.CreateAndMarshalChangeDNSAction

// PutNetworkDNS put the network dns API endpoint.
func (n Network) PutNetworkDNS(
	ctx echo.Context,
) error {
	var newDNSConfig gen.PutNetworkDNSJSONRequestBody

	if err := ctx.Bind(&newDNSConfig); err != nil {
		return ctx.JSON(http.StatusBadRequest, gen.NetworkErrorResponse{
			Error: err.Error(),
		})
	}

	if err := validate.Struct(newDNSConfig); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		return ctx.JSON(http.StatusBadRequest, gen.NetworkErrorResponse{
			Error: validationErrors.Error(),
		})
	}

	var servers []string
	if newDNSConfig.Servers != nil {
		servers = *newDNSConfig.Servers
	}

	var searchDomains []string
	if newDNSConfig.SearchDomains != nil {
		searchDomains = *newDNSConfig.SearchDomains
	}

	data, err := CreateAndMarshalChangeDNSActionFunc(servers, searchDomains)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.NetworkErrorResponse{
			Error: err.Error(),
		})
	}

	_, err = n.TaskClientManager.PublishToStream(ctx.Request().Context(), data)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.NetworkErrorResponse{
			Error: err.Error(),
		})
	}

	return ctx.NoContent(http.StatusAccepted)
}
