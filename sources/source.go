package sources

import (
	"config-hub/domain"
)

const (
	DefaultLabel = "master"
)

type Source interface {
	FindProperties(app string, profiles []string, label string) ([]*domain.PropertySource, error)
}
