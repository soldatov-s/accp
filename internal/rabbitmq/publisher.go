package rabbitmq

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	context "github.com/soldatov-s/accp/internal/ctx"
	"github.com/streadway/amqp"
)

type empty struct{}

type PublisherConfig struct {
	DSN           string
	BackoffPolicy []time.Duration
	ExchangeName  string
}

func (pc *PublisherConfig) Merge(target *PublisherConfig) *PublisherConfig {
	result := &PublisherConfig{
		DSN:           pc.DSN,
		BackoffPolicy: pc.BackoffPolicy,
		ExchangeName:  pc.ExchangeName,
	}

	if target.DSN != "" {
		result.DSN = target.DSN
	}

	if len(target.BackoffPolicy) > 0 {
		result.BackoffPolicy = target.BackoffPolicy
	}

	if target.ExchangeName != "" {
		result.ExchangeName = target.ExchangeName
	}

	return result
}

type Publish struct {
	ctx               *context.Context
	log               zerolog.Logger
	cfg               *PublisherConfig
	Conn              *amqp.Connection
	Channel           *amqp.Channel
	weAreShuttingDown bool
}

func NewPublisher(ctx *context.Context, cfg *PublisherConfig) (*Publish, error) {
	p := &Publish{ctx: ctx, cfg: cfg}
	p.log = ctx.GetPackageLogger(empty{})

	if err := p.connectPublisher(); err != nil {
		return nil, err
	}

	go p.publishStatus()

	p.log.Info().Msg("Publisher connection established")

	return p, nil
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

	err = p.Channel.ExchangeDeclare(p.cfg.ExchangeName, "direct", true,
		false, false,
		false, nil)
	if err != nil {
		return err
	}

	return nil
}

func (p *Publish) publishStatus() {
	for {
		reason, ok := <-p.Channel.NotifyClose(make(chan *amqp.Error))
		if !ok {
			if p.weAreShuttingDown {
				break
			}

			p.log.Error().Msgf("RabbitMQ channel unexpected closed %s", reason)
			err := p.connectPublisher()
			if err != nil {
				p.log.Error().Msgf("Can't reconnect to rabbit %s", err)
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

	p.log.Info().Msg("Closing queue publisher connection...")

	err := p.Channel.Close()
	if err != nil {
		p.log.Error().Err(err).Msg("Failed to close queue channel")
	}
	err = p.Conn.Close()
	if err != nil {
		p.log.Error().Err(err).Msg("Failed to close queue connection")
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

	p.log.Debug().Msgf("send message: %s", string(body))

	err = p.Channel.Publish(p.cfg.ExchangeName, routingKey, false,
		false, amqp.Publishing{ContentType: "text/plain", Body: body})
	if err != nil {
		for _, i := range p.cfg.BackoffPolicy {
			connErr := p.connectPublisher()
			if connErr != nil {
				p.log.Error().Msgf("Error: %s. Trying to reconnect to rabbitMQ", connErr)
				time.Sleep(i * time.Second)
				continue
			}
			break
		}

		pubErr := p.Channel.Publish(p.cfg.ExchangeName, routingKey, false,
			false, amqp.Publishing{ContentType: "text/plain", Body: body})
		if pubErr != nil {
			p.log.Error().Msgf("Failed to publish a message %s", pubErr)
			return pubErr
		}
	}

	return nil
}

func (p *Publish) Ping() (err error) {
	client, err := amqp.Dial(p.cfg.DSN)
	if err != nil {
		p.log.Error().Msgf("Can't rabbit to rabbitMQ %s, error %s", p.cfg.DSN, err)
		return
	}

	_ = client.Close()
	return
}
