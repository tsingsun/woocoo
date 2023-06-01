package gds

// KeyValue is a key-value string pair. You can use expvar.KeyValue in advanced scenarios.
type KeyValue struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}
