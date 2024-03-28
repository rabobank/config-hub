package credhub_source

import (
	"fmt"
	"strings"

	err "github.com/gomatbase/go-error"
	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources/spi"
	"github.com/rabobank/credhub-client"
)

const (
	InvalidConfigurationObjectError = err.Error("expected a credhub configuration object")
	OnlyOneCredhubSourceError       = err.Error("only one credhub source is allowed")
)

var l, _ = log.GetWithOptions("CREDHUB_SOURCE", log.Standard().WithFailingCriticals().WithLogPrefix(log.Name, log.LogLevel, log.Separator).WithStartingLevel(cfg.LogLevel))

var defaultSource *source

type credentialsIndex map[string]map[string]map[string]string

func newCredentialsIndex() *credentialsIndex {
	index := credentialsIndex(make(map[string]map[string]map[string]string))
	return &index
}

func (ci *credentialsIndex) add(name string) {
	var profiles map[string]map[string]string
	var labels map[string]string
	components := strings.Split(name, "/")
	nameSize := len(components)

	// check if the app exist, if not create it and add a profiles map
	if profiles = (*ci)[components[nameSize-4]]; profiles == nil {
		profiles = make(map[string]map[string]string)
		(*ci)[components[nameSize-4]] = profiles
	}

	// check if the profile exist, if not create it and add a labels map
	if labels = profiles[components[nameSize-3]]; labels == nil {
		labels = make(map[string]string)
		profiles[components[nameSize-3]] = labels
	}

	labels[components[nameSize-2]] = name
}

func (ci *credentialsIndex) contains(app, profile, label string) bool {
	if profiles, found := (*ci)[app]; found {
		if labels, found := profiles[profile]; found {
			if _, found := labels[label]; found {
				return true
			}
		}
	}
	return false
}

func (ci *credentialsIndex) filterCredentials(apps, profiles, labels []string) []string {
	var result []string
	if apps == nil {
		for _, p := range *ci {
			for _, b := range p {
				for _, credential := range b {
					result = append(result, credential)
				}
			}
		}
	} else {
		for _, app := range apps {
			appProfiles := (*ci)[app]
			if profiles == nil {
				for _, b := range appProfiles {
					for _, credential := range b {
						result = append(result, credential)
					}
				}
			} else if appProfiles != nil {
				for _, profile := range profiles {
					profileLabels := appProfiles[profile]
					if labels == nil {
						for _, credential := range profileLabels {
							result = append(result, credential)
						}
					} else if profileLabels != nil {
						for _, label := range labels {
							if credential, found := profileLabels[label]; found {
								result = append(result, credential)
							}
						}
					}
				}
			}
		}
	}
	return result
}

type source struct {
	prefix string
	client credhub.Client
}

func (s *source) Name() string {
	return "credhub"
}

func (s *source) DashboardReport() *string {
	return nil
}

func (s *source) appendProfilesSecrets(app string, profiles []string, label string, result []*domain.PropertySource) []*domain.PropertySource {
	var defaultRequested bool
	for _, profile := range profiles {
		if profile == "default" {
			defaultRequested = true
		}
		name := fmt.Sprintf("%s%s/%s/%s/secrets", s.prefix, app, profile, label)
		fmt.Println("Getting secrets from", name)
		if secrets, e := s.client.GetJsonByName(name); e == nil {
			result = append(result, &domain.PropertySource{
				Source:     name,
				Properties: secrets,
			})
		}
	}
	if !defaultRequested {
		name := fmt.Sprintf("%s%s/%s/%s/secrets", s.prefix, app, "default", label)
		fmt.Println("Getting secrets from", name)
		if secrets, e := s.client.GetJsonByName(name); e == nil {
			result = append(result, &domain.PropertySource{
				Source:     name,
				Properties: secrets,
			})
		}
	}
	return result
}

func (s *source) FindProperties(app string, profiles []string, label string) ([]*domain.PropertySource, error) {
	return s.findProperties(ensureApplication(app), ensureDefaultProfile(profiles), ensureMasterLabel(label))
}

