package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"ariga.io/atlas/atlasexec"
	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/logger"
)

func (m *manager) Migrate(ctx context.Context) (int, error) {
	var lastErr error
	var applied int

	for i := 0; i < m.config.migrateRetries; i++ {
		applied, err := m.runMigration(ctx)
		if err != nil {
			lastErr = err
			logger.Error(ctx,
				fmt.Sprintf("Migration attempt %d/%d failed", i+1, m.config.migrateRetries),
				logger.Err(err),
			)

			if i < m.config.migrateRetries-1 {
				time.Sleep(time.Second * time.Duration(2*(i+1)))
				continue
			}
		}
		return applied, err
	}

	return applied, fmt.Errorf("all migration attempts failed, last error: %v", lastErr)
}

func (m *manager) runMigration(ctx context.Context) (int, error) {
	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(
			os.DirFS("./migrations"),
		),
	)
	if err != nil {
		return 0, err
	}
	defer workdir.Close()

	client, err := atlasexec.NewClient(workdir.Path(), "atlas")
	if err != nil {
		return 0, err
	}

	dsn := m.buildDSN()

	res, err := client.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
		URL:        dsn,
		ExecOrder:  atlasexec.ExecOrderNonLinear, // порядок не линейный
		AllowDirty: true,                         // Yandex Managed PostgreSQL создаёт служебную схему metric_helpers, которая ломает проверку чистоты БД
	})
	if err != nil {
		return 0, err
	}
	return len(res.Applied), nil
}
