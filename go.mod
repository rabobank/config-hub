module github.com/rabobank/config-hub

go 1.23.0

toolchain go1.23.2

replace (
	golang.org/x/net => golang.org/x/net v0.39.0
	google.golang.org/protobuf => google.golang.org/protobuf v1.36.6
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.0
	github.com/gomatbase/go-error v1.1.0
	github.com/gomatbase/go-log v1.1.0
	github.com/gomatbase/go-we v1.0.0-b9
	github.com/rabobank/credhub-client v0.0.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/cloudfoundry-community/go-uaa v0.3.5 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/oauth2 v0.29.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
