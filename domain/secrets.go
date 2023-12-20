package domain

type SecretName struct {
	App     string `json:"app"`
	Profile string `json:"profile"`
	Label   string `json:"label"`
	Name    string `json:"name"`
}