func (s *source) findProperties(apps []string, profiles []string, labels []string) ([]*domain.PropertySource, error) {
	l.Debugf("Find properties for apps: %s, profiles: %v, labels: %s", apps, profiles, labels)
	var result []*domain.PropertySource

	existingCredentials, e := s.getExistingCredentials()
	if e != nil {
		return nil, e
	}

	relevantCredentials := existingCredentials.filterCredentials(apps, ensureDefaultProfile(profiles), labels)
	for _, name := range relevantCredentials {
		if credential, e := s.client.GetJsonByName(name); e != nil {
			// log it
		} else {
			result = append(result, &domain.PropertySource{
				Source:     name,
				Properties: credential,
			})
		}
	}

	return result, nil
}

func (s *source) getExistingCredentials() (*credentialsIndex, error) {
	l.Debugf("Find all credentials for %s", s.prefix)
	if credentials, e := s.client.FindByPath(s.prefix); e != nil {
		l.Errorf("Failed to retrieve credentials for %s : %v", s.prefix, e)
		return nil, e
	} else {
		result := newCredentialsIndex()
		for _, credential := range credentials.Credentials {
			l.Debugf("Found Credentials : %s", credential.Name)
			result.add(credential.Name)
		}
		return result, nil
	}
}

// func (s *source) filterCredentials(credentials *credentialsIndex, apps []string, profiles []string, labels []string) []string {
//     if len(*credentials) == 0 {
//         return nil
//     }
//
//     var matchingApps []map[string]map[string]string
//     var defaultPresent bool
//     if apps == nil {
//         for _, v := range *credentials {
//             matchingApps = append(matchingApps, v)
//         }
//     } else if profiles == nil {
//         for _, app := range apps {
//             if matchingProfiles := (*credentials)[app]; matchingProfiles != nil {
//                 matchingApps = append(matchingApps, matchingProfiles)
//             }
//             if app == "application" {
//                 defaultPresent = true
//             }
//         }
//         if !defaultPresent {
//             if matchingProfiles := (*credentials)["application"]; matchingProfiles != nil {
//                 matchingApps = append(matchingApps, matchingProfiles)
//             }
//         }
//     } else if labels == nil {
//
//     }
//
//     matchingNames := make(map[string]bool)
//     if apps == nil {
//         return credentials
//     } else if profiles == nil {
//         for _, name := range credentials {
//         }
//         matchingNames[fmt.Sprintf("%s%s", s.prefix, "application")] = true
//         for _, app := range apps {
//             matchingNames[fmt.Sprintf("%s%s", s.prefix, app)] = true
//         }
//     } else if labels == nil {
//         matchingNames[fmt.Sprintf("%s%s/%s", s.prefix, "application", "default")] = true
//         for _, app := range apps {
//             matchingNames[fmt.Sprintf("%s%s/%s", s.prefix, app, "default")] = true
//             for _, profile := range profiles {
//                 matchingNames[fmt.Sprintf("%s%s/%s", s.prefix, app, profile)] = true
//             }
//         }
//     } else {
//         matchingNames[fmt.Sprintf("%s%s/%s/%s", s.prefix, "application", "default", "master")] = true
//         for _, app := range apps {
//             matchingNames[fmt.Sprintf("%s%s/%s/%s", s.prefix, app, "default", "master")] = true
//             for _, profile := range profiles {
//                 matchingNames[fmt.Sprintf("%s%s/%s/%s", s.prefix, app, profile, "master")] = true
//                 for _, label := range labels {
//                     matchingNames[fmt.Sprintf("%s%s/%s/%s", s.prefix, app, profile, label)] = true
//                 }
//             }
//         }
//     }
//
//     var result []string
//     for _, credential := range credentials {
//         if matchingNames[credential] {
//             result = append(result, credential)
//         }
//     }
//
//     return result
// }

