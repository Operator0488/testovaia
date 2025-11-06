package di

import (
	"context"
	"fmt"
	"testing"

	clickdb "git.vepay.dev/knoknok/backend-platform/internal/pkg/mock/db/click/db"
	pgdb "git.vepay.dev/knoknok/backend-platform/internal/pkg/mock/db/pg/db"
	usrv "git.vepay.dev/knoknok/backend-platform/internal/pkg/mock/services/userservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	Register[pgdb.IUserRepository](ctx, &pgdb.UserRepository{User: "test"})

	err := testContainer.Build()
	require.NoError(t, err)

	myService := Resolve[pgdb.IUserRepository](ctx)
	require.NotNil(t, myService)

	assert.Equal(t, "test", myService.GetProfile())
}

func TestRegisterInterface(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)
	var repo pgdb.IUserRepository = &pgdb.UserRepository{User: "test from interface"}
	Register(ctx, repo)

	err := testContainer.Build()
	require.NoError(t, err)

	myService := Resolve[pgdb.IUserRepository](ctx)
	require.NotNil(t, myService)

	assert.Equal(t, "test from interface", myService.GetProfile())
}

func TestRegisterStructure(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)
	var repo = &pgdb.UserRepository{User: "test from interface"}
	Register(ctx, repo)

	err := testContainer.Build()
	require.NoError(t, err)

	myService := Resolve[*pgdb.UserRepository](ctx)
	require.NotNil(t, myService)

	assert.Equal(t, "test from interface", myService.GetProfile())
}

func TestRegisterFactory(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	RegisterFactory(ctx, func() pgdb.IUserRepository {
		return &pgdb.UserRepository{User: "test-factory"}
	})

	err := testContainer.Build()
	require.NoError(t, err)

	myService := Resolve[pgdb.IUserRepository](ctx)
	require.NotNil(t, myService)

	assert.Equal(t, "test-factory", myService.GetProfile())
}

func TestRegisterWithSameName(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	Register[pgdb.IUserRepository](ctx, &pgdb.UserRepository{User: "pg-user"})

	Register[clickdb.IUserRepository](ctx, &clickdb.UserRepository{User: "click-user"})

	err := testContainer.Build()
	require.NoError(t, err)

	pgRepo := Resolve[pgdb.IUserRepository](ctx)
	require.NotNil(t, pgRepo)
	assert.Equal(t, "pg-user", pgRepo.GetProfile())

	clickRepo := Resolve[clickdb.IUserRepository](ctx)
	require.NotNil(t, clickRepo)
	assert.Equal(t, "click-user", clickRepo.GetProfile())
}

func TestRegisterWithInjectSuccess(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	Register[clickdb.IUserRepository](ctx, &clickdb.UserRepository{})
	Register[usrv.IUserService](ctx, &usrv.UserService{})

	err := testContainer.Build()
	require.NoError(t, err)

	userService := Resolve[usrv.IUserService](ctx)
	require.NotNil(t, userService)
	userService.SetProfile("some-test")

	assert.Equal(t, "some-test", userService.GetProfile())
}

func TestResolveBeforeBuild(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	Register[clickdb.IUserRepository](ctx, &clickdb.UserRepository{})
	Register[usrv.IUserService](ctx, &usrv.UserService{})

	userService := Resolve[usrv.IUserService](ctx)
	require.NotNil(t, userService)
	userService.SetProfile("some-test")

	assert.Equal(t, "some-test", userService.GetProfile())
}

func TestResolveBeforeBuildNoDeps(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	Register[usrv.IUserService](ctx, &usrv.UserService{})

	_ = Resolve[usrv.IUserService](ctx)

	err := testContainer.Build()
	require.ErrorIs(t, err, ErrUnregisteredType, err)
}

func TestRegisterWithMultipleInjectSuccess(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	Register[iSuperServiceRepository](ctx, &superServiceRepository{})
	Register[clickdb.IUserRepository](ctx, &clickdb.UserRepository{})
	Register[pgdb.IUserRepository](ctx, &pgdb.UserRepository{})

	err := testContainer.Build()
	require.NoError(t, err)

	srv := Resolve[iSuperServiceRepository](ctx)
	require.NotNil(t, srv)
	srv.SetPgProfile("pg-test")
	srv.SetClickProfile("click-test")

	assert.Equal(t, "pg-test", srv.GetPgProfile())
	assert.Equal(t, "click-test", srv.GetClickProfile())
}

func TestRegisterNotFoundInjectErr(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	Register[pgdb.IUserRepository](ctx, &clickdb.UserRepository{})
	Register[usrv.IUserService](ctx, &usrv.UserService{})

	err := testContainer.Build()
	require.ErrorIs(t, err, ErrUnregisteredType, err)
}

func TestRegisterMustBePointerErr(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	requirePanicsWithMessage(t, "instance must be a pointer", func() {
		Register[clickdb.IUserRepository](ctx, nonPointerUserRepository{})
	})
}

func TestRegisterFactoryMustBePointerErr(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	requirePanicsWithMessage(t, "instance must be a pointer", func() {
		RegisterFactory(ctx, func() clickdb.IUserRepository { return nonPointerUserRepository{} })
	})
}

func TestRegisterTwiceErr(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)
	Register[pgdb.IUserRepository](ctx, &pgdb.UserRepository{})

	requirePanicsWithMessage(t, "already registered in services", func() {
		Register[pgdb.IUserRepository](ctx, &clickdb.UserRepository{})
	})
}

func TestRegisterFactoryTwiceErr(t *testing.T) {
	testContainer := New()
	ctx := WithContainer(context.Background(), testContainer)

	RegisterFactory(ctx, func() pgdb.IUserRepository {
		return &pgdb.UserRepository{}
	})

	requirePanicsWithMessage(t, "already registered in services", func() {
		Register[pgdb.IUserRepository](ctx, &pgdb.UserRepository{})
	})
}

type nonPointerUserRepository struct {
	User string
}

func (m nonPointerUserRepository) GetProfile() string {
	return m.User
}

func (m nonPointerUserRepository) SetProfile(name string) {

}

type iSuperServiceRepository interface {
	GetPgProfile() string
	SetPgProfile(name string)
	GetClickProfile() string
	SetClickProfile(name string)
}

type superServiceRepository struct {
	repoPg    pgdb.IUserRepository
	repoClick clickdb.IUserRepository
}

func (m *superServiceRepository) GetPgProfile() string {
	return m.repoPg.GetProfile()
}

func (m *superServiceRepository) SetPgProfile(name string) {
	m.repoPg.SetProfile(name)
}

func (m *superServiceRepository) GetClickProfile() string {
	return m.repoClick.GetProfile()
}

func (m *superServiceRepository) SetClickProfile(name string) {
	m.repoClick.SetProfile(name)
}

func (m *superServiceRepository) ResolveDeps(repo1 pgdb.IUserRepository, repo2 clickdb.IUserRepository) {
	m.repoPg = repo1
	m.repoClick = repo2
}

func requirePanicsWithMessage(t *testing.T, expectedSubstring string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("%v", r)
			require.Contains(t, errMsg, expectedSubstring,
				"panic message should contain expected substring")
		} else {
			t.Fatal("expected panic, but no panic occurred")
		}
	}()
	f()
}
