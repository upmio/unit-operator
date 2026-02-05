package backends

import (
	"errors"

	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/backends/content"
)

// The StoreClient interface is implemented by objects that can retrieve
// key/value pairs from a backend store.
type StoreClient interface {
	GetValues() (map[string]string, error)
}

// New is used to create a storage client based on our configuration.
func New(config Config) (StoreClient, error) {
	switch config.Backend {
	case "content":
		return content.NewContentClient(config.Contents), nil
	}

	return nil, errors.New("invalid backend")
}
