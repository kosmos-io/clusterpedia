package rabbitmq

import (
	"github.com/clusterpedia-io/clusterpedia/pkg/watcher/middleware"
	"github.com/clusterpedia-io/clusterpedia/pkg/watcher/options"
)

const (
	PushlisherName  = "rabbitmq"
	SubscribeerName = "rabbitmq"
)

func NewPulisher(mo *options.MiddlerwareOptions) (middleware.Publisher, error) {
	return &RabbitmqPublisher{}, nil
}

func NewSubscriber(mo *options.MiddlerwareOptions) (middleware.Subscriber, error) {
	return &RabbitmqSubscriber{}, nil
}
