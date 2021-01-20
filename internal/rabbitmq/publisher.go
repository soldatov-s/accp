package rabbitmq

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/accp/internal/logger"
	"github.com/soldatov-s/accp/internal/metrics"
	"github.com/soldatov-s/accp/internal/utils"
	"github.com/streadway/amqp"
)

type empty struct{}

type Publish struct {
	ctx               context.Context
	log               zerolog.Logger
	cfg               *Config
	Conn              *amqp.Connection
	Channel           *amqp.Channel
	weAreShuttingDown bool

	// Metrics
	metrics.Service
}

func NewPublisher(ctx context.Context, cfg *Config) (*Publish, error) {
	cfg.SetDefault()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Publish{
		ctx: ctx,
		cfg: cfg,
		log: logger.GetPackageLogger(ctx, empty{}),
	}

	return p, nil
}

func (p *Publish) Start() error {
	if err := p.connectPublisher(); err != nil {
		return err
	}

	go p.publishStatus()

	p.log.Info().Msg("publisher connection established")

	return nil
}

func (p *Publish) connectPublisher() error {
	var err error

	p.Conn, err = amqp.Dial(p.cfg.DSN)
	if err != nil {
		return err
	}

	p.Channel, err = p.Conn.Channel()
	if err != nil {
		return err
	}

	return p.Channel.ExchangeDeclare(p.cfg.ExchangeName, "direct", true,
		false, false,
		false, nil)
}

func (p *Publish) publishStatus() {
	for {
		reason, ok := <-p.Channel.NotifyClose(make(chan *amqp.Error))
		if !ok {
			if p.weAreShuttingDown {
				break
			}

			p.log.Error().Msgf("rabbitmq channel unexpected closed %s", reason)
			err := p.connectPublisher()
			if err != nil {
				p.log.Error().Msgf("can't reconnect to rabbit %s", err)
				time.Sleep(10 * time.Second)
				continue
			}
		}
	}
}

func (p *Publish) Shutdown() error {
	if p == nil || p.Conn == nil {
		return nil
	}

	p.weAreShuttingDown = true

	p.log.Info().Msg("closing queue publisher connection...")

	err := p.Channel.Close()
	if err != nil {
		p.log.Error().Err(err).Msg("failed to close queue channel")
	}
	err = p.Conn.Close()
	if err != nil {
		p.log.Error().Err(err).Msg("failed to close queue connection")
	}

	p.Channel = nil
	p.Conn = nil

	return nil
}

// SendMessage publish message to exchange
func (p *Publish) SendMessage(message interface{}, routingKey string) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	p.log.Debug().Msgf("exchangeName %s, routingKey %s, send message: %s", p.cfg.ExchangeName, routingKey, string(body))

	err = p.Channel.Publish(p.cfg.ExchangeName, routingKey, false,
		false, amqp.Publishing{ContentType: "text/plain", Body: body})
	if err != nil {
		for _, i := range p.cfg.BackoffPolicy {
			connErr := p.connectPublisher()
			if connErr != nil {
				p.log.Error().Msgf("error: %s, trying to reconnect to rabbitMQ", connErr)
				time.Sleep(i * time.Second)
				continue
			}
			break
		}

		pubErr := p.Channel.Publish(p.cfg.ExchangeName, routingKey, false,
			false, amqp.Publishing{ContentType: "text/plain", Body: body})
		if pubErr != nil {
			p.log.Error().Msgf("failed to publish a message %s", pubErr)
			return pubErr
		}
	}

	return nil
}

func (p *Publish) Ping() (err error) {
	client, err := amqp.Dial(p.cfg.DSN)
	if err != nil {
		p.log.Error().Msgf("can't rabbit to rabbitMQ %s, error %s", p.cfg.DSN, err)
		return
	}

	_ = client.Close()
	return
}

// GetMetrics return map of the metrics from cache connection
func (p *Publish) GetMetrics() metrics.MapMetricsOptions {
	_ = p.Service.GetMetrics()
	p.Metrics[ProviderName+"_status"] = &metrics.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: ProviderName + "_status",
				Help: ProviderName + " status link to " + utils.RedactedDSN(p.cfg.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			err := p.Ping()
			if err == nil {
				(m.(prometheus.Gauge)).Set(1)
			}
		},
	}
	return p.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (p *Publish) GetReadyHandlers() metrics.MapCheckFunc {
	_ = p.Service.GetReadyHandlers()
	p.ReadyHandlers[strings.ToUpper(ProviderName+"_notfailed")] = func() (bool, string) {
		if err := p.Ping(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return p.ReadyHandlers
}
