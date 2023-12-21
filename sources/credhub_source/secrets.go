package credhub_source

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gomatbase/go-we"
	"github.com/gomatbase/go-we/util"
	"github.com/rabobank/config-hub/domain"
)

func getParameters(r we.RequestScope) (apps, profiles, labels []string) {
	apps = fromListParameter(r.Parameter("apps"))
	profiles = fromListParameter(r.Parameter("profiles"))
	labels = fromListParameter(r.Parameter("labels"))

	return
}

func fromListParameter(parameter string) []string {
	if len(parameter) == 0 {
		return nil
	}
	return strings.Split(parameter, ",")
}

func ListSecretsCompatible(w we.ResponseWriter, r we.RequestScope) error {
	apps, profiles, labels := getParameters(r)
	if secretNames, e := defaultSource.listSecrets(apps, profiles, labels); e != nil {
		return e
	} else {
		// convert to old config-server format
		var convertedNames []domain.SecretName
		for app, profileSecrets := range secretNames {
			for profile, labelSecrets := range profileSecrets {
				for label, names := range labelSecrets {
					for _, name := range names {
						convertedNames = append(convertedNames, domain.SecretName{App: app, Profile: profile, Label: label, Name: name})
					}
				}
			}
		}
		return util.ReplyJson(w, http.StatusOK, convertedNames)
	}
}

func ListSecrets(w we.ResponseWriter, r we.RequestScope) error {
	apps, profiles, labels := getParameters(r)
	if secretNames, e := defaultSource.listSecrets(apps, profiles, labels); e != nil {
		return e
	} else {
		return util.ReplyJson(w, http.StatusOK, secretNames)
	}
}

func AddSecrets(w we.ResponseWriter, r we.RequestScope) error {
	apps, profiles, labels := getParameters(r)
	fmt.Println("adding secrets for", apps, profiles, labels)
	if secrets, e := util.ReadJsonBody[map[string]any](r); e != nil {
		fmt.Println("error reading json body", e)
		return e
	} else if e = defaultSource.addSecrets(apps, profiles, labels, *secrets); e != nil {
		return e
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
	return nil
}

func DeleteSecrets(w we.ResponseWriter, r we.RequestScope) error {
	return nil
}
