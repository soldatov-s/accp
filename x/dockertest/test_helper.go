package dockertest

import (
	"fmt"
	"os"

	"github.com/ory/dockertest/v3"
	"github.com/pkg/errors"
)

// nolint : gochecknoglobals
var resources = []*dockertest.Resource{}

// nolint : gochecknoglobals
var pool *dockertest.Pool

// KillAllDockers deletes all test dockers.
func KillAllDockers() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(err)
	}

	for _, r := range resources {
		if err := pool.Purge(r); err != nil {
			panic(err)
		}
	}
}

func startRedis() (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("redislabs/rejson", "1.0.6", []string{"ALLOW_EMPTY_PASSWORD=yes"})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunRedis runs a Redis database and returns the URL to it.
func RunRedis() (string, error) {
	// WASD: adds for gitlab pipeline
	if gitlab := os.Getenv("GITLAB"); gitlab != "" {
		return RunRedisGitlabPipeline()
	}

	resource, err := startRedis()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("redis://127.0.0.1:%s", resource.GetPort("6379/tcp")), nil
}

func RunRedisGitlabPipeline() (string, error) {
	resource, err := startRedis()
	if err != nil {
		return "", err
	}

	return "redis://" + resource.Container.NetworkSettings.IPAddress + ":6379", nil
}

func startRabbitMQ() (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("rabbitmq", "3.8.5-management-alpine", []string{""})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunRabbitMQ runs a RabbitMQ and returns the URL to it.
func RunRabbitMQ() (string, error) {
	// WASD: adds for gitlab pipeline
	if gitlab := os.Getenv("GITLAB"); gitlab != "" {
		return RunRabbitMQGitlabPipeline()
	}

	resource, err := startRedis()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("amqp://guest:guest@127.0.0.1:%s", resource.GetPort("5672/tcp")), nil
}

func RunRabbitMQGitlabPipeline() (string, error) {
	resource, err := startRabbitMQ()
	if err != nil {
		return "", err
	}

	return "amqp://guest:guest@" + resource.Container.NetworkSettings.IPAddress + ":5672", nil
}