func (s *source) listSecrets(apps []string, profiles []string, labels []string) (map[string]map[string]map[string][]string, error) {
	var credentials *credentialsIndex
	var e error

	if credentials, e = s.getExistingCredentials(); e != nil {
		return nil, e
	}

	relevantCredentials := credentials.filterCredentials(apps, profiles, labels)
	result := make(map[string]map[string]map[string][]string)
	for _, name := range relevantCredentials {
		if credential, e := s.client.GetJsonByName(name); e != nil {
			// log it
		} else {
			app, profile, label := extractScope(name)
			appSecrets := result[app]
			if appSecrets == nil {
				appSecrets = make(map[string]map[string][]string)
				result[app] = appSecrets
			}
			profileSecrets := appSecrets[profile]
			if profileSecrets == nil {
				profileSecrets = make(map[string][]string)
				appSecrets[profile] = profileSecrets
			}
			for key := range credential {
				profileSecrets[label] = append(profileSecrets[label], key)
			}
		}
	}

	return result, nil
}

func (s *source) addSecrets(apps []string, profiles []string, labels []string, secrets map[string]any) error {
	if len(apps) == 0 {
		apps = []string{"application"}
	}
	if len(profiles) == 0 {
		profiles = []string{"default"}
	}
	if len(labels) == 0 {
		labels = []string{"master"}
	}
	secrets = flattenSecrets("", secrets)
	existingCredentials, e := s.getExistingCredentials()
	if e != nil {
		return e
	}
	for _, app := range apps {
		for _, profile := range profiles {
			for _, label := range labels {
				credentialName := fmt.Sprintf("%s%s/%s/%s/secrets", s.prefix, app, profile, label)
				if existingCredentials.contains(app, profile, label) {
					if existingCredential, e := s.client.GetJsonByName(credentialName); e != nil {
						fmt.Println("Unable to read credentials", credentialName)
						return e
					} else {
						secrets = mergeSecrets(existingCredential, secrets)
					}
				}

				if _, e := s.client.SetJsonByName(credentialName, secrets); e != nil {
					fmt.Println("Failed to write credentials", e)
					return e
				}
			}
		}
	}
	return nil
}

func mergeSecrets(existingSecrets map[string]any, secrets map[string]any) map[string]any {
	for k, v := range secrets {
		existingSecrets[k] = v
	}
	return existingSecrets
}

func flattenSecrets(prefix string, secrets map[string]any) map[string]any {
	return secrets
}

func extractScope(name string) (string, string, string) {
	components := strings.Split(name, "/")
	size := len(components)
	return components[size-4], components[size-3], components[size-2]
}

func Source(sourceConfig domain.SourceConfig) (result spi.Source, e error) {
	if defaultSource != nil {
		return nil, OnlyOneCredhubSourceError
	} else if credhubConfig, isType := sourceConfig.(*domain.CredhubConfig); !isType {
		return nil, InvalidConfigurationObjectError
	} else {
		s := &source{
			prefix: credhubConfig.Prefix,
		}
		if !strings.HasPrefix(s.prefix, "/") {
			s.prefix = "/" + s.prefix
		}
		if !strings.HasSuffix(s.prefix, "/") {
			s.prefix = s.prefix + "/"
		}

		if credhubConfig.Client != nil && credhubConfig.Secret != nil {
			if s.client, e = credhub.New(&credhub.Options{
				Client: *credhubConfig.Client,
				Secret: *credhubConfig.Secret,
			}); e != nil {
				return
			}
		} else {
			// creating a credhub client with mtls authentication doesn't raise errors
			s.client, _ = credhub.New(nil)
		}
		defaultSource = s
		return s, nil
	}
}

func ensureApplication(app string) []string {
	if len(app) == 0 || app == "application" {
		return []string{"application"}
	}
	return []string{app, "application"}
}

func ensureMasterLabel(label string) []string {
	if len(label) == 0 || label == "master" {
		return []string{"master"}
	}
	return []string{label, "master"}
}

func ensureDefaultProfile(profiles []string) []string {
	if len(profiles) == 0 {
		return []string{"default"}
	}
	containsDefault := false
	for _, profile := range profiles {
		if profile == "default" {
			containsDefault = true
			break
		}
	}

	if containsDefault {
		return profiles
	} else {
		return append(profiles, "default")
	}
}
