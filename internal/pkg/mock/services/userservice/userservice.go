package userservice

import (
	"context"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/mock/db/click/db"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
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
