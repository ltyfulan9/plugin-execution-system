package validator

type CreateExecutionRequest struct {
	PluginIDs []string       `json:"plugin_ids"`
	Input     map[string]any `json:"input"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func ValidateCreateExecutionRequest(req CreateExecutionRequest) bool {
	return len(req.PluginIDs) > 0 && req.Input != nil
}

func ValidateLoginRequest(req LoginRequest) bool {
	return req.Token != "" || (req.Username != "" && req.Password != "")
}
