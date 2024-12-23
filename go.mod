module github.com/rabobank/config-hub

go 1.23

replace (
	golang.org/x/net => golang.org/x/net v0.33.0
	google.golang.org/protobuf => google.golang.org/protobuf v1.36.0
)

require (
	github.com/gomatbase/go-error v1.1.0
	github.com/gomatbase/go-log v1.1.0
	github.com/gomatbase/go-we v1.0.0-b9
	github.com/rabobank/credhub-client v0.0.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cloudfoundry-community/go-uaa v0.3.3 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	google.golang.org/protobuf v1.36.0 // indirect
)
