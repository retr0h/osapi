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

// Package docker provides the Docker container management provider using the Docker Engine API.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/osapi-io/osapi/internal/provider"
)

// Compile-time check: Client must satisfy FactsSetter.
var _ provider.FactsSetter = (*Client)(nil)

// Client implements Driver using the Docker Engine API.
type Client struct {
	provider.FactsAware
	client APIClient
}

// New creates a new Docker provider using default client options.
func New() (*Client, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	return &Client{client: cli}, nil
}

// NewWithClient creates a Docker provider with an injected client (for testing).
func NewWithClient(
	client APIClient,
) *Client {
	return &Client{client: client}
}

// Ping verifies connectivity to the Docker daemon.
func (d *Client) Ping(
	ctx context.Context,
) error {
	_, err := d.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("ping docker daemon: %w", err)
	}

	return nil
}

// Create creates a new container from the given parameters.
func (d *Client) Create(
	ctx context.Context,
	params CreateParams,
) (*Container, error) {
	// Build Docker container configuration
	config := &container.Config{
		Image: params.Image,
	}

	if params.Hostname != "" {
		config.Hostname = params.Hostname
	}

	if len(params.Command) > 0 {
		config.Cmd = params.Command
	}

	if len(params.Env) > 0 {
		envVars := make([]string, 0, len(params.Env))
		for k, v := range params.Env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		config.Env = envVars
	}

	// Build host configuration
	hostConfig := &container.HostConfig{}

	if len(params.DNS) > 0 {
		hostConfig.DNS = params.DNS
	}

	// Convert port mappings
	if len(params.Ports) > 0 {
		portBindings := nat.PortMap{}
		exposedPorts := nat.PortSet{}

		for _, pm := range params.Ports {
			containerPort := nat.Port(fmt.Sprintf("%d/tcp", pm.Container))
			exposedPorts[containerPort] = struct{}{}
			portBindings[containerPort] = []nat.PortBinding{
				{
					HostPort: strconv.Itoa(pm.Host),
				},
			}
		}

		config.ExposedPorts = exposedPorts
		hostConfig.PortBindings = portBindings
	}

	// Convert volume mappings
	if len(params.Volumes) > 0 {
		mounts := make([]mount.Mount, 0, len(params.Volumes))
		for _, vm := range params.Volumes {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: vm.Host,
				Target: vm.Container,
			})
		}
		hostConfig.Mounts = mounts
	}

	// Create the container
	resp, err := d.client.ContainerCreate(
		ctx,
		config,
		hostConfig,
		&network.NetworkingConfig{},
		nil,
		params.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	// Start the container if AutoStart is enabled
	if params.AutoStart {
		if _, err := d.Start(ctx, resp.ID); err != nil {
			return nil, fmt.Errorf("auto-start container: %w", err)
		}
	}

	// Return container summary
	return &Container{
		ID:      resp.ID,
		Name:    params.Name,
		Image:   params.Image,
		State:   "created",
		Created: time.Now(),
		Changed: true,
	}, nil
}

// Start starts a stopped container.
func (d *Client) Start(
	ctx context.Context,
	id string,
) (*ActionResult, error) {
	if err := d.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	return &ActionResult{
		Message: "Container started successfully",
		Changed: true,
	}, nil
}

// Stop stops a running container with an optional timeout.
func (d *Client) Stop(
	ctx context.Context,
	id string,
	timeout *time.Duration,
) (*ActionResult, error) {
	opts := container.StopOptions{}
	if timeout != nil {
		seconds := int(timeout.Seconds())
		opts.Timeout = &seconds
	}

	if err := d.client.ContainerStop(ctx, id, opts); err != nil {
		return nil, fmt.Errorf("stop container: %w", err)
	}

	return &ActionResult{
		Message: "Container stopped successfully",
		Changed: true,
	}, nil
}

// Remove removes a container.
func (d *Client) Remove(
	ctx context.Context,
	id string,
	force bool,
) (*ActionResult, error) {
	opts := container.RemoveOptions{
		Force: force,
	}

	if err := d.client.ContainerRemove(ctx, id, opts); err != nil {
		return nil, fmt.Errorf("remove container: %w", err)
	}

	return &ActionResult{
		Message: "Container removed successfully",
		Changed: true,
	}, nil
}

