package cfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	err "github.com/gomatbase/go-error"
	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/credhub-client"
)

var (
	Version  = "0.0.0"
	LogLevel = log.INFO

	DebugOutput = os.Getenv("DEBUG_OUTPUT")

	// credhub & uaaConfiguration configuration
	OpenIdUrl   = os.Getenv("OPENID_URL")
	UaaUrl      = os.Getenv("UAA_URL")
	Client      = os.Getenv("CLIENT_ID")
	Secret      = os.Getenv("CLIENT_SECRET")
	ClientScope = os.Getenv("AUTHORIZED_CLIENT_SCOPE")

	// cf configuration
	CfUrl             = "https://api.cf.internal"
	ServiceInstanceId = os.Getenv("cf.service-instance-id")

	Port        = "8080"
	HttpTimeout = 5
	Sources     []domain.SourceConfig

	OneTimeToken *string
	BaseDir      string
)

type CfApplication struct {
	ApplicationId   string   `json:"application_id"`
	ApplicationName string   `json:"application_name"`
	ApplicationUris []string `json:"application_uris"`
	CfApi           string   `json:"cf_api"`
	Limits          struct {
		Fds int `json:"fds"`
	} `json:"limits"`
	Name             string      `json:"name"`
	OrganizationId   string      `json:"organization_id"`
	OrganizationName string      `json:"organization_name"`
	SpaceId          string      `json:"space_id"`
	SpaceName        string      `json:"space_name"`
	Uris             []string    `json:"uris"`
	Users            interface{} `json:"users"`
}

func init() {
	if level, found := os.LookupEnv("LOG_LEVEL"); found {
		if severity := log.LevelSeverity(level); severity != log.UNKNOWN {
			LogLevel = severity
		}
	}
}

func Println(items ...any) {
	if len(DebugOutput) != 0 {
		f, _ := os.OpenFile(DebugOutput, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
		fmt.Fprintln(f, items...)
		f.Close()
	}
}

func FinishServerConfiguration() error {

	errors := err.Errors()

	if port, found := os.LookupEnv("PORT"); found {
		Port = port
	}

	var e error
	BaseDir, e = filepath.Abs(path.Dir(os.Args[0]))
	if e != nil {
		errors.Add(fmt.Sprintf("Unable to assess base dir : %v", e))
	}

	if vcap, found := os.LookupEnv("VCAP_APPLICATION"); found {
		// running inside cf, get the cf url from the environment
		cfApplication := &CfApplication{}
		if e = json.Unmarshal([]byte(vcap), cfApplication); e != nil {
			errors.Add(fmt.Sprintf("Unable to parse VCAP_APPLICATION : %v", e))
		} else {
			CfUrl = cfApplication.CfApi
		}
	} else if CfUrl, found = os.LookupEnv("CF_URL"); !found {
		errors.Add("No CF_URL provided")
	}

	if credhubRef, found := os.LookupEnv("CREDHUB-REF"); found {
		credhubClient, _ := credhub.New(nil)
		if credentials, e := credhubClient.GetJsonByName(credhubRef); e != nil {
			errors.Add(fmt.Sprintf("Unable to retrieve credhub credentials from %s : %v", credhubRef, e))
		} else {
			var isType bool
			if Client, isType = credentials["uaa_client"].(string); !isType {
				errors.Add("uaa_client is not a string")
			}
			if Client, isType = credentials["uaa_secret"].(string); !isType {
				errors.Add("uaa_secret is not a string")
			}
			var sources string
			if sources, isType = credentials["sources"].(string); !isType {
				errors.Add("sources is not a string")
			} else if Sources, e = jsonToSources(sources); e != nil {
				errors.AddError(e)
			}
		}
	} else if sources, found := os.LookupEnv("CH_SOURCES"); found {
		if Sources, e = jsonToSources(sources); e != nil {
			errors.AddError(e)
		}
	} else {
		errors.Add("No sources provided")
	}

	if errors.Count() == 0 {
		errors = nil
	}

	return errors
}

func jsonToSources(sources string) ([]domain.SourceConfig, error) {
	var propertiesArray []map[string]interface{}
	if e := json.Unmarshal([]byte(sources), &propertiesArray); e != nil {
		return nil, e
	}

	sourcesArray := make([]domain.SourceConfig, len(propertiesArray))
	errors := err.Errors()

	for i, properties := range propertiesArray {
		if sourceType, found := properties["type"]; found {
			switch sourceType {
			case "credhub":
				credhubConfig := &domain.CredhubConfig{}
				if e := credhubConfig.FromMap(properties); e != nil {
					errors.AddError(e)
				} else {
					sourcesArray[i] = credhubConfig
				}
			case "git":
				gitConfig := &domain.GitConfig{}
				if e := gitConfig.FromMap(properties); e != nil {
					errors.AddError(e)
				} else {
					sourcesArray[i] = gitConfig
				}
			}
		} else {
			errors.Add(fmt.Sprintf("source without source type %v", properties))
		}
	}

	if errors.Count() == 0 {
		errors = nil
	}

	return sourcesArray, errors
}

func FinishCredentialsConfiguration() {
	if token := os.Getenv("CH_TOKEN"); len(token) != 0 {
		OneTimeToken = &token
	}
}
