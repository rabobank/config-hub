package git_source

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"

	err "github.com/gomatbase/go-error"
	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources/spi"
	"gopkg.in/yaml.v3"
)

const (
	InvalidConfigurationObjectError = err.Error("expected GitConfig configuration object")
)

var l, _ = log.GetWithOptions("GIT_SOURCE", log.Standard().WithLogPrefix(log.Name, log.LogLevel, log.Separator).WithStartingLevel(cfg.LogLevel))

var dashboardTemplate = template.Must(template.New("dashboard").Parse("" +
	"                    <div class=\"source-report-group\">\n" +
	"                        <h3 class=\"title\">Remote Branches</h3>\n" +
	"{{if .Error}}" +
	"                        <div class=\"error\">{{.Error}}</div>\n" +
	"{{end}}" +
	"{{range .Remote}}" +
	"                        <div class=\"git-branch\">\n" +
	"                            <div class=\"source-report-line\">\n" +
	"                                <span class=\"label\">Name</span>&nbsp;<span class=\"value\">{{.Name}}</span>\n" +
	"                            </div>\n" +
	"                            <div class=\"source-report-line\">\n" +
	"                               <span class=\"label\">Commit</span>&nbsp;<span class=\"value\">{{.CommitId}}</span>\n" +
	"                            </div>\n" +
	"                            <div class=\"source-report-line\">\n" +
	"                                <span class=\"label\">Commit Date</span>&nbsp;<span class=\"value\">{{.Date}}</span>\n" +
	"                            </div>\n" +
	"                        </div>\n" +
	"{{else}}" +
	"                        <div class=\"error\">Unable to show remote branches. May be an invalid PAT.</div>\n" +
	"{{end}}" +
	"                    </div>\n" +
	"                    <div class=\"source-report-group\">\n" +
	"                        <h3 class=\"title\">Local Branches</h3>\n" +
	"{{range .Local}}" +
	"                        <div class=\"git-branch\">\n" +
	"                            <div class=\"source-report-line\">\n" +
	"                                <span class=\"label\">Name</span>&nbsp;<span class=\"value\">{{.Name}}</span>\n" +
	"                            </div>\n" +
	"                            <div class=\"source-report-line\">\n" +
	"                               <span class=\"label\">Commit</span>&nbsp;<span class=\"value\">{{.CommitId}}</span>\n" +
	"                            </div>\n" +
	"                            <div class=\"source-report-line\">\n" +
	"                                <span class=\"label\">Commit Date</span>&nbsp;<span class=\"value\">{{.Date}}</span>\n" +
	"                            </div>\n" +
	"                        </div>\n" +
	"{{else}}" +
	"                        <div class=\"error\">No local branches checked out. Probably never been successfully used.</div>\n" +
	"{{end}}" +
	"                    </div>\n"))

type source struct {
	repository   *Repository
	repo         string
	baseDir      string
	defaultLabel string
	searchPaths  []string

	lock sync.Mutex
}

type Branches struct {
	Error  string
	Remote []Branch
	Local  []Branch
}

func (s *source) String() string {
	return fmt.Sprintf("GitSource{repo:%s, baseDir:%s, defaultLabel:%s, searchPaths:%s}", s.repo, s.baseDir, s.defaultLabel, s.searchPaths)
}

func (s *source) Name() string {
	return s.repo
}

func (s *source) DashboardReport() *string {
	branches := &Branches{}
	var e error

	s.lock.Lock()
	defer s.lock.Unlock()
	if branches.Remote, e = s.repository.Branches(Remote); e != nil {
		// the error is probably from a PAT issue, listing tracked remote branches is not expected to fail
		l.Errorf("Unable to list remote branches : %v", e)
		branches.Error = e.Error()
	}
	if branches.Local, e = s.repository.Branches(Local); e != nil {
		// not really expected
		l.Errorf("Unable to list remote branches : %v", e)
	}
	buffer := &bytes.Buffer{}
	if e = dashboardTemplate.Execute(buffer, branches); e != nil {
		l.Errorf("Failure to execute the template : %v", e)
		return nil
	}

	report := buffer.String()
	return &report
}

