package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"

	"github.com/docker/docker/api/types"

	"github.com/docker/docker/client"
)

const (
	stopTimeout = time.Second * 30
)

// TODO: dependency injection pattern for logs
// TODO: more verbose logs

type api struct {
	docker *client.Client
}

func newAPI() (*api, error) {
	a := &api{}
	if err := a.Init(); err != nil {
		return nil, errors.Wrap(err, "init API")
	}

	return a, nil
}

// Init creates client to communicate with Docker
func (a *api) Init() error {
	docker, err := client.NewClientWithOpts()
	if err != nil {
		return errors.Wrap(err, "create client")
	}
	a.docker = docker

	return nil
}

func imageName(image string) string {
	s := strings.Split(image, "/")
	if len(s) == 0 {
		return ""
	}
	return s[len(s)-1]
}

func (a api) imagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	log.Printf("Pulling image %s", refStr)
	return a.docker.ImagePull(ctx, refStr, options)
}

func (a api) containerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	log.Printf("Creating container %s", config.Image)
	return a.docker.ContainerCreate(ctx, config, hostConfig, networkingConfig, containerName)
}

func (a api) containerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	log.Printf("Starting container %s", containerID)
	return a.docker.ContainerStart(ctx, containerID, options)
}

// RunContainerBackground creates a container from given image and starts it
func (a api) RunContainerBackground(image string) error {
	log.Printf("Starting image %s", image)
	ctx := context.Background()

	_, err := a.imagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return errors.Wrap(err, "pull image")
	}

	resp, err := a.containerCreate(ctx, &container.Config{
		Image: imageName(image),
	}, nil, nil, "")
	if err != nil {
		return errors.Wrap(err, "create container")
	}

	if err := a.containerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "start container")
	}

	return nil
}

// RunContainerCmd creates a container from given image and and executes the command in it
func (a api) RunContainerCmd(image string, cmd []string) (string, error) {
	ctx := context.Background()

	_, err := a.imagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", errors.Wrap(err, "pull image")
	}

	resp, err := a.containerCreate(ctx, &container.Config{
		Image: imageName(image),
		Cmd:   cmd,
		Tty:   true,
	}, nil, nil, "")
	if err != nil {
		return "", errors.Wrap(err, "create container")
	}

	if err := a.containerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", errors.Wrap(err, "start container")
	}

	statusCh, errCh := a.docker.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", errors.Wrap(err, "wait for container")
		}
	case <-statusCh:
	}

	out, err := a.docker.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", errors.Wrap(err, "read container logs")
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("Error on closing container logs output: %s", err)
		}
	}()

	outRead, err := ioutil.ReadAll(out)
	if err != nil {
		return "", errors.Wrap(err, "read command output")
	}

	return string(outRead), nil
}

// StopContainer stops container
func (a *api) StopContainer(id string) error {
	log.Printf("Stopping container %s", id)

	tmp := stopTimeout
	return a.docker.ContainerStop(context.Background(), id, &tmp)
}

// ListContainers lists containers
func (a *api) ListContainers() ([]types.Container, error) {
	containers, err := a.docker.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list containers")
	}

	return containers, nil
}
