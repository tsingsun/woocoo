package polarismesh

// dialOptions
type dialOptions struct {
	Namespace   string            `json:"Namespace"`
	DstMetadata map[string]string `json:"dst_metadata"`
	SrcMetadata map[string]string `json:"src_metadata"`
	SrcService  string            `json:"src_service"`
	// 可选，规则路由Meta匹配前缀，用于过滤作为路由规则的gRPC Header
	HeaderPrefix []string `json:"header_prefix"`
}