func (s *source) FindProperties(app string, profiles []string, requestedLabel string) ([]*domain.PropertySource, error) {
	l.Debugf("Finding properties from git source %s for app:%s, profiles:%s and label %s", s.repo, app, profiles, requestedLabel)

	s.lock.Lock()
	defer s.lock.Unlock()

	label := s.defaultLabel
	if len(requestedLabel) != 0 {
		label = requestedLabel
	}

	if e := s.repository.Refresh(label); e != nil {
		if label == "master" {
			if e = s.repository.Refresh("main"); e == nil {
				s.defaultLabel = "main"
			}
		}
		if e != nil {
			l.Errorf("Failed to refresh repository %s : %v", s.repo, e)
		}
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
	// TODO improve this process

	l.Info("Searching for files for app ", app, " and profiles ", profiles)
	files := make([]*os.File, 0)
	for _, profile := range profiles {
		for _, baseDir := range s.searchPaths {
			baseDir = strings.ReplaceAll(baseDir, "{application}", app)
			baseDir = strings.ReplaceAll(baseDir, "{profile}", profile)
			if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s-%s.yml", app, profile))); file != nil {
				files = append(files, file)
			}
			if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s-%s.yaml", app, profile))); file != nil {
				files = append(files, file)
			}
			if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s-%s.properties", app, profile))); file != nil {
				files = append(files, file)
			}
			if app != "application" {
				if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("application-%s.yml", profile))); file != nil {
					files = append(files, file)
				}
				if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("application-%s.yaml", profile))); file != nil {
					files = append(files, file)
				}
				if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("application-%s.properties", profile))); file != nil {
					files = append(files, file)
				}
			}
		}
	}
	for _, baseDir := range s.searchPaths {
		baseDir = strings.ReplaceAll(baseDir, "{application}", app)
		if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s.yml", app))); file != nil {
			files = append(files, file)
		}
		if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s.yaml", app))); file != nil {
			files = append(files, file)
		}
		if file := openFile(path.Join(s.baseDir, baseDir, fmt.Sprintf("%s.properties", app))); file != nil {
			files = append(files, file)
		}
		if app != "application" {
			if file := openFile(path.Join(s.baseDir, baseDir, "application.yml")); file != nil {
				files = append(files, file)
			}
			if file := openFile(path.Join(s.baseDir, baseDir, "application.yaml")); file != nil {
				files = append(files, file)
			}
			if file := openFile(path.Join(s.baseDir, baseDir, "application.properties")); file != nil {
				files = append(files, file)
			}

		}
	}

	l.Info("Found files ", files)

	return files
}

func Source(sourceConfig domain.SourceConfig, index int) (spi.Source, error) {
	if gitConfig, isType := sourceConfig.(*domain.GitConfig); !isType {
		return nil, InvalidConfigurationObjectError
	} else {
		var e error
		result := &source{}
		result.baseDir = path.Join(cfg.BaseDir, fmt.Sprintf("config-repo-%d", index))

		// this block can be used for local tests cleaning up temporary folders
		if _, e = os.Stat(result.baseDir); e == nil {
			// an object already exists. Let's attempt to delete it
			if e = os.RemoveAll(result.baseDir); e != nil {
				return nil, e
			}
		}

		if e = os.MkdirAll(result.baseDir, os.ModeDir|os.ModePerm); e != nil {
			return nil, e
		}

		if result.repository, e = Git(gitConfig, result.baseDir); e != nil {
			return nil, e
		}
		addCredentials(gitConfig)

		if gitConfig.DefaultLabel == nil || len(*gitConfig.DefaultLabel) == 0 {
			result.defaultLabel = "master"
		} else {
			result.defaultLabel = *gitConfig.DefaultLabel
		}

		result.searchPaths = gitConfig.SearchPaths
		result.repo = gitConfig.Uri

		return result, nil
	}

}

func openFile(filename string) *os.File {
	if f, e := os.Open(filename); e == nil {
		l.Info("reading ", filename)
		return f
	}
	return nil
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

	if object == nil {
		object = ""
	}

	t := reflect.ValueOf(object).Kind()
	if t == reflect.Pointer {
		object = reflect.ValueOf(object).Elem().Interface()
		t = reflect.ValueOf(object).Kind()
	}

	switch t {
	case reflect.Map:
		// if it's a map we expect it to be a type of map[string]interface{}
		for key, value := range object.(map[string]any) {
			if e := flattenProperties(prefix+"."+key, value, properties); e != nil {
				errors.AddError(e)
			}
		}
	case reflect.Slice:
		// if it's an array we expect it to be a type of []]interface{}
		for i, value := range object.([]interface{}) {
			if e := flattenProperties(prefix+"["+strconv.Itoa(i)+"]", value, properties); e != nil {
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
		if t == reflect.String {
			// special string-to-boolean cases
			switch strings.ToUpper(object.(string)) {
			case "OFF":
				object = false
			case "ON":
				object = true
			}
		}

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
