package spi

import (
	"fmt"

	"github.com/rabobank/config-hub/domain"
)

type Source interface {
	fmt.Stringer
	FindProperties(apps []string, profiles []string, label string) ([]*domain.PropertySource, error)
	Name() string
	DashboardReport() *string
}
