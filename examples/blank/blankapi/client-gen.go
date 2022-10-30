// Code generated by Microbus. DO NOT EDIT.

package blankapi

// Client provides type-safe access to the endpoints of the "blank.example" microservice.
// This simple version is for unicast calls.
type Client struct {
	svc  Service
	host string
}

// NewClient creates a new unicast client to the "blank.example" microservice.
func NewClient(caller Service) *Client {
	return &Client{
		svc:  caller,
		host: "blank.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *Client) ForHost(host string) *Client {
	_c.host = host
	return _c
}

// MulticastClient provides type-safe access to the endpoints of the "blank.example" microservice.
// This advanced version is for multicast calls.
type MulticastClient struct {
	svc  Service
	host string
}

// NewMulticastClient creates a new multicast client to the "blank.example" microservice.
func NewMulticastClient(caller Service) *MulticastClient {
	return &MulticastClient{
		svc:  caller,
		host: "blank.example",
	}
}

// ForHost replaces the default host name of this client.
func (_c *MulticastClient) ForHost(host string) *MulticastClient {
	_c.host = host
	return _c
}
