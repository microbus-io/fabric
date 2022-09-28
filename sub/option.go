package sub

// Option is used to construct a subscription in Connector.Subscribe
type Option func(sub *Subscription) error
