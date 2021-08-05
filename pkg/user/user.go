package user

const (
	OrgIdHeader = "X-Org-Id"
	IDKey       = "id"
	OrgIDKey    = "orgID"
)

type Identity interface {
	FindIdentity(id string) Identity
	FindIdentityByToken(token string) Identity
	ID() string
}

type ContextUserTag struct{}
type User map[string]interface{}

func (u User) ID() string {
	return u[IDKey].(string)
}

func (u User) OrgID() string {
	return u[OrgIDKey].(string)
}

func (u *User) FindIdentity(id string) Identity {
	return nil
}

func (u *User) FindIdentityByToken(token string) Identity {
	return nil
}
