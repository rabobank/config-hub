package sources

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources/credhub_source"
	"github.com/rabobank/config-hub/sources/git_source"
	"github.com/rabobank/config-hub/sources/spi"
)

var (
	l, _            = log.GetWithOptions("SRC", log.Standard().WithFailingCriticals().WithStartingLevel(cfg.LogLevel))
	propertySources []spi.Source
)

func Setup() error {
	var e error
	propertySources = make([]spi.Source, len(cfg.Sources))
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
	sources := findProperties(app, profiles, label)
	for i, properties := range sources {
		flattenedProperties := make(map[string]interface{})
		if e := flattenProperties("", properties.Properties, &flattenedProperties); e != nil {
			l.Errorf("Failed to flatten properties source %s: %v", properties.Source, e)
		} else {
			sources[i].Properties = flattenedProperties
		}
	}
	return sources
}

type dListItem struct {
	n *dListItem
	m *map[string]any
}

type profileMatch struct {
	name    string
	matcher *regexp.Regexp
}

func pushHead(head *dListItem, m *map[string]any) *dListItem {
	if head == nil {
		head = &dListItem{m: m}
	} else {
		head = &dListItem{n: head, m: m}
	}
	return head
}

func insert(point *dListItem, m *map[string]any) *dListItem {
	if point.m == nil {
		point.m = m
		return point
	} else {
		item := &dListItem{n: point.n, m: m}
		point.n = item
		return item
	}
}

func FindPropertiesMap(app string, profiles []string, label string) map[string]any {
	sources := findProperties(app, profiles, label)

	// we now need to merge all source properties from least relevant to most relevant
	profileIndex := make(map[string]*dListItem)
	defaultProfile := false
	var matchingProfiles []*profileMatch
	var head *dListItem
	for _, profile := range profiles {
		if profile == "default" {
			defaultProfile = true
		} else {
			matchingProfiles = append(matchingProfiles, &profileMatch{profile, regexp.MustCompile("^.*-" + profile + ".*.*")})
		}
		head = pushHead(head, nil)
		profileIndex[profile] = head
	}
	if !defaultProfile {
		head = pushHead(head, nil)
		profileIndex[""] = head
	}

	for _, source := range sources {
		matchedProfile := false
		for _, matchingProfile := range matchingProfiles {
			if matchingProfile.matcher.MatchString(source.Source) {
				matchedProfile = true
				profileIndex[matchingProfile.name] = insert(profileIndex[matchingProfile.name], &source.Properties)
				break
			}
		}
		if !matchedProfile {
			profileIndex[""] = insert(profileIndex[""], &source.Properties)
		}
	}

	properties := make(map[string]any)
	for head != nil {
		if head.m != nil {
			// merge only if properties are available for the profile
			properties = mergeMap(properties, *head.m)
		}
		head = head.n
	}

	return properties
}

func mergeMap(baseMap map[string]any, mergingMap map[string]any) map[string]any {
	for k, v := range mergingMap {
		if existingSecret, found := baseMap[k]; found {
			if newSecret, isMap := v.(map[string]any); isMap {
				if existingSecretMap, isMap := existingSecret.(map[string]any); isMap {
					baseMap[k] = mergeMap(existingSecretMap, newSecret)
					continue
				}
			}
		}
		baseMap[k] = v
	}
	return baseMap
}

func findProperties(app string, profiles []string, label string) []*domain.PropertySource {
	var sources []*domain.PropertySource
	apps := strings.Split(app, ",")

	// clean them up stripping the spaces
	for i := range apps {
		apps[i] = strings.TrimSpace(apps[i])
	}

	for _, source := range propertySources {
		if foundProperties, e := source.FindProperties(apps, profiles, label); e != nil {
			l.Errorf("Error when calling source %v: %v", reflect.TypeOf(source).Name(), e)
		} else if foundProperties != nil {
			sources = append(sources, foundProperties...)
		}
	}

	return sources
}
