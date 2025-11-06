package workflow

import (
	"context"
	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"os/signal"
	"syscall"
	"time"
)

//
// WorkflowBuilder Конфигуратор клиента workflow
type WorkflowBuilder interface {
	//
	// Run Запускает обслуживание workflow после настройки. Возвращает новый экземпляр Workflow
	Run(ctx context.Context) Workflow
	//
	// WithProcess Регистрирует BPMN процесс в workflow из указанного файла
	WithProcess(bpmnPath string) WorkflowBuilder
	//
	// WithHandler Регистрирует обработчик задач workflow.
	// taskName: уникальное имя задачи в workflow.
	// handler: функция-обработчик задачи.
	// config: необязательный аргумент с тонкой конфигурацией
	WithHandler(taskName string, handler WorkflowTaskHandler, config ...TaskHandlerConfig) WorkflowBuilder
}

//
// TaskHandlerConfig Тонкий конфигуратор конкретного обработчика задач
type TaskHandlerConfig struct {
	// RetryTimeoutSec Таймаут перед повторной попыткой делегировать задачу в случае ошибки.
	// Если не указано явно, используется значение по умолчанию 60 секунд
	RetryTimeoutSec time.Duration
	// IncidentMaxRetries Количество попыток выполнения задачи прежде чем процесс становится в
	// состояние "инцидент". Если не указано явно, используется значение по умолчанию 3 попытки
	IncidentMaxRetries int32
	// MaxActiveTasks Максимальное количество одновременно выполняемых задач, которые могут
	// быть активированы для этого обработчикоа. Если не задано явно, значение по умолчанию 10
	MaxActiveTasks int
	// Concurrency Максимальное количество одновременно создаваемых горутин для выполнения
	// задач этого обработчика. Если не задано явно, значение по умолчанию 3
	Concurrency int
	// PoolIntervalSec Интервал запроса новых задач. Если не указано явно, значение по умолчанию 1 секунда
	PoolIntervalSec time.Duration
}

type handlerDescriptor struct {
	Handler WorkflowTaskHandler
	Config  TaskHandlerConfig
}

type workflowBuilder struct {
	serviceName     string
	host            string
	credentialUser  string
	credentialPwd   string
	deploymentProcs []string
	handlers        map[string]handlerDescriptor
}

func (svc *workflowBuilder) Run(ctx context.Context) Workflow {
	result := &workflowService{
		serviceName: svc.serviceName,
	}

	go svc.runWorkflow(result)
	return result
}

func (svc *workflowBuilder) WithProcess(bpmnPath string) WorkflowBuilder {
	svc.deploymentProcs = append(svc.deploymentProcs, bpmnPath)
	return svc
}

func (svc *workflowBuilder) WithHandler(
	taskName string,
	handler WorkflowTaskHandler,
	config ...TaskHandlerConfig) WorkflowBuilder {

	var handleConfig TaskHandlerConfig
	if len(config) > 0 {
		handleConfig = config[0]
	}

	fillDefaults(&handleConfig)
	svc.handlers[taskName] = handlerDescriptor{
		Handler: handler,
		Config:  handleConfig,
	}
	return svc
}

func (svc *workflowBuilder) runWorkflow(workflow *workflowService) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info(ctx, "Starting workflow client")
	for {
		coreStarted := false
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Workflow builder has received a signal to break init")
			return
		default:
			client, err := zbc.NewClient(&zbc.ClientConfig{
				GatewayAddress:         svc.host,
				UsePlaintextConnection: true,
			})
			if err != nil {
				logger.Error(ctx, "Failed to create zeebe client", logger.Err(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(initializeErrBackoff):
					continue
				}
			}

			workflow.SetClient(client)
			coreStarted = true
		}

		if coreStarted {
			break
		}
	}

	if len(svc.deploymentProcs) > 0 {
		logger.Info(ctx, "Deploying BPMN process")
		for _, processPath := range svc.deploymentProcs {
			select {
			case <-ctx.Done():
				logger.Info(ctx, "Workflow builder has received a signal to break init on deployment procs")
				return
			default:
				workflow.DeployBpmn(ctx, processPath)
			}
		}
	}

	// В целом тут уже можно считать работу начатой. Запуск воркеров не влияет на возможность
	// запуска процессов
	workflow.ReadyToWork()
	logger.Info(ctx, "Workflow client is ready to work")

	if len(svc.handlers) == 0 {
		logger.Info(ctx, "Make task subscriber workers")
		for taskName, taskHandler := range svc.handlers {
			select {
			case <-ctx.Done():
				logger.Info(ctx, "Workflow builder has received a signal to break on subscribe task handlers")
				return
			default:
				workflow.Subscribe(taskName, taskHandler.Handler, taskHandler.Config)
			}
		}
	}

	logger.Info(ctx, "Workflow task handlers have been started")
}

func fillDefaults(config *TaskHandlerConfig) {
	if config.RetryTimeoutSec == 0 {
		config.RetryTimeoutSec = workflowRetryTimeoutSeconds
	}
	if config.IncidentMaxRetries == 0 {
		config.IncidentMaxRetries = maxRetryCountBeforeIncident
	}
	if config.MaxActiveTasks == 0 {
		config.MaxActiveTasks = maxActiveJobs
	}
	if config.Concurrency == 0 {
		config.Concurrency = tasksConcurrency
	}
	if config.PoolIntervalSec == 0 {
		config.PoolIntervalSec = poolIntervalSec
	}
}
