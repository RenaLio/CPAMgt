package v1

type CpaAccountServiceConfig struct {
	CpaUrl    string `json:"cpaUrl" binding:"required"`
	CpaToken  string `json:"cpaToken" binding:"required"`
	CpaEnable bool   `json:"enable"`
}
