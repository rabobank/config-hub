package domain

type Configs struct {
	App      string           `json:"name"`
	Profiles []string         `json:"profiles"`
	Label    *string          `json:"label"`
	Version  *string          `json:"version"`
	State    *string          `json:"state"`
	Sources  []PropertySource `json:"propertySources"`
}

type PropertySource struct {
	Source     string                 `json:"name"`
	Properties map[string]interface{} `json:"source"`
}
