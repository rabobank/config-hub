package git_source

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	err "github.com/gomatbase/go-error"
	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources/spi"
	"github.com/rabobank/config-hub/util"
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

func (s *source) ClearCache() {
	l.Debugf("Clearing cache for git source %s", s.repo)
	s.repository.ClearTTL()
}

func (s *source) FindProperties(apps []string, profiles []string, requestedLabel string) ([]*domain.PropertySource, error) {
	l.Debugf("Finding properties from git source %s for app(s):%v, profiles:%s and label %s", s.repo, apps, profiles, requestedLabel)

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
		if UnableToFetchError.IsKindOf(e) {
			if s.repository.failOnFetch {
				l.Errorf("Fetching failed and repository setup to fail on fetch: %v", e)
				return nil, e
			}
		} else if e != nil {
			l.Errorf("Error when refreshing repository: %v", e)
			return nil, e
		}
	}

	var sourcesProperties []*domain.PropertySource
	// search all app specific files
	for _, file := range s.findFiles(apps, profiles) {
		if fileProperties, e := readFile(file); e != nil {
			l.Error(e)
		} else {
			sourcesProperties = append(sourcesProperties, fileProperties)

			// TODO cleanup
			if v, found := util.Get("spring.config.import", fileProperties.Properties); found {
				if importedFilename, isType := v.(string); !isType {
					l.Errorf("Imported spring config file not a string : %v\n", v)
				} else {
					for _, file := range s.findFile(apps, profiles, importedFilename) {
						if fileProperties, e = readFile(file); e != nil {
							l.Errorf("Unable to read imported file %s : %v\n", importedFilename, e)
						} else {
							sourcesProperties = append(sourcesProperties, fileProperties)
						}
					}
				}
			}
		}
	}

	return sourcesProperties, nil
}

func addExistingFiles(file string, files []*os.File) []*os.File {
	for _, ext := range []string{"yml", "yaml", "properties"} {
		l.Tracef("Search for file %s%s", file, ext)
		if file := openFile(file + ext); file != nil {
			files = append(files, file)
		}
	}
	return files
}

func addExistingFile(file string, files []*os.File) []*os.File {
	l.Tracef("Search for file %s", file)
	if file := openFile(file); file != nil {
		files = append(files, file)
	}
	return files
}

func findMatchingPaths(base string, segments []string) []string {
	var paths []string
	if entries, e := os.ReadDir(base); e != nil {
		l.Errorf("Unable to read directory %s : %v", base, e)
	} else {
		match := regexp.MustCompile("^" + strings.ReplaceAll(segments[0], "*", ".*") + "$")
		for _, entry := range entries {
			if entry.IsDir() && match.MatchString(entry.Name()) {
				dir := path.Join(base, entry.Name())
				if len(segments) == 1 {
					paths = append(paths, dir)
				} else {
					paths = append(paths, findMatchingPaths(dir, segments[1:])...)
				}
			}
		}
	}
	return paths
}

func (s *source) matchingPaths(searchPath, app, profile string) []string {
	// we replace the placeholders of the search path
	searchPath = strings.ReplaceAll(strings.ReplaceAll(searchPath, "{application}", app), "{profile}", profile)

	// now we check if there are wildcards
	if strings.Contains(searchPath, "*") {
		return findMatchingPaths(s.baseDir, strings.Split(searchPath, "/"))
	}

	// no wildcards so we return the search path as it is
	return []string{path.Join(s.baseDir, searchPath)}
}

func (s *source) findFiles(apps []string, profiles []string) []*os.File {
	// TODO improve this process

	var searchPaths []string
	// TODO JV can't remember if this piece of code is still relevant
	for _, searchPath := range s.searchPaths {
		if strings.Contains(searchPath, "{application}") {
			for _, app := range apps {
				searchPaths = append(searchPaths, strings.ReplaceAll(searchPath, "{application}", app))
			}
		} else {
			searchPaths = append(searchPaths, searchPath)
		}
	}

	l.Info("Searching for files for app(s) ", apps, " and profiles ", profiles)
	files := make([]*os.File, 0)
	for _, profile := range profiles {
		for _, baseDir := range searchPaths {
			for _, app := range apps {
				profileSearchPath := strings.Contains(baseDir, "{profile}")
				for _, searchPath := range s.matchingPaths(baseDir, app, profile) {
					files = addExistingFiles(path.Join(searchPath, fmt.Sprintf("%s-%s.", app, profile)), files)
					if profileSearchPath {
						files = addExistingFiles(path.Join(searchPath, fmt.Sprintf("%s.", app)), files)
					}
				}
			}
		}
	}
	for _, baseDir := range searchPaths {
		if !strings.Contains(baseDir, "{profile}") {
			for _, app := range apps {
				for _, searchPath := range s.matchingPaths(baseDir, app, "") {
					files = addExistingFiles(path.Join(searchPath, fmt.Sprintf("%s.", app)), files)
				}
			}
		}
	}

	if !util.HasApplication(apps) {
		for _, profile := range profiles {
			for _, baseDir := range searchPaths {
				profileSearchPath := strings.Contains(baseDir, "{profile}")
				for _, searchPath := range s.matchingPaths(baseDir, "application", profile) {
					files = addExistingFiles(path.Join(searchPath, fmt.Sprintf("application-%s.", profile)), files)
					if profileSearchPath {
						files = addExistingFiles(path.Join(searchPath, "application."), files)
					}
				}
			}
		}
		for _, baseDir := range searchPaths {
			if !strings.Contains(baseDir, "{profile}") {
				for _, app := range apps {
					for _, searchPath := range s.matchingPaths(baseDir, app, "") {
						files = addExistingFiles(path.Join(searchPath, "application."), files)
					}
				}
			}
		}
	}

	return files
}

