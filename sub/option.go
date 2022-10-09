package sub

import (
	"regexp"

	"github.com/microbus-io/fabric/errors"
)

// Option is used to construct a subscription in Connector.Subscribe
type Option func(sub *Subscription) error

// Queue names the queue of the subscription.
// Requests will be load-balanced among all consumers with the same queue name
func Queue(queue string) Option {
	return func(sub *Subscription) error {
		match, err := regexp.MatchString(`^[a-zA-Z0-9\.]+$`, queue)
		if err != nil {
			return errors.Trace(err)
		}
		if !match {
			return errors.Newf("invalid queue name: %s", queue)
		}
		sub.Queue = queue
		return nil
	}
}

// NoQueue sets no queue for this subscription.
// Requests will be not be load-balanced, all instances of this microservice will receive the request
func NoQueue() Option {
	return func(sub *Subscription) error {
		sub.Queue = ""
		return nil
	}
}

// Pervasive is synonymous with NoQueue.
// Requests will be not be load-balanced, all instances of this microservice will receive the request
func Pervasive() Option {
	return NoQueue()
}

// DefaultQueue names the queue of this subscription to the host name of the service.
// Requests will be load-balanced among all instances of this microservice.
func DefaultQueue() Option {
	return func(sub *Subscription) error {
		sub.Queue = sub.Host
		return nil
	}
}

// LoadBalanced is synonymous with DefaultQueue.
// Requests will be load-balanced among all instances of this microservice
func LoadBalanced() Option {
	return DefaultQueue()
}
