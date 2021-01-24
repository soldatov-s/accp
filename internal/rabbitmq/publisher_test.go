package rabbitmq

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/meta"
	"github.com/soldatov-s/accp/x/dockertest"
	rabbitMQConsumer "github.com/soldatov-s/accp/x/test_helpers/rabbitmq"
	"github.com/soldatov-s/accp/x/test_helpers/resilience"
	"github.com/stretchr/testify/require"
)

const (
	testDSN          = "amqp://guest:guest@rabbitmq:5672"
	testExchangeName = "accp.test.events"
	testRouteKey     = "ACCP_TEST"
	testQueue        = "test.queue"
	testMessage      = "test message"
	testConsumer     = "test_consumer"
)

func initApp(ctx context.Context) context.Context {
	return meta.SetAppInfo(ctx, "accp", "", "", "", "test")
}

func initLogger(ctx context.Context) context.Context {
	// Registrate logger
	logCfg := &logger.Config{
		Level:           logger.LoggerLevelDebug,
		NoColoredOutput: true,
		WithTrace:       false,
	}
	ctx = logger.RegistrateAndInitilize(ctx, logCfg)

	return ctx
}

func initConfig() *Config {
	return &Config{
		ExchangeName: testExchangeName,
	}
}

func initPublish(t *testing.T, dsn string) *Publish {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()

	cfg.DSN = dsn

	t.Logf("connecting to rabbitmq: %s", dsn)

	publish, err := NewPublisher(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, publish)

	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		publish.Start,
	)

	require.Nil(t, err)

	t.Logf("connected to rabbitmq: %s", dsn)

	return publish
}

func TestNewPublisher(t *testing.T) {
	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()
	cfg.DSN = testDSN

	publish, err := NewPublisher(ctx, cfg)
	require.Nil(t, err)
	require.NotNil(t, publish)
}

func TestStart(t *testing.T) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	_ = initPublish(t, dsn)
}

func TestConnectPublisher(t *testing.T) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()
	cfg.DSN = dsn

	publish, err := NewPublisher(ctx, cfg)
	require.Nil(t, err)

	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			err = publish.connectPublisher()
			return err
		},
	)

	require.Nil(t, err)

	err = publish.Shutdown()
	require.Nil(t, err)
}

func TestPublishStatus(t *testing.T) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	ctx := context.Background()
	ctx = initApp(ctx)
	ctx = initLogger(ctx)
	cfg := initConfig()
	cfg.DSN = dsn

	publish, err := NewPublisher(ctx, cfg)
	require.Nil(t, err)

	err = resilience.Retry(
		t,
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			err = publish.connectPublisher()
			return err
		},
	)
	require.Nil(t, err)

	go publish.publishStatus()

	err = publish.Channel.Close()
	require.Nil(t, err)

	time.Sleep(100 * time.Millisecond)
	err = publish.Ping()
	require.Nil(t, err)

	err = publish.Shutdown()
	require.Nil(t, err)
}

func TestSendMessage(t *testing.T) {
	messages := 10

	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initPublish(t, dsn)

	consum, err := rabbitMQConsumer.CreateConsumer(dsn)
	require.Nil(t, err)

	shutdownConsumer := make(chan bool)

	successCount := 0
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		msgs, err1 := consum.StartConsume(testExchangeName, testQueue, testRouteKey, testConsumer)
		require.Nil(t, err1)
		wg.Done()

		t.Log("start consumer")
		for {
			select {
			case <-shutdownConsumer:
				return
			default:
				for d := range msgs {
					successCount++
					t.Logf("received a message: %s", d.Body)
					_ = d.Ack(true)
				}
			}
		}
	}()

	wg.Wait()

	for m := 1; m <= messages; m++ {
		t.Logf("send message %d", m)
		err1 := c.SendMessage(testMessage+" "+strconv.Itoa(m), testRouteKey)
		require.Nil(t, err1)
	}
	close(shutdownConsumer)

	// timeout for consumer
	time.Sleep(10 * time.Second)

	err = c.Shutdown()
	require.Nil(t, err)
	err = consum.Shutdown()
	require.Nil(t, err)

	require.Equal(t, messages, successCount)
}

func TestPing(t *testing.T) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initPublish(t, dsn)
	err = c.Ping()
	require.Nil(t, err)

	err = c.Shutdown()
	require.Nil(t, err)
}

func TestGetMetrics(t *testing.T) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initPublish(t, dsn)
	m := c.GetMetrics()
	require.NotNil(t, m)

	_, ok := m[ProviderName+"_status"]
	require.True(t, ok)

	err = c.Shutdown()
	require.Nil(t, err)
}

func TestGetReadyHandlers(t *testing.T) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initPublish(t, dsn)
	h := c.GetReadyHandlers()
	require.NotNil(t, h)

	_, ok := h[strings.ToUpper(ProviderName+"_notfailed")]
	require.True(t, ok)

	err = c.Shutdown()
	require.Nil(t, err)
}

func TestShutdown(t *testing.T) {
	dsn, err := dockertest.RunRabbitMQ()
	require.Nil(t, err)
	defer dockertest.KillAllDockers()

	c := initPublish(t, dsn)
	h := c.GetReadyHandlers()
	require.NotNil(t, h)

	err = c.Shutdown()
	require.Nil(t, err)
}
