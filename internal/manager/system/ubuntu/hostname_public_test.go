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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/retr0h/osapi/internal/manager/system"
	"github.com/retr0h/osapi/internal/manager/system/ubuntu/mocks"
)

type UbuntuPublicTestSuite struct {
	suite.Suite

	appFs        afero.Fs
	ctrl         *gomock.Controller
	ctx          context.Context
	dockerClient *client.Client
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

	fmt.Println("Container terminated gracefully and removed.")
}

func (suite *UbuntuPublicTestSuite) SetupTest() {
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	suite.appFs = afero.NewMemMapFs()
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.dockerClient, _ = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func (suite *UbuntuPublicTestSuite) TearDownTest() {
	defer suite.ctrl.Finish()
}

func (suite *UbuntuPublicTestSuite) TestGetHostnameOk() {
	mockHostnameProvider := mocks.NewMockHostnameProvider(suite.ctrl)
	mockHostnameProvider.EXPECT().GetHostname().Return("test-hostname", nil)

	sys := system.New(suite.appFs)
	sys.HostnameProvider = mockHostnameProvider

	got, err := sys.GetHostname()

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-hostname", got)
}

func (suite *UbuntuPublicTestSuite) TestGetHostnameReturnsError() {
	mockHostnameProvider := mocks.NewMockHostnameProvider(suite.ctrl)
	mockHostnameProvider.EXPECT().GetHostname().Return("", errors.New("unable to get hostname"))

	sys := system.New(suite.appFs)
	sys.HostnameProvider = mockHostnameProvider

	_, err := sys.GetHostname()

	assert.Error(suite.T(), err)
}

func (suite *UbuntuPublicTestSuite) TestDocker() {
	_, filename, _, _ := runtime.Caller(0)
	buildContext := filepath.Join(filepath.Dir(filename), "../../../..")
	dockerFilePath := "./internal/manager/system/ubuntu/Dockerfile"

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       buildContext,
			Dockerfile:    dockerFilePath,
			PrintBuildLog: false,
			KeepImage:     false,
		},
		Env:          map[string]string{"GOCOVERDIR": "/coverage"},
		Mounts:       testcontainers.Mounts(testcontainers.BindMount(filepath.Join(buildContext, "coverage"), "/coverage")),
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForListeningPort("8080/tcp"),
		Hostname:     "ubuntu-hostname",
	}

	ubuntuContainer, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(suite.T(), err)

	hostPort, err := ubuntuContainer.MappedPort(suite.ctx, "8080/tcp")
	assert.NoError(suite.T(), err)

	defer stopContainer(suite.ctx, suite.dockerClient, ubuntuContainer.GetContainerID())

	url := fmt.Sprintf("http://%s:%s/system/status", "0.0.0.0", hostPort.Port())
	cmd := exec.Command("curl", "-X", "GET", url)
	_, err = cmd.CombinedOutput()
	assert.NoError(suite.T(), err)
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuPublicTestSuite))
}
