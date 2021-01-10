package publisher

type Publisher interface {
	SendMessage(message interface{}, routingKey string) error
}
