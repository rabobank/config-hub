package domain

type HttpCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CredentialsRequest struct {
	Token    *string `json:"token,omitempty"`
	Protocol string  `json:"protocol"`
	Host     string  `json:"host"`
	Repo     string  `json:"repo"`
}
