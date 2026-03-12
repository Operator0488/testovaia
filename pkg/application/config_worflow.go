package application

const (
	envWorkflowHost     = "workflow.host"
	envWorkflowUser     = "workflow.user"
	envWorkflowPassword = "workflow.password"
)

type workflowConfig struct {
	ServiceName string
	Host        string
	User        string
	Password    string
}

func (a *appConfig) getWorkflowConfig() workflowConfig {
	return workflowConfig{
		ServiceName: a.GetAppName(),
		Host:        a.GetString(envWorkflowHost),
		User:        a.GetString(envWorkflowUser),
		Password:    a.GetString(envWorkflowPassword),
	}
}
