package rabbitmq

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

type RabbitmqSubscriber struct {
}

func (r *RabbitmqSubscriber) InitSubscriber(stopCh <-chan struct{}) error {
	return nil
}

func (r *RabbitmqSubscriber) SubscribeTopic(gvr schema.GroupVersionResource, codec runtime.Codec, newFunc func() runtime.Object) error {
	return nil
}

func (r *RabbitmqSubscriber) EventReceiving(gvr schema.GroupVersionResource, enqueueFunc func(event *watch.Event), clearfunc func()) error {
	return nil
}

func (r *RabbitmqSubscriber) StopSubscribing(gvr schema.GroupVersionResource) error {
	return nil
}

func (r *RabbitmqSubscriber) StopSubscriber() error {
	return nil
}
