package domain

// Supported source types
const (
	GitSourceType     = "git"
	CredhubSourceType = "credhub"
)

type Configuration struct {
	UaaClient string `json:"uaa_client"`
	UaaSecret string `json:"uaa_secret"`
	Sources   string `json:"sources"`
}

type SourceConfig interface {
	Type() string
}
