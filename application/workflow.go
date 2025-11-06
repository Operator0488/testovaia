package application

import (
	"context"
	"errors"
	"git.vepay.dev/knoknok/backend-platform/pkg/di"
	"git.vepay.dev/knoknok/backend-platform/pkg/workflow"
)

var (
	workflowComponent = NewComponent("workflow", initWorkflow, runWorkflow)
)

func WithWorkflow() Option {
	return func(app *Application) error {
		app.components.add(component(workflowComponent))
		return nil
	}
}

func initWorkflow(_ context.Context, app *Application) error {
	cfg := app.config.getWorkflowConfig()
	app.Workflow = workflow.NewWorkflowBuilder(
		cfg.ServiceName,
		cfg.Host,
		cfg.User,
		cfg.Password)

	return nil
}

func runWorkflow(ctx context.Context, app *Application) error {
	if app.Workflow == nil {
		return errors.New("workflow is not initialized")
	}

	app.workflow = app.Workflow.Run(ctx)
	app.Health.Add("workflow", app.workflow.Health)
	app.Closer.Add(func() error {
		return app.workflow.Close(ctx)
	})

	di.Register(ctx, app.workflow)
	return nil
}
