package userservice

import (
	"context"

	"git.vepay.dev/knoknok/backend-platform/internal/pkg/mock/db/click/db"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
)

type IUserService interface {
	GetProfile() string
	SetProfile(string)
}

type UserService struct {
	repo db.IUserRepository
}

func (m *UserService) GetProfile() string {
	return m.repo.GetProfile()
}

func (m *UserService) SetProfile(name string) {
	m.repo.SetProfile(name)
}

func (m *UserService) ResolveDeps(repo db.IUserRepository) {
	m.repo = repo
	logger.Info(context.TODO(), "ResolveDeps", logger.Any("repo", repo))
}
