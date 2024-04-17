package spi

import (
	"github.com/rabobank/config-hub/domain"
)

type Source interface {
	FindProperties(apps []string, profiles []string, label string) ([]*domain.PropertySource, error)
	Name() string
	DashboardReport() *string
}
