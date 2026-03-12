package db

type IUserRepository interface {
	GetProfile() string
	SetProfile(string)
}

type UserRepository struct {
	User string
}

func (m *UserRepository) GetProfile() string {
	return m.User
}

func (m *UserRepository) SetProfile(name string) {
	m.User = name
}
