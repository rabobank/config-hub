package git_source

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"

	err "github.com/gomatbase/go-error"
	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources"
	"gopkg.in/yaml.v2"
)

const (
	InvalidConfigurationObjectError = err.Error("expected GitConfig configuration object")
)

var l, _ = log.GetWithOptions("GIT_SOURCE", log.Standard().WithLogPrefix(log.Name, log.LogLevel, log.Separator).WithStartingLevel(cfg.LogLevel))

type source struct {
	repo         string
	baseDir      string
	defaultLabel string
	searchPaths  []string
}

func (s *source) FindProperties(app string, profiles []string, requestedLabel string) ([]*domain.PropertySource, error) {
	l.Debugf("Finding properties from git source %s for app:%s, profiles:%s and label %s", s.repo, app, profiles, requestedLabel)
	label := s.defaultLabel
	if len(requestedLabel) != 0 {
		label = requestedLabel
	}

	if e := refresh(s.baseDir, label); e == nil {
		if label == "master" {
			if e = refresh(s.baseDir, "main"); e != nil {
				return nil, e
			} else {
				s.defaultLabel = "main"
			}
		}
	} else {
		l.Errorf("Failed to refresh repository %s : %v", s.repo, e)
	}

	var sourcesProperties []*domain.PropertySource
	// search all app specific files
	for _, file := range s.findFiles(app, profiles) {
		if fileProperties, e := readFile(file); e != nil {
			l.Error(e)
		} else {
			sourcesProperties = append(sourcesProperties, fileProperties)
		}
	}

	return sourcesProperties, nil
}

func (s *source) findFiles(app string, profiles []string) []*os.File {
	var files []*os.File
	for _, profile := range profiles {
		for _, baseDir := range s.searchPaths {
			baseDir = strings.ReplaceAll(baseDir, "{application}", app)
			baseDir = strings.ReplaceAll(baseDir, "{profile}", profile)
			if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s-%s.yml", app, profile))); file != nil {
				files = append(files, file)
			}
			if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s-%s.properties", app, profile))); file != nil {
				files = append(files, file)
			}
		}
		if file := openFile(path.Join(s.baseDir, fmt.Sprintf("%s-%s.yml", app, profile))); file != nil {
			files = append(files, file)
		}
		if file := openFile(path.Join(s.baseDir, fmt.Sprintf("%s-%s.properties", app, profile))); file != nil {
			files = append(files, file)
		}
	}
	for _, baseDir := range s.searchPaths {
		baseDir = strings.ReplaceAll(baseDir, "{application}", app)
		if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s.yml", app))); file != nil {
			files = append(files, file)
		}
		if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s.properties", app))); file != nil {
			files = append(files, file)
		}
	}
	if file := openFile(path.Join(s.baseDir, fmt.Sprintf("%s.yml", app))); file != nil {
		files = append(files, file)
	}
	if file := openFile(path.Join(s.baseDir, fmt.Sprintf("%s.properties", app))); file != nil {
		files = append(files, file)
	}

	return files
}

func openFile(filename string) *os.File {
	if f, e := os.Open(filename); e == nil {
		return f
	}
	return nil
}

func Source(sourceConfig domain.SourceConfig, index int) (sources.Source, error) {
	if gitConfig, isType := sourceConfig.(*domain.GitConfig); !isType {
		return nil, InvalidConfigurationObjectError
	} else {
		result := &source{}
		result.baseDir = path.Join(cfg.BaseDir, fmt.Sprintf("config-repo-%d", index))

		// this block can be used for local tests cleaning up temporary folders
		if _, e := os.Stat(result.baseDir); e == nil {
			// an object already exists. Let's attempt to delete it
			if e = os.RemoveAll(result.baseDir); e != nil {
				return nil, e
			}
		}

		if e := os.MkdirAll(result.baseDir, os.ModeDir|os.ModePerm); e != nil {
			return nil, e
		}
		if e := initializeGitRepository(gitConfig, result.baseDir); e != nil {
			return nil, e
		}
		addCredentials(gitConfig)

		if gitConfig.DefaultLabel == nil || len(*gitConfig.DefaultLabel) == 0 {
			result.defaultLabel = "master"
		} else {
			result.defaultLabel = *gitConfig.DefaultLabel
		}

		return result, nil
	}

}

func readFile(file *os.File) (*domain.PropertySource, error) {
	result := &domain.PropertySource{
		Source:     file.Name(),
		Properties: make(map[string]interface{}),
	}
	var e error
	if strings.HasSuffix(file.Name(), ".properties") {
		result.Properties, e = readPropertiesFile(file)
	} else {
		result.Properties, e = readYamlFile(file)
	}
	return result, e
}

func readYamlFile(file *os.File) (map[string]interface{}, error) {
	object := new(interface{})
	if e := yaml.NewDecoder(file).Decode(object); e != nil {
		return nil, e
	}

	properties := make(map[string]interface{})
	e := flattenProperties("", object, &properties)
	return properties, e
}

func flattenProperties(prefix string, object interface{}, properties *map[string]interface{}) error {

	errors := err.Errors()

	t := reflect.ValueOf(object).Kind()
	if t == reflect.Pointer {
		object = reflect.ValueOf(object).Elem().Interface()
		t = reflect.ValueOf(object).Kind()
	}

	switch t {
	case reflect.Map:
		// if it's a map we expect it to be a type of map[string]interface{}
		for key, value := range object.(map[any]any) {
			if e := flattenProperties(prefix+"."+key.(string), value, properties); e != nil {
				errors.AddError(e)
			}
		}
	case reflect.Array:
		// if it's an array we expect it to be a type of []]interface{}
		for i, value := range object.([]interface{}) {
			if e := flattenProperties(prefix+"["+strconv.Itoa(i)+"]", value, properties); e != nil {
				errors.AddError(e)
			}
		}
	default:
		if len(prefix) == 0 || prefix[0] != '.' {
			(*properties)[""] = object
		} else {
			(*properties)[prefix[1:]] = object
		}
	}

	if errors.Count() > 0 {
		return errors
	}

	return nil
}

func readPropertiesFile(file *os.File) (map[string]interface{}, error) {
	scanner := bufio.NewScanner(file)
	defer file.Close()

	properties := make(map[string]interface{})
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), "=")
		if found {
			properties[key] = value
		} else {
			properties[key] = nil
		}
	}
	if e := scanner.Err(); e != nil && e != io.EOF {
		return nil, e
	}

	return properties, nil
}
