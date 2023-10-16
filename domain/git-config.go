package domain

type ConfigServerConfiguration struct {
	EncryptKey *string      `json:"encrypt.key,omitempty"`
	Composite  []*GitConfig `json:"composite,omitempty"`
}

type GitConfig struct {
	Type                   *string  `json:"type,omitempty"`
	Uri                    string   `json:"uri"`
	DefaultLabel           *string  `json:"defaultLabel,omitempty"`
	BaseDir                string   `json:"basedir"`
	SearchPaths            []string `json:"searchPaths,omitempty"`
	Username               *string  `json:"username,omitempty"`
	Password               *string  `json:"password,omitempty"`
	PrivateKey             *string  `json:"privateKey,omitempty"`
	RefreshRate            *int     `json:"refreshRate,omitempty"`
	SkipSslValidation      bool     `json:"skipSslValidation"`
	IgnoreLocalSshSettings *bool    `json:"ignoreLocalSshSettings,omitempty"`
}
