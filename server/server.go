package server

import (
	"net/http"
	"strings"

	"github.com/gomatbase/go-log"
	"github.com/gomatbase/go-we"
	"github.com/gomatbase/go-we/events"
	"github.com/gomatbase/go-we/security"
	"github.com/gomatbase/go-we/util"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources"
	"github.com/rabobank/config-hub/sources/credhub_source"
	"github.com/rabobank/config-hub/sources/git_source"
	"gopkg.in/yaml.v3"
)

var (
	l, _ = log.GetWithOptions("MAIN", log.Standard().WithFailingCriticals().WithStartingLevel(cfg.LogLevel))
)

func Server() {
	if e := cfg.FinishServerConfiguration(); e != nil {
		l.Critical(e)
	}

	if e := sources.Setup(); e != nil {
		l.Critical(e)
	}

	l.Infof("OpenId Url: %s\n", cfg.OpenIdUrl)
	openIdProvider := security.OpenIdIdentityProvider(cfg.OpenIdUrl).
		Client(cfg.Client, cfg.Secret).Scope("cloud_controller.read", "openid").
		UserEnrichment(enrichUaaUser).
		Build()
	ssoAuthenticationProvider := security.SSOAuthenticationProvider().DefaultAuthenticatedEndpoint("/dashboard").
		AuthorizationCodeProvider(openIdProvider.AuthorizationCodeProvider()).Build()
	bearerAuthenticationProvider := security.BearerAuthenticationProvider().
		Introspector(openIdProvider.TokenIntrospector()).Build()

	allowedUsers := security.Either(security.Scope("cloud_controller.admin"), security.AuthorizationFunc(isDeveloper))

	securityFilter := security.Filter(true).
		Path("/health", "/info").Anonymous().
		Path("/credentials").Authorize(security.AuthorizationFunc(localhost)).
		Path("/secrets", "/secrets/add", "/secrets/delete", "/secrets/list").Authentication(bearerAuthenticationProvider).Authorize(allowedUsers).
		Path("/dashboard").Authentication(ssoAuthenticationProvider).Authorize(allowedUsers).
		Path("/**").Authentication(bearerAuthenticationProvider).Authorize(security.Scope("config_hub_" + cfg.ServiceInstanceId + ".read")).
		Build()

	engine := we.New()
	engine.AddFilter(securityFilter)

	// git credentials helper
	engine.HandleMethod("POST", "/credentials", git_source.ServeCredentials)

	// credentials management endpoints
	engine.HandleMethod("POST", "/secrets/add", credhub_source.AddSecrets)
	engine.HandleMethod("POST", "/secrets", credhub_source.AddSecrets)
	engine.HandleMethod("DELETE", "/secrets/delete", credhub_source.DeleteSecrets)
	engine.HandleMethod("DELETE", "/secrets", credhub_source.DeleteSecrets)
	engine.HandleMethod("GET", "/secrets/list", credhub_source.ListSecretsCompatible)
	engine.HandleMethod("GET", "/secrets", credhub_source.ListSecrets)

	// dashboard
	engine.HandleMethod("GET", "/dashboard", sources.Dashboard)

	// config-server compatible endpoints
	engine.HandleMethod("GET", "/{app}/{profiles}", findProperties) // will also take care of /{label}/{app}-{profiles}.(json|properties|yml|yaml)
	engine.HandleMethod("GET", "/{app}/{profiles}/{label}", findProperties)

	// config-server alternative format endpoints
	engine.HandleMethod("GET", "/{appProfiles}", findFormattedProperties)

	l.Critical(engine.Listen(":" + cfg.Port))
}

func localhost(_ *security.User, scope we.RequestScope) bool {
	parts := strings.Split(scope.Request().RemoteAddr, ":")
	return parts[0] == "127.0.0.1"
}

func findProperties(w we.ResponseWriter, scope we.RequestScope) error {
	app := scope.Var("app")
	var profiles []string

	// revert order of profiles returned to follow config-server logic... so... first priority requested should be last one served
	for _, profile := range strings.Split(scope.Var("profiles"), ",") {
		profiles = append([]string{profile}, profiles...)
	}

	label := scope.Var("label")
	label = strings.ReplaceAll(label, "(_)", "/")

	l.Debugf("Received properties request for app: %s, profiles: %v and label: %s", app, profiles, label)
	if properties := sources.FindProperties(app, profiles, label); properties != nil {
		response := &domain.Configs{
			App:      app,
			Profiles: strings.Split(scope.Var("profiles"), ","),
			Sources:  properties,
			Label:    &label,
		}
		if e := util.ReplyJson(w, http.StatusOK, response); e != nil {
			l.Errorf("Error when replying to properties request: %v", e)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	return nil
}

func findFormattedProperties(w we.ResponseWriter, scope we.RequestScope) error {
	app := scope.Var("appProfiles")
	var suffix string
	if dotIndex := strings.LastIndex(app, "."); dotIndex == -1 {
		// not a format specific request and not picked by other handlers... not found
		return events.NotFoundError
	} else {
		suffix = app[dotIndex+1:]
		app = app[:dotIndex]
	}

	var replyFunction func(we.ResponseWriter, int, any) error
	switch suffix {
	case "json":
		replyFunction = ReplyJson
	case "yml", "yaml":
		replyFunction = ReplyYaml
	case "properties":
		replyFunction = ReplyProperties
	default:
		return events.BadRequestError
	}

	// split the app between app and profile(s). for now let's just support simple profiles
	commaIndex := strings.Index(app, ",")
	if commaIndex == -1 {
		// expect single profile
		commaIndex = len(app)
	}
	if dashIndex := strings.LastIndex(app[:commaIndex], "-"); dashIndex == -1 {
		return events.BadRequestError
	} else {
		profiles := app[dashIndex+1:]
		app = app[:dashIndex]
		if properties := sources.FindPropertiesMap(app, strings.Split(profiles, ","), ""); properties != nil {
			if e := replyFunction(w, http.StatusOK, properties); e != nil {
				l.Errorf("Error when replying in %s to properties request: %v", suffix, e)
			}
		} else {
			return events.NotFoundError
		}
	}
	return nil
}

func ReplyJson(w we.ResponseWriter, status int, value any) (e error) {
	return util.ReplyJson(w, http.StatusOK, value)
}

func ReplyYaml(w we.ResponseWriter, status int, value any) (e error) {
	var body []byte
	if body, e = yaml.Marshal(value); e == nil {
		w.Header().Add("Content-type", "application/yaml")
		w.WriteHeader(status)
		if _, e = w.Write(body); e != nil {
			l.Errorf("Failure to write yaml response through reponse writer: %v", e)
		}
	}

	return e
}

func ReplyProperties(w we.ResponseWriter, status int, value any) (e error) {
	var body []byte
	if body, e = yaml.Marshal(value); e == nil {
		w.Header().Add("Content-type", "text/plain")
		w.WriteHeader(status)
		if _, e = w.Write(body); e != nil {
			l.Errorf("Failure to write yaml response through reponse writer: %v", e)
		}
	}

	return e
}
