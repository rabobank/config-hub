package sources

import (
	"reflect"

	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources/credhub_source"
	"github.com/rabobank/config-hub/sources/git_source"
)

const (
	DefaultLabel = "master"
)

var (
	l, _            = log.GetWithOptions("SRC", log.Standard().WithFailingCriticals().WithStartingLevel(cfg.LogLevel))
	propertySources []Source
)

type Source interface {
	FindProperties(app string, profiles []string, label string) ([]*domain.PropertySource, error)
}

func Setup() error {
	var e error
	propertySources = make([]Source, len(cfg.Sources))
	for i, sourceCfg := range cfg.Sources {
		switch sourceCfg.Type() {
		case "git":
			if propertySources[i], e = git_source.Source(sourceCfg, i); e != nil {
				l.Critical(e)
			}
		case "credhub":
			if propertySources[i], e = credhub_source.Source(sourceCfg); e != nil {
				l.Critical(e)
			}
		default:
			l.Criticalf("Unsupported source type %s\n", sourceCfg.Type())
		}
	}

	return nil
}

func FindProperties(app string, profiles []string, label string) []*domain.PropertySource {
	var sources []*domain.PropertySource
	for _, source := range propertySources {
		if foundProperties, e := source.FindProperties(app, profiles, label); e != nil {
			l.Errorf("Error when calling source %v: %v", reflect.TypeOf(source).Name(), e)
		} else if foundProperties != nil {
			sources = append(sources, foundProperties...)
		}
	}

	return sources
}
