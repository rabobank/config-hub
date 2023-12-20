package git_source

import (
	"fmt"
	"net/http"

	"config-hub/domain"
	err "github.com/gomatbase/go-error"
	"github.com/gomatbase/go-we"
	"github.com/gomatbase/go-we/util"
)

const (
	HttpUriFormat = "%s://%s%s"
)

var credentials = make(map[string]*domain.GitConfig)

func addCredentials(config *domain.GitConfig) {
	credentials[config.Uri] = config
}

func ServeCredentials(w we.ResponseWriter, r we.RequestScope) error {
	credentialsRequest, e := util.ReadJsonBody[domain.CredentialsRequest](r)
	if e != nil {
		return e
	}

	if gitConfig := credentials[fmt.Sprintf(HttpUriFormat, credentialsRequest.Protocol, credentialsRequest.Host, credentialsRequest.Repo)]; gitConfig != nil {
		response := &domain.HttpCredentials{
			Username: ite[string](gitConfig.Username, "user"),
			Password: ite[string](gitConfig.Password, "password"),
		}
		return util.ReplyJson(w, http.StatusOK, response)
	}

	return err.Error("Not Found")
}

func ite[T any](value *T, alternative T) T {
	if value == nil {
		return alternative
	}
	return *value
}