func (s *source) findFile(apps []string, profiles []string, filename string) []*os.File {
	var searchPaths []string
	for _, searchPath := range s.searchPaths {
		if strings.Contains(searchPath, "{application}") {
			for _, app := range apps {
				if strings.Contains(searchPath, "{profile}") {
					for _, profile := range profiles {
						searchPaths = append(searchPaths, strings.ReplaceAll(strings.ReplaceAll(searchPath, "{application}", app), "{profile}", profile))
					}
				} else {
					searchPaths = append(searchPaths, strings.ReplaceAll(searchPath, "{application}", app))
				}
			}
		} else if strings.Contains(searchPath, "{profile}") {
			for _, profile := range profiles {
				searchPaths = append(searchPaths, strings.ReplaceAll(searchPath, "{profile}", profile))
			}
		} else {
			searchPaths = append(searchPaths, searchPath)
		}
	}

	l.Info("Searching for file", filename, "for app(s) ", apps, " and profiles ", profiles)
	files := make([]*os.File, 0)
	for _, searchPath := range searchPaths {
		l.Info("Check search path $s", searchPath)
		files = addExistingFile(path.Join(s.baseDir, searchPath, filename), files)
	}

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
	// fmt.Println("Testing", filename)
	if f, e := os.Open(filename); e == nil {
		l.Info("reading", filename)
		return f
	}
	return nil
}

func readFile(file *os.File) (*domain.PropertySource, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in readFile", r)
		}
	}()

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
	// we expect it to always be a map[string]any
	object := make(map[string]any)
	decoder := yaml.NewDecoder(file)
	var e error
	for {
		saved := object
		if e = decoder.Decode(&object); e != nil {
			if e == io.EOF {
				e = nil
				break
			}
			fmt.Printf("Error decoding %s: %s", file.Name(), e)
			break
		}

		// JV A bug with the yaml parser zeroes the map when reading an empty document (section), this prevents it
		if len(object) == 0 {
			object = saved
		}
	}

	// properties := make(map[string]interface{})
	// e := flattenProperties("", object, &properties)
	return object, e
}

// func flattenProperties(prefix string, object interface{}, properties *map[string]interface{}) error {
//
// 	errors := err.Errors()
//
// 	if object == nil {
// 		object = ""
// 	}
//
// 	t := reflect.ValueOf(object).Kind()
// 	if t == reflect.Pointer {
// 		object = reflect.ValueOf(object).Elem().Interface()
// 		t = reflect.ValueOf(object).Kind()
// 	}
//
// 	switch t {
// 	case reflect.Map:
// 		// if it's a map we expect it to be a type of map[string]interface{}, although yaml allows for numbers to be keys in which case they are meant to array/map indexes...
// 		if m, isType := object.(map[string]interface{}); isType {
// 			for key, value := range m {
// 				if e := flattenProperties(prefix+"."+key, value, properties); e != nil {
// 					errors.AddError(e)
// 				}
// 			}
// 		} else {
// 			for key, value := range object.(map[any]any) {
// 				if reflect.TypeOf(key).Kind() == reflect.Int {
// 					if e := flattenProperties(fmt.Sprintf("%s[%v]", prefix, key), value, properties); e != nil {
// 						errors.AddError(e)
// 					}
// 				} else {
// 					if e := flattenProperties(fmt.Sprintf("%s.%v", prefix, key), value, properties); e != nil {
// 						errors.AddError(e)
// 					}
// 				}
// 			}
// 		}
// 	case reflect.Slice:
// 		// if it's an array we expect it to be a type of []interface{}
// 		if len(object.([]interface{})) == 0 {
// 			if len(prefix) == 0 || prefix[0] != '.' {
// 				(*properties)[prefix] = object
// 			} else {
// 				(*properties)[prefix[1:]] = object
// 			}
// 		} else {
// 			for i, value := range object.([]interface{}) {
// 				if e := flattenProperties(prefix+"["+strconv.Itoa(i)+"]", value, properties); e != nil {
// 					errors.AddError(e)
// 				}
// 			}
// 		}
// 	case reflect.Array:
// 		// if it's an array we expect it to be a type of []interface{}
// 		if len(object.([]interface{})) == 0 {
// 			if len(prefix) == 0 || prefix[0] != '.' {
// 				(*properties)[prefix] = object
// 			} else {
// 				(*properties)[prefix[1:]] = object
// 			}
// 		} else {
// 			for i, value := range object.([]interface{}) {
// 				if e := flattenProperties(prefix+"["+strconv.Itoa(i)+"]", value, properties); e != nil {
// 					errors.AddError(e)
// 				}
// 			}
// 		}
// 	default:
// 		if t == reflect.String {
// 			// special string-to-boolean cases
// 			switch strings.ToUpper(object.(string)) {
// 			case "OFF":
// 				object = false
// 			case "ON":
// 				object = true
// 			}
// 		}
//
// 		if len(prefix) == 0 || prefix[0] != '.' {
// 			(*properties)[prefix] = object
// 		} else {
// 			(*properties)[prefix[1:]] = object
// 		}
// 	}
//
// 	if errors.Count() > 0 {
// 		return errors
// 	}
//
// 	return nil
// }

func readPropertiesFile(file *os.File) (map[string]interface{}, error) {
	scanner := bufio.NewScanner(file)
	defer func() { _ = file.Close() }()

	properties := make(map[string]interface{})
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), "=")
		if found && !strings.HasPrefix(key, "#") {
			properties[key] = value
		}
	}
	if e := scanner.Err(); e != nil && e != io.EOF {
		return nil, e
	}

	return properties, nil
}