// List returns a list of containers matching the given parameters.
func (d *Client) List(
	ctx context.Context,
	params ListParams,
) ([]Container, error) {
	opts := container.ListOptions{}

	// Apply state filter
	switch params.State {
	case "running":
		opts.All = false
	case "stopped":
		opts.All = true
		opts.Filters = filters.NewArgs(filters.Arg("status", "exited"))
	case "all":
		opts.All = true
	default:
		opts.All = false
	}

	// Apply limit
	if params.Limit > 0 {
		opts.Limit = params.Limit
	}

	containers, err := d.client.ContainerList(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	// Convert to Container
	result := make([]Container, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			// Remove leading "/" from container name
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		result = append(result, Container{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Created: time.Unix(c.Created, 0),
		})
	}

	return result, nil
}

// Inspect returns detailed information about a container.
func (d *Client) Inspect(
	ctx context.Context,
	id string,
) (*ContainerDetail, error) {
	resp, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	// Build container detail
	name := strings.TrimPrefix(resp.Name, "/")
	state := "unknown"
	if resp.State != nil {
		state = resp.State.Status
	}

	// Parse Created timestamp
	created, err := time.Parse(time.RFC3339Nano, resp.Created)
	if err != nil {
		created = time.Now()
	}

	detail := &ContainerDetail{
		Container: Container{
			ID:      resp.ID,
			Name:    name,
			Image:   resp.Config.Image,
			State:   state,
			Created: created,
		},
	}

	// Add network settings
	if resp.NetworkSettings != nil {
		for _, netConfig := range resp.NetworkSettings.Networks {
			detail.NetworkSettings = &NetworkSettings{
				IPAddress: netConfig.IPAddress,
				Gateway:   netConfig.Gateway,
			}
			break // Use first network
		}
	}

	// Add port mappings
	if resp.HostConfig != nil && resp.HostConfig.PortBindings != nil {
		ports := make([]PortMapping, 0)
		for containerPort, bindings := range resp.HostConfig.PortBindings {
			for _, binding := range bindings {
				hostPort, _ := strconv.Atoi(binding.HostPort)
				cPort, _ := strconv.Atoi(containerPort.Port())
				ports = append(ports, PortMapping{
					Host:      hostPort,
					Container: cPort,
				})
			}
		}
		detail.Ports = ports
	}

	// Add mounts
	if len(resp.Mounts) > 0 {
		mounts := make([]VolumeMapping, 0, len(resp.Mounts))
		for _, m := range resp.Mounts {
			mounts = append(mounts, VolumeMapping{
				Host:      m.Source,
				Container: m.Destination,
			})
		}
		detail.Mounts = mounts
	}

	// Add health status
	if resp.State != nil && resp.State.Health != nil {
		detail.Health = resp.State.Health.Status
	}

	return detail, nil
}

// Exec executes a command in a running container.
func (d *Client) Exec(
	ctx context.Context,
	id string,
	params ExecParams,
) (*ExecResult, error) {
	// Create exec configuration
	execConfig := container.ExecOptions{
		Cmd:          params.Command,
		AttachStdout: true,
		AttachStderr: true,
	}

	if len(params.Env) > 0 {
		envVars := make([]string, 0, len(params.Env))
		for k, v := range params.Env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		execConfig.Env = envVars
	}

	if params.WorkingDir != "" {
		execConfig.WorkingDir = params.WorkingDir
	}

	// Create exec instance
	execResp, err := d.client.ContainerExecCreate(ctx, id, execConfig)
	if err != nil {
		return nil, fmt.Errorf("create exec: %w", err)
	}

	// Attach to exec instance
	attachResp, err := d.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("attach exec: %w", err)
	}
	defer attachResp.Close()

	// Read output
	var stdout, stderr bytes.Buffer
	if _, err := io.Copy(&stdout, attachResp.Reader); err != nil {
		return nil, fmt.Errorf("read exec output: %w", err)
	}

	// Get exit code
	inspectResp, err := d.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("inspect exec: %w", err)
	}

	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: inspectResp.ExitCode,
		Changed:  true,
	}, nil
}

// ImageRemove removes a container image from the local Docker daemon.
func (d *Client) ImageRemove(
	ctx context.Context,
	imageName string,
	force bool,
) (*ActionResult, error) {
	opts := image.RemoveOptions{
		Force: force,
	}

	_, err := d.client.ImageRemove(ctx, imageName, opts)
	if err != nil {
		return nil, fmt.Errorf("remove image: %w", err)
	}

	return &ActionResult{
		Message: "Image removed successfully",
		Changed: true,
	}, nil
}

// Pull pulls a container image from a registry. Compares the image
// digest before and after the pull to determine if anything changed.
func (d *Client) Pull(
	ctx context.Context,
	imageName string,
) (*PullResult, error) {
	// Capture the image ID before pulling (empty if not present).
	var beforeID string
	if before, err := d.client.ImageInspect(ctx, imageName); err == nil {
		beforeID = before.ID
	}

	pullResp, err := d.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("pull image: %w", err)
	}
	defer func() { _ = pullResp.Close() }()

	// Read pull output to ensure completion
	var lastEvent map[string]interface{}
	decoder := json.NewDecoder(pullResp)
	for {
		var event map[string]interface{}
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf("decode pull response: %w", err)
		}
		lastEvent = event
	}

	// Inspect the pulled image to get details
	inspectResp, err := d.client.ImageInspect(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("inspect pulled image: %w", err)
	}

	tag := "latest"
	if len(inspectResp.RepoTags) > 0 {
		parts := strings.Split(inspectResp.RepoTags[0], ":")
		if len(parts) > 1 {
			tag = parts[1]
		}
	}

	// Changed is true only if the image was not present before or
	// the image ID differs (new layer was downloaded).
	changed := beforeID == "" || beforeID != inspectResp.ID

	result := &PullResult{
		ImageID: inspectResp.ID,
		Tag:     tag,
		Size:    inspectResp.Size,
		Changed: changed,
	}

	// Extract digest from last event if available
	if lastEvent != nil {
		if status, ok := lastEvent["status"].(string); ok && status == "Status" {
			if digest, ok := lastEvent["id"].(string); ok {
				result.ImageID = digest
			}
		}
	}

	return result, nil
}
