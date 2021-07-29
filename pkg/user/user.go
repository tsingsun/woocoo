package user

type Identity interface {
	FindIdentity(id string) User
	FindIdentityByToken(token string) User
	ID() string
}

type User map[string]interface{}

func (u User) ID() string {
	return u["ID"].(string)
}

func (u User) OrgID() string {
	return u["orgID"].(string)
}
