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

	appFs afero.Fs
	ctrl  *gomock.Controller
	ctx   context.Context
}

func (suite *UbuntuPublicTestSuite) SetupTest() {
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	suite.appFs = afero.NewMemMapFs()
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
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
	dockerFilePath := "./internal/manager/system/ubuntu/Dockerfile"
	// TODO(retr0h): change
	buildContext := "/Users/retr0h/git/osapi"

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       buildContext,
			Dockerfile:    dockerFilePath,
			PrintBuildLog: true,
			KeepImage:     true,
		},
		Env:          map[string]string{"GOCOVERDIR": "/coverage"},
		Mounts:       testcontainers.Mounts(testcontainers.BindMount("/Users/retr0h/git/osapi/coverage", "/coverage")),
		ExposedPorts: []string{"8080/tcp"},
		// WaitingFor:   wait.ForLog("http server started on"),
		WaitingFor: wait.ForListeningPort("8080/tcp"),
		Hostname:   "ubuntu-hostname",
	}

	ubuntuContainer, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Println(err)
	}

	hostPort, err := ubuntuContainer.MappedPort(suite.ctx, "8080/tcp")
	if err != nil {
		//
	}

	// Get the Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		//
	}

	containerID := ubuntuContainer.GetContainerID()

	// Defer the graceful shutdown using SIGTERM
	defer func() {
		err = dockerClient.ContainerKill(suite.ctx, containerID, "SIGTERM")
		if err != nil {
			// log.Printf("Failed to send SIGTERM to container: %s", err)
		} else {
			// log.Println("Sent SIGTERM to container.")
		}

		// Wait for a while to allow graceful shutdown
		time.Sleep(10 * time.Second)

		removeOptions := container.RemoveOptions{
			Force:         true, // Set to true if you want to force remove a running container
			RemoveVolumes: true, // Set to true to remove volumes associated with the container
		}

		// Finally, remove the container
		err = dockerClient.ContainerRemove(suite.ctx, containerID, removeOptions)
		if err != nil {
			// log.Fatalf("Failed to remove container: %s", err)
		}

		fmt.Println("Container terminated gracefully and removed.")
	}()

	url := fmt.Sprintf("http://%s:%s/system/status", "0.0.0.0", hostPort.Port())
	fmt.Println(url)
	fmt.Println(url)
	fmt.Println(url)
	cmd := exec.Command("curl", "-X", "GET", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(err)
		fmt.Println(err)
		fmt.Println(err)
	}
	fmt.Printf("Output:\n%s\n", output)
	fmt.Printf("Output:\n%s\n", output)
	fmt.Printf("Output:\n%s\n", output)

	// Send SIGTERM for graceful shutdown
	err = dockerClient.ContainerKill(suite.ctx, containerID, "SIGTERM")
	if err != nil {
		// log.Fatalf("Failed to send SIGTERM to container: %s", err)
		fmt.Println(err)
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuPublicTestSuite))
}
