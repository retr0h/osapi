// Copyright (c) 2026 John Dewey

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

package docker_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/common"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	dockerprov "github.com/osapi-io/osapi/internal/provider/container/docker"
	dockermocks "github.com/osapi-io/osapi/internal/provider/container/docker/mocks"
)

// newHijackedResponse creates a HijackedResponse with the given content
// suitable for testing. It uses a pipe-based net.Conn so Close() does not panic.
func newHijackedResponse(
	content string,
) types.HijackedResponse {
	serverConn, clientConn := net.Pipe()
	_ = serverConn.Close()

	return types.HijackedResponse{
		Conn:   clientConn,
		Reader: bufio.NewReader(strings.NewReader(content)),
	}
}

// errorConn is a net.Conn that always returns an error on Read.
type errorConn struct {
	net.Conn
}

func (c *errorConn) Read(
	_ []byte,
) (int, error) {
	return 0, errors.New("forced read error")
}

func (c *errorConn) Close() error {
	return nil
}

func newErrorHijackedResponse() types.HijackedResponse {
	return types.HijackedResponse{
		Conn:   &errorConn{},
		Reader: bufio.NewReader(&errorConn{}),
	}
}

type DockerDriverPublicTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *DockerDriverPublicTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *DockerDriverPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		setupEnv     func() (cleanup func())
		validateFunc func(d dockerprov.Provider, err error)
	}{
		{
			name: "returns non-nil driver",
			validateFunc: func(
				d dockerprov.Provider,
				err error,
			) {
				s.NoError(err)
				s.NotNil(d)
			},
		},
		{
			name: "returns error when docker client creation fails",
			setupEnv: func() (cleanup func()) {
				orig, existed := os.LookupEnv("DOCKER_HOST")
				_ = os.Setenv("DOCKER_HOST", "tcp://invalid:::::port")

				return func() {
					if existed {
						_ = os.Setenv("DOCKER_HOST", orig)
					} else {
						_ = os.Unsetenv("DOCKER_HOST")
					}
				}
			},
			validateFunc: func(
				d dockerprov.Provider,
				err error,
			) {
				s.Error(err)
				s.Contains(err.Error(), "create docker client")
				s.Nil(d)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setupEnv != nil {
				cleanup := tt.setupEnv()
				defer cleanup()
			}

			d, err := dockerprov.New()
			tt.validateFunc(d, err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestPing() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		validateFunc func(err error)
	}{
		{
			name: "successful ping",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					Ping(gomock.Any()).
					Return(types.Ping{APIVersion: "1.45"}, nil)
				return m
			},
			validateFunc: func(
				err error,
			) {
				s.NoError(err)
			},
		},
		{
			name: "returns error when ping fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					Ping(gomock.Any()).
					Return(types.Ping{}, fmt.Errorf("connection refused"))
				return m
			},
			validateFunc: func(
				err error,
			) {
				s.Error(err)
				s.Contains(err.Error(), "ping docker daemon")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			err := d.Ping(s.ctx)
			tt.validateFunc(err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestCreate() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		params       dockerprov.CreateParams
		validateFunc func(c *dockerprov.Container, err error)
	}{
		{
			name: "successful container creation",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(container.CreateResponse{ID: "test-id"}, nil)
				return m
			},
			params: dockerprov.CreateParams{
				Image: "nginx:latest",
				Name:  "test-nginx",
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("test-id", c.ID)
			},
		},
		{
			name: "successful container creation with auto-start",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(container.CreateResponse{ID: "test-id-auto"}, nil)
				m.EXPECT().
					ContainerStart(gomock.Any(), "test-id-auto", gomock.Any()).
					Return(nil)
				return m
			},
			params: dockerprov.CreateParams{
				Image:     "nginx:latest",
				Name:      "test-nginx-auto",
				AutoStart: true,
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("test-id-auto", c.ID)
			},
		},
		{
			name: "with command set",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.AssignableToTypeOf(&container.Config{}),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						config *container.Config,
						_ *container.HostConfig,
						_ *network.NetworkingConfig,
						_ *ocispec.Platform,
						_ string,
					) (container.CreateResponse, error) {
						s.Equal([]string{"echo", "hello"}, []string(config.Cmd))
						return container.CreateResponse{ID: "cmd-id"}, nil
					})
				return m
			},
			params: dockerprov.CreateParams{
				Image:   "alpine:latest",
				Name:    "test-cmd",
				Command: []string{"echo", "hello"},
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("cmd-id", c.ID)
			},
		},
		{
			name: "with env map",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						config *container.Config,
						_ *container.HostConfig,
						_ *network.NetworkingConfig,
						_ *ocispec.Platform,
						_ string,
					) (container.CreateResponse, error) {
						s.Len(config.Env, 1)
						s.Contains(config.Env[0], "FOO=bar")
						return container.CreateResponse{ID: "env-id"}, nil
					})
				return m
			},
			params: dockerprov.CreateParams{
				Image: "alpine:latest",
				Name:  "test-env",
				Env:   map[string]string{"FOO": "bar"},
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("env-id", c.ID)
			},
		},
		{
			name: "with ports",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						config *container.Config,
						hostConfig *container.HostConfig,
						_ *network.NetworkingConfig,
						_ *ocispec.Platform,
						_ string,
					) (container.CreateResponse, error) {
						s.NotNil(config.ExposedPorts)
						s.NotNil(hostConfig.PortBindings)
						containerPort := nat.Port("80/tcp")
						s.Contains(hostConfig.PortBindings, containerPort)
						s.Equal("8080", hostConfig.PortBindings[containerPort][0].HostPort)
						return container.CreateResponse{ID: "ports-id"}, nil
					})
				return m
			},
			params: dockerprov.CreateParams{
				Image: "nginx:latest",
				Name:  "test-ports",
				Ports: []dockerprov.PortMapping{
					{Host: 8080, Container: 80},
				},
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("ports-id", c.ID)
			},
		},
		{
			name: "with volumes",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						_ *container.Config,
						hostConfig *container.HostConfig,
						_ *network.NetworkingConfig,
						_ *ocispec.Platform,
						_ string,
					) (container.CreateResponse, error) {
						s.Len(hostConfig.Mounts, 1)
						s.Equal("/host/data", hostConfig.Mounts[0].Source)
						s.Equal("/container/data", hostConfig.Mounts[0].Target)
						return container.CreateResponse{ID: "vols-id"}, nil
					})
				return m
			},
			params: dockerprov.CreateParams{
				Image: "nginx:latest",
				Name:  "test-vols",
				Volumes: []dockerprov.VolumeMapping{
					{Host: "/host/data", Container: "/container/data"},
				},
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("vols-id", c.ID)
			},
		},
		{
			name: "with hostname",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.AssignableToTypeOf(&container.Config{}),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						config *container.Config,
						_ *container.HostConfig,
						_ *network.NetworkingConfig,
						_ *ocispec.Platform,
						_ string,
					) (container.CreateResponse, error) {
						s.Equal("web-01", config.Hostname)
						return container.CreateResponse{ID: "hostname-id"}, nil
					})
				return m
			},
			params: dockerprov.CreateParams{
				Image:    "nginx:latest",
				Name:     "test-hostname",
				Hostname: "web-01",
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("hostname-id", c.ID)
			},
		},
		{
			name: "with dns",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.AssignableToTypeOf(&container.HostConfig{}),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						_ *container.Config,
						hostConfig *container.HostConfig,
						_ *network.NetworkingConfig,
						_ *ocispec.Platform,
						_ string,
					) (container.CreateResponse, error) {
						s.Equal([]string{"8.8.8.8", "8.8.4.4"}, hostConfig.DNS)
						return container.CreateResponse{ID: "dns-id"}, nil
					})
				return m
			},
			params: dockerprov.CreateParams{
				Image: "nginx:latest",
				Name:  "test-dns",
				DNS:   []string{"8.8.8.8", "8.8.4.4"},
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("dns-id", c.ID)
			},
		},
		{
			name: "with hostname and dns",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.AssignableToTypeOf(&container.Config{}),
						gomock.AssignableToTypeOf(&container.HostConfig{}),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						config *container.Config,
						hostConfig *container.HostConfig,
						_ *network.NetworkingConfig,
						_ *ocispec.Platform,
						_ string,
					) (container.CreateResponse, error) {
						s.Equal("web-01", config.Hostname)
						s.Equal([]string{"1.1.1.1"}, hostConfig.DNS)
						return container.CreateResponse{ID: "both-id"}, nil
					})
				return m
			},
			params: dockerprov.CreateParams{
				Image:    "nginx:latest",
				Name:     "test-both",
				Hostname: "web-01",
				DNS:      []string{"1.1.1.1"},
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.NotNil(c)
				s.Equal("both-id", c.ID)
			},
		},
		{
			name: "returns error when container create fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(container.CreateResponse{}, fmt.Errorf("image not found"))
				return m
			},
			params: dockerprov.CreateParams{
				Image: "nonexistent:latest",
				Name:  "test-fail",
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.Error(err)
				s.Nil(c)
				s.Contains(err.Error(), "create container")
			},
		},
		{
			name: "returns error when auto-start fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerCreate(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).
					Return(container.CreateResponse{ID: "autostart-fail-id"}, nil)
				m.EXPECT().
					ContainerStart(gomock.Any(), "autostart-fail-id", gomock.Any()).
					Return(fmt.Errorf("start failed"))
				return m
			},
			params: dockerprov.CreateParams{
				Image:     "nginx:latest",
				Name:      "test-autostart-fail",
				AutoStart: true,
			},
			validateFunc: func(
				c *dockerprov.Container,
				err error,
			) {
				s.Error(err)
				s.Nil(c)
				s.Contains(err.Error(), "auto-start container")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			c, err := d.Create(s.ctx, tt.params)
			tt.validateFunc(c, err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestStart() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		containerID  string
		validateFunc func(err error)
	}{
		{
			name: "successful start",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerStart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				return m
			},
			containerID: "test-id",
			validateFunc: func(
				err error,
			) {
				s.NoError(err)
			},
		},
		{
			name: "returns error when start fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerStart(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("container not found"))
				return m
			},
			containerID: "missing-id",
			validateFunc: func(
				err error,
			) {
				s.Error(err)
				s.Contains(err.Error(), "start container")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			_, err := d.Start(s.ctx, tt.containerID)
			tt.validateFunc(err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestStop() {
	timeout := 10 * time.Second
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		containerID  string
		timeout      *time.Duration
		validateFunc func(err error)
	}{
		{
			name: "successful stop with timeout",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerStop(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						options container.StopOptions,
					) error {
						s.NotNil(options.Timeout)
						return nil
					})
				return m
			},
			containerID: "test-id",
			timeout:     &timeout,
			validateFunc: func(
				err error,
			) {
				s.NoError(err)
			},
		},
		{
			name: "successful stop with nil timeout",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerStop(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						options container.StopOptions,
					) error {
						s.Nil(options.Timeout)
						return nil
					})
				return m
			},
			containerID: "test-id",
			timeout:     nil,
			validateFunc: func(
				err error,
			) {
				s.NoError(err)
			},
		},
		{
			name: "returns error when stop fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerStop(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("container not running"))
				return m
			},
			containerID: "stopped-id",
			timeout:     nil,
			validateFunc: func(
				err error,
			) {
				s.Error(err)
				s.Contains(err.Error(), "stop container")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			_, err := d.Stop(s.ctx, tt.containerID, tt.timeout)
			tt.validateFunc(err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestRemove() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		containerID  string
		force        bool
		validateFunc func(err error)
	}{
		{
			name: "successful remove with force",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerRemove(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				return m
			},
			containerID: "test-id",
			force:       true,
			validateFunc: func(
				err error,
			) {
				s.NoError(err)
			},
		},
		{
			name: "returns error when remove fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerRemove(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("container is running"))
				return m
			},
			containerID: "running-id",
			force:       false,
			validateFunc: func(
				err error,
			) {
				s.Error(err)
				s.Contains(err.Error(), "remove container")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			_, err := d.Remove(s.ctx, tt.containerID, tt.force)
			tt.validateFunc(err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		params       dockerprov.ListParams
		validateFunc func(containers []dockerprov.Container, err error)
	}{
		{
			name: "successful list all containers",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerList(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						options container.ListOptions,
					) ([]container.Summary, error) {
						s.True(options.All)
						return []container.Summary{
							{
								ID:      "test-id-1",
								Names:   []string{"/test-container-1"},
								Image:   "nginx:latest",
								State:   "running",
								Created: time.Now().Unix(),
							},
						}, nil
					})
				return m
			},
			params: dockerprov.ListParams{State: "all"},
			validateFunc: func(
				containers []dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.Len(containers, 1)
				s.Equal("test-id-1", containers[0].ID)
			},
		},
		{
			name: "state running sets all to false",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerList(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						options container.ListOptions,
					) ([]container.Summary, error) {
						s.False(options.All)
						return []container.Summary{}, nil
					})
				return m
			},
			params: dockerprov.ListParams{State: "running"},
			validateFunc: func(
				containers []dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.Empty(containers)
			},
		},
		{
			name: "state stopped sets filter",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerList(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						options container.ListOptions,
					) ([]container.Summary, error) {
						s.True(options.All)
						s.NotNil(options.Filters)
						return []container.Summary{}, nil
					})
				return m
			},
			params: dockerprov.ListParams{State: "stopped"},
			validateFunc: func(
				containers []dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.Empty(containers)
			},
		},
		{
			name: "default empty state",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerList(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						options container.ListOptions,
					) ([]container.Summary, error) {
						s.False(options.All)
						return []container.Summary{}, nil
					})
				return m
			},
			params: dockerprov.ListParams{State: ""},
			validateFunc: func(
				containers []dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.Empty(containers)
			},
		},
		{
			name: "with limit",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerList(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						options container.ListOptions,
					) ([]container.Summary, error) {
						s.Equal(5, options.Limit)
						return []container.Summary{}, nil
					})
				return m
			},
			params: dockerprov.ListParams{State: "all", Limit: 5},
			validateFunc: func(
				containers []dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.Empty(containers)
			},
		},
		{
			name: "container with empty names slice",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerList(gomock.Any(), gomock.Any()).
					Return([]container.Summary{
						{
							ID:      "no-name-id",
							Names:   []string{},
							Image:   "alpine:latest",
							State:   "running",
							Created: time.Now().Unix(),
						},
					}, nil)
				return m
			},
			params: dockerprov.ListParams{State: "all"},
			validateFunc: func(
				containers []dockerprov.Container,
				err error,
			) {
				s.NoError(err)
				s.Len(containers, 1)
				s.Equal("", containers[0].Name)
			},
		},
		{
			name: "returns error when list fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerList(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("daemon error"))
				return m
			},
			params: dockerprov.ListParams{State: "all"},
			validateFunc: func(
				containers []dockerprov.Container,
				err error,
			) {
				s.Error(err)
				s.Nil(containers)
				s.Contains(err.Error(), "list containers")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			containers, err := d.List(s.ctx, tt.params)
			tt.validateFunc(containers, err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestInspect() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		containerID  string
		validateFunc func(detail *dockerprov.ContainerDetail, err error)
	}{
		{
			name: "successful inspect",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							ID:      "test-id",
							Name:    "/test-container",
							State:   &container.State{Status: "running"},
							Created: time.Now().Format(time.RFC3339Nano),
						},
						Config: &container.Config{Image: "nginx:latest"},
					}, nil)
				return m
			},
			containerID: "test-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.NoError(err)
				s.NotNil(detail)
				s.Equal("test-id", detail.ID)
				s.Equal("running", detail.State)
			},
		},
		{
			name: "nil state returns unknown",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							ID:      "nil-state-id",
							Name:    "/nil-state",
							State:   nil,
							Created: time.Now().Format(time.RFC3339Nano),
						},
						Config: &container.Config{Image: "nginx:latest"},
					}, nil)
				return m
			},
			containerID: "nil-state-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.NoError(err)
				s.NotNil(detail)
				s.Equal("unknown", detail.State)
			},
		},
		{
			name: "invalid created timestamp falls back to now",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							ID:      "bad-time-id",
							Name:    "/bad-time",
							State:   &container.State{Status: "running"},
							Created: "not-a-valid-timestamp",
						},
						Config: &container.Config{Image: "nginx:latest"},
					}, nil)
				return m
			},
			containerID: "bad-time-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.NoError(err)
				s.NotNil(detail)
				s.WithinDuration(time.Now(), detail.Created, 5*time.Second)
			},
		},
		{
			name: "with network settings",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							ID:      "net-id",
							Name:    "/net-container",
							State:   &container.State{Status: "running"},
							Created: time.Now().Format(time.RFC3339Nano),
						},
						Config: &container.Config{Image: "nginx:latest"},
						NetworkSettings: &container.NetworkSettings{
							Networks: map[string]*network.EndpointSettings{
								"bridge": {
									IPAddress: "172.17.0.2",
									Gateway:   "172.17.0.1",
								},
							},
						},
					}, nil)
				return m
			},
			containerID: "net-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.NoError(err)
				s.NotNil(detail)
				s.NotNil(detail.NetworkSettings)
				s.Equal("172.17.0.2", detail.NetworkSettings.IPAddress)
				s.Equal("172.17.0.1", detail.NetworkSettings.Gateway)
			},
		},
		{
			name: "with port bindings",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							ID:   "ports-id",
							Name: "/ports-container",
							State: &container.State{
								Status: "running",
							},
							Created: time.Now().Format(time.RFC3339Nano),
							HostConfig: &container.HostConfig{
								PortBindings: nat.PortMap{
									"80/tcp": []nat.PortBinding{
										{HostPort: "8080"},
									},
								},
							},
						},
						Config: &container.Config{Image: "nginx:latest"},
					}, nil)
				return m
			},
			containerID: "ports-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.NoError(err)
				s.NotNil(detail)
				s.Len(detail.Ports, 1)
				s.Equal(8080, detail.Ports[0].Host)
				s.Equal(80, detail.Ports[0].Container)
			},
		},
		{
			name: "with mounts",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							ID:      "mounts-id",
							Name:    "/mounts-container",
							State:   &container.State{Status: "running"},
							Created: time.Now().Format(time.RFC3339Nano),
						},
						Config: &container.Config{Image: "nginx:latest"},
						Mounts: []container.MountPoint{
							{
								Source:      "/host/path",
								Destination: "/container/path",
							},
						},
					}, nil)
				return m
			},
			containerID: "mounts-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.NoError(err)
				s.NotNil(detail)
				s.Len(detail.Mounts, 1)
				s.Equal("/host/path", detail.Mounts[0].Host)
				s.Equal("/container/path", detail.Mounts[0].Container)
			},
		},
		{
			name: "with health status",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{
						ContainerJSONBase: &container.ContainerJSONBase{
							ID:   "health-id",
							Name: "/health-container",
							State: &container.State{
								Status: "running",
								Health: &container.Health{
									Status: "healthy",
								},
							},
							Created: time.Now().Format(time.RFC3339Nano),
						},
						Config: &container.Config{Image: "nginx:latest"},
					}, nil)
				return m
			},
			containerID: "health-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.NoError(err)
				s.NotNil(detail)
				s.Equal("healthy", detail.Health)
			},
		},
		{
			name: "returns error when inspect fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerInspect(gomock.Any(), gomock.Any()).
					Return(container.InspectResponse{}, fmt.Errorf("no such container"))
				return m
			},
			containerID: "missing-id",
			validateFunc: func(
				detail *dockerprov.ContainerDetail,
				err error,
			) {
				s.Error(err)
				s.Nil(detail)
				s.Contains(err.Error(), "inspect container")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			detail, err := d.Inspect(s.ctx, tt.containerID)
			tt.validateFunc(detail, err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestExec() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		containerID  string
		params       dockerprov.ExecParams
		validateFunc func(result *dockerprov.ExecResult, err error)
	}{
		{
			name: "successful exec",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerExecCreate(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						config container.ExecOptions,
					) (common.IDResponse, error) {
						s.Equal([]string{"echo", "hello"}, config.Cmd)
						s.True(config.AttachStdout)
						s.True(config.AttachStderr)
						return common.IDResponse{ID: "exec-id"}, nil
					})
				m.EXPECT().
					ContainerExecAttach(gomock.Any(), "exec-id", gomock.Any()).
					Return(newHijackedResponse("hello\n"), nil)
				m.EXPECT().
					ContainerExecInspect(gomock.Any(), "exec-id").
					Return(container.ExecInspect{ExitCode: 0}, nil)
				return m
			},
			containerID: "test-id",
			params: dockerprov.ExecParams{
				Command: []string{"echo", "hello"},
			},
			validateFunc: func(
				result *dockerprov.ExecResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
				s.Equal("hello\n", result.Stdout)
				s.Equal(0, result.ExitCode)
			},
		},
		{
			name: "with env set",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerExecCreate(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						config container.ExecOptions,
					) (common.IDResponse, error) {
						s.Len(config.Env, 1)
						s.Contains(config.Env[0], "MY_VAR=value")
						return common.IDResponse{ID: "exec-env-id"}, nil
					})
				m.EXPECT().
					ContainerExecAttach(gomock.Any(), "exec-env-id", gomock.Any()).
					Return(newHijackedResponse(""), nil)
				m.EXPECT().
					ContainerExecInspect(gomock.Any(), "exec-env-id").
					Return(container.ExecInspect{ExitCode: 0}, nil)
				return m
			},
			containerID: "test-id",
			params: dockerprov.ExecParams{
				Command: []string{"env"},
				Env:     map[string]string{"MY_VAR": "value"},
			},
			validateFunc: func(
				result *dockerprov.ExecResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
			},
		},
		{
			name: "with working dir set",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerExecCreate(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						config container.ExecOptions,
					) (common.IDResponse, error) {
						s.Equal("/app", config.WorkingDir)
						return common.IDResponse{ID: "exec-wd-id"}, nil
					})
				m.EXPECT().
					ContainerExecAttach(gomock.Any(), "exec-wd-id", gomock.Any()).
					Return(newHijackedResponse(""), nil)
				m.EXPECT().
					ContainerExecInspect(gomock.Any(), "exec-wd-id").
					Return(container.ExecInspect{ExitCode: 0}, nil)
				return m
			},
			containerID: "test-id",
			params: dockerprov.ExecParams{
				Command:    []string{"pwd"},
				WorkingDir: "/app",
			},
			validateFunc: func(
				result *dockerprov.ExecResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
			},
		},
		{
			name: "returns error when exec create fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerExecCreate(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(common.IDResponse{}, fmt.Errorf("container not running"))
				return m
			},
			containerID: "stopped-id",
			params: dockerprov.ExecParams{
				Command: []string{"ls"},
			},
			validateFunc: func(
				result *dockerprov.ExecResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "create exec")
			},
		},
		{
			name: "returns error when exec attach fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerExecCreate(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(common.IDResponse{ID: "exec-attach-fail"}, nil)
				m.EXPECT().
					ContainerExecAttach(gomock.Any(), "exec-attach-fail", gomock.Any()).
					Return(types.HijackedResponse{}, fmt.Errorf("attach failed"))
				return m
			},
			containerID: "test-id",
			params: dockerprov.ExecParams{
				Command: []string{"ls"},
			},
			validateFunc: func(
				result *dockerprov.ExecResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "attach exec")
			},
		},
		{
			name: "returns error when io.Copy read fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerExecCreate(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(common.IDResponse{ID: "exec-read-fail"}, nil)
				m.EXPECT().
					ContainerExecAttach(gomock.Any(), "exec-read-fail", gomock.Any()).
					Return(newErrorHijackedResponse(), nil)
				return m
			},
			containerID: "container-1",
			params: dockerprov.ExecParams{
				Command: []string{"ls"},
			},
			validateFunc: func(
				result *dockerprov.ExecResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "read exec output")
			},
		},
		{
			name: "returns error when exec inspect fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ContainerExecCreate(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(common.IDResponse{ID: "exec-inspect-fail"}, nil)
				m.EXPECT().
					ContainerExecAttach(gomock.Any(), "exec-inspect-fail", gomock.Any()).
					Return(newHijackedResponse(""), nil)
				m.EXPECT().
					ContainerExecInspect(gomock.Any(), "exec-inspect-fail").
					Return(container.ExecInspect{}, fmt.Errorf("inspect failed"))
				return m
			},
			containerID: "test-id",
			params: dockerprov.ExecParams{
				Command: []string{"ls"},
			},
			validateFunc: func(
				result *dockerprov.ExecResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "inspect exec")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			result, err := d.Exec(s.ctx, tt.containerID, tt.params)
			tt.validateFunc(result, err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestPull() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		imageName    string
		validateFunc func(result *dockerprov.PullResult, err error)
	}{
		{
			name: "new image pull reports changed true",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				var calls int
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						_ ...dockerclient.ImageInspectOption,
					) (image.InspectResponse, error) {
						calls++
						if calls == 1 {
							return image.InspectResponse{}, fmt.Errorf("not found")
						}
						return image.InspectResponse{
							ID:       "sha256:test123",
							RepoTags: []string{"nginx:latest"},
							Size:     2048,
						}, nil
					}).
					Times(2)
				m.EXPECT().
					ImagePull(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(io.NopCloser(strings.NewReader("{}")), nil)
				return m
			},
			imageName: "nginx:latest",
			validateFunc: func(
				result *dockerprov.PullResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
				s.Equal("sha256:test123", result.ImageID)
				s.Equal("latest", result.Tag)
				s.Equal(int64(2048), result.Size)
				s.True(result.Changed)
			},
		},
		{
			name: "existing image pull reports changed false",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{
						ID:       "sha256:test123",
						RepoTags: []string{"nginx:latest"},
						Size:     2048,
					}, nil).
					Times(2)
				m.EXPECT().
					ImagePull(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(io.NopCloser(strings.NewReader("{}")), nil)
				return m
			},
			imageName: "nginx:latest",
			validateFunc: func(
				result *dockerprov.PullResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
				s.Equal("sha256:test123", result.ImageID)
				s.False(result.Changed)
			},
		},
		{
			name: "successful pull with status digest in last event",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				events := `{"status":"Pulling from library/nginx"}
{"status":"Status","id":"sha256:digest-from-event"}`
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{}, fmt.Errorf("not found"))
				m.EXPECT().
					ImagePull(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(io.NopCloser(strings.NewReader(events)), nil)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{
						ID:       "sha256:inspect-id",
						RepoTags: []string{"nginx:1.25"},
						Size:     4096,
					}, nil)
				return m
			},
			imageName: "nginx:1.25",
			validateFunc: func(
				result *dockerprov.PullResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
				s.Equal("1.25", result.Tag)
				s.Equal(int64(4096), result.Size)
			},
		},
		{
			name: "returns error when pull fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{}, fmt.Errorf("not found"))
				m.EXPECT().
					ImagePull(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("registry unavailable"))
				return m
			},
			imageName: "nonexistent:latest",
			validateFunc: func(
				result *dockerprov.PullResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "pull image")
			},
		},
		{
			name: "returns error on decode failure",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{}, fmt.Errorf("not found"))
				m.EXPECT().
					ImagePull(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(io.NopCloser(strings.NewReader("{}\n{invalid json")), nil)
				return m
			},
			imageName: "nginx:latest",
			validateFunc: func(
				result *dockerprov.PullResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "decode pull response")
			},
		},
		{
			name: "image with no repo tags defaults to latest",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{}, fmt.Errorf("not found"))
				m.EXPECT().
					ImagePull(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(io.NopCloser(strings.NewReader("{}")), nil)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{
						ID:       "sha256:no-tags",
						RepoTags: []string{},
						Size:     512,
					}, nil)
				return m
			},
			imageName: "custom-image",
			validateFunc: func(
				result *dockerprov.PullResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
				s.Equal("latest", result.Tag)
			},
		},
		{
			name: "returns error when image inspect fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{}, fmt.Errorf("not found"))
				m.EXPECT().
					ImagePull(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(io.NopCloser(strings.NewReader("{}")), nil)
				m.EXPECT().
					ImageInspect(gomock.Any(), gomock.Any()).
					Return(image.InspectResponse{}, fmt.Errorf("image not found"))
				return m
			},
			imageName: "nginx:latest",
			validateFunc: func(
				result *dockerprov.PullResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "inspect pulled image")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			result, err := d.Pull(s.ctx, tt.imageName)
			tt.validateFunc(result, err)
		})
	}
}

