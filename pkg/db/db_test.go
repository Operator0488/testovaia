package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDBDefault(t *testing.T) {
	m, err := NewPostgresManager(context.Background(), Config{})
	man := m.(*manager)
	dbInput := &gorm.DB{Config: &gorm.Config{NowFunc: func() time.Time { return time.Now() }}}
	man.db = dbInput
	require.NoError(t, err)
	ctx := context.Background()
	dbOutput := m.DB(ctx)
	assert.Equal(t, dbInput, dbOutput)
}

func TestDBinTransaction(t *testing.T) {
	m, err := NewPostgresManager(context.Background(), Config{})
	man := m.(*manager)
	dbInput := &gorm.DB{Config: &gorm.Config{NowFunc: func() time.Time { return time.Now() }}}
	man.db = dbInput
	require.NoError(t, err)

	ctx := context.Background()
	dbTransaction := &gorm.DB{Config: &gorm.Config{NowFunc: func() time.Time { return time.Now() }}}
	ctx = withTransactionContext(ctx, dbTransaction)
	dbOutut := m.DB(ctx)

	assert.NotEqual(t, dbInput, dbOutut)
	assert.Equal(t, dbTransaction, dbOutut)
}
