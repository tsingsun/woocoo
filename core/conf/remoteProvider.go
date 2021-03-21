package conf

//remote configuration provider
type remoteProvider struct {
	provider      string
	endpoint      string
	path          string
	secretKeyring string
}

func (rp remoteProvider) Provider() string {
	return rp.provider
}

func (rp remoteProvider) Endpoint() string {
	return rp.endpoint
}

func (rp remoteProvider) Path() string {
	return rp.path
}

func (rp remoteProvider) SecretKeyring() string {
	return rp.secretKeyring
}