func (s *DockerDriverPublicTestSuite) TestImageRemove() {
	tests := []struct {
		name         string
		setupMock    func(ctrl *gomock.Controller) *dockermocks.MockAPIClient
		imageName    string
		force        bool
		validateFunc func(result *dockerprov.ActionResult, err error)
	}{
		{
			name: "successful image remove",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageRemove(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]image.DeleteResponse{
						{Deleted: "sha256:abc123"},
					}, nil)
				return m
			},
			imageName: "nginx:latest",
			force:     false,
			validateFunc: func(
				result *dockerprov.ActionResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
				s.True(result.Changed)
				s.Contains(result.Message, "Image removed")
			},
		},
		{
			name: "successful image remove with force",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageRemove(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						options image.RemoveOptions,
					) ([]image.DeleteResponse, error) {
						s.True(options.Force)
						return []image.DeleteResponse{
							{Deleted: "sha256:abc123"},
						}, nil
					})
				return m
			},
			imageName: "nginx:latest",
			force:     true,
			validateFunc: func(
				result *dockerprov.ActionResult,
				err error,
			) {
				s.NoError(err)
				s.NotNil(result)
			},
		},
		{
			name: "returns error when remove fails",
			setupMock: func(
				ctrl *gomock.Controller,
			) *dockermocks.MockAPIClient {
				m := dockermocks.NewMockAPIClient(ctrl)
				m.EXPECT().
					ImageRemove(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("image in use"))
				return m
			},
			imageName: "nginx:latest",
			force:     false,
			validateFunc: func(
				result *dockerprov.ActionResult,
				err error,
			) {
				s.Error(err)
				s.Nil(result)
				s.Contains(err.Error(), "remove image")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			mockClient := tt.setupMock(ctrl)
			d := dockerprov.NewWithClient(mockClient)
			result, err := d.ImageRemove(s.ctx, tt.imageName, tt.force)
			tt.validateFunc(result, err)
		})
	}
}

func TestDockerDriverPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DockerDriverPublicTestSuite))
}
