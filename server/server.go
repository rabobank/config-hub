package server

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gomatbase/go-log"
	"github.com/gomatbase/go-we"
	"github.com/gomatbase/go-we/security"
	"github.com/gomatbase/go-we/util"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/sources"
	"github.com/rabobank/config-hub/sources/credhub_source"
	"github.com/rabobank/config-hub/sources/git_source"
)

var (
	l, _            = log.GetWithOptions("MAIN", log.Standard().WithFailingCriticals().WithStartingLevel(cfg.LogLevel))
	propertySources []sources.Source
)

func Server() {
	if e := cfg.FinishServerConfiguration(); e != nil {
		l.Critical(e)
	}

	if e := setupSources(); e != nil {
		l.Critical(e)
	}

	l.Infof("OpenId Url: %s", cfg.OpenIdUrl)
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
		Path("/**").Authentication(bearerAuthenticationProvider).Authorize(security.Scope("config_server_" + cfg.ServiceInstanceId + ".read")).
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
	engine.HandleMethod("GET", "/dashboard", dashboard)

	// config-server compatible endpoints
	engine.HandleMethod("GET", "/{app}/{profiles}", findProperties)
	engine.HandleMethod("GET", "/{app}/{profiles}/{label}", findProperties)

	l.Critical(engine.Listen(":" + cfg.Port))
}

func localhost(_ *security.User, scope we.RequestScope) bool {
	// TODO check that it comes from the localhost
	parts := strings.Split(scope.Request().RemoteAddr, ":")
	return parts[0] == "127.0.0.1"
}

func findProperties(w we.ResponseWriter, scope we.RequestScope) error {
	app := scope.Var("app")
	profiles := strings.Split(scope.Var("profiles"), ",")
	label := scope.Var("label")

	l.Debugf("Received properties request for app: %s, profiles: %v and label: %s", app, profiles, label)

	var sources []*domain.PropertySource
	for _, source := range propertySources {
		if foundProperties, e := source.FindProperties(app, profiles, label); e != nil {
			l.Errorf("Error when calling source %v: %v", reflect.TypeOf(source).Name(), e)
		} else if foundProperties != nil {
			sources = append(sources, foundProperties...)
		}
	}

	if sources != nil {
		response := &domain.Configs{
			App:      app,
			Profiles: profiles,
			Sources:  sources,
		}
		util.ReplyJson(w, http.StatusOK, response)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	return nil
}

func setupSources() error {
	var e error
	propertySources = make([]sources.Source, len(cfg.Sources))
	fmt.Println(cfg.Sources)
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
