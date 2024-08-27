//go:build integration

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

package ubuntu_test

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type UbuntuIntegrationTestSuite struct {
	suite.Suite

	ctx          context.Context
	dockerClient *client.Client
}

type containerCreateOptions struct {
	dockerFilePath string
	hostname       string
	exposedPorts   []string
	printBuildLog  bool
	keepImage      bool
}

func startContainer(
	ctx context.Context,
	opts containerCreateOptions,
) (testcontainers.Container, error) {
	// pass in config type
	_, filename, _, _ := runtime.Caller(0)
	buildContext := filepath.Join(filepath.Dir(filename), "../../../..")

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       buildContext,
			Dockerfile:    opts.dockerFilePath,
			PrintBuildLog: opts.printBuildLog,
			KeepImage:     opts.keepImage,
		},
		Env:          map[string]string{"GOCOVERDIR": "/coverage"},
		Mounts:       testcontainers.Mounts(testcontainers.BindMount(filepath.Join(buildContext, "coverage"), "/coverage")),
		ExposedPorts: opts.exposedPorts,
		WaitingFor:   wait.ForListeningPort("8080/tcp"),
		Hostname:     opts.hostname,
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// graceful shutdown using SIGTERM
func stopContainer(
	ctx context.Context,
	dockerClient *client.Client,
	containerID string,
) {
	_ = dockerClient.ContainerKill(ctx, containerID, "SIGTERM")

	// Wait for a while to allow graceful shutdown
	time.Sleep(10 * time.Second)

	removeOptions := container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}

	_ = dockerClient.ContainerRemove(ctx, containerID, removeOptions)
}

func (suite *UbuntuIntegrationTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.dockerClient, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func (suite *UbuntuIntegrationTestSuite) TearDownTest() {}

func (suite *UbuntuIntegrationTestSuite) TestGetHostnameOk() {
	createOptions := containerCreateOptions{
		dockerFilePath: "internal/manager/system/ubuntu/Dockerfile",
		hostname:       "ubuntu-hostname",
		exposedPorts:   []string{"8080/tcp"},
		printBuildLog:  false,
		keepImage:      false,
	}

	c, err := startContainer(suite.ctx, createOptions)
	require.NoError(suite.T(), err)
	defer stopContainer(suite.ctx, suite.dockerClient, c.GetContainerID())

	hostPort, err := c.MappedPort(suite.ctx, "8080/tcp")
	require.NoError(suite.T(), err)

	url := fmt.Sprintf("http://%s:%s/system/status", "0.0.0.0", hostPort.Port())
	cmd := exec.Command("curl", "-X", "GET", url)
	_, err = cmd.CombinedOutput()
	assert.NoError(suite.T(), err)
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuIntegrationTestSuite))
}
