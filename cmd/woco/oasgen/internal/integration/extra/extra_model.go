package extra

type ApiResponse struct {
	Code    int32  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
}
