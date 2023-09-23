package queue

type Queuer interface {
	Send(topic string, entry []byte)
	AddHandler(handler func(string, []byte) error)
}
