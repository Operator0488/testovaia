package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"git.vepay.dev/knoknok/backend-platform/pkg/logger"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/google/uuid"
)

// NewWorkflowBuilder Создает новый экземпляр для подготовки и конфигурации клиента Zeebe Workflow
func NewWorkflowBuilder(serviceName, host, user, password string) WorkflowBuilder {
	return &workflowBuilder{
		serviceName:    serviceName,
		host:           host,
		credentialUser: user,
		credentialPwd:  password,
	}
}

// Workflow Интерфейс для взаимодействия с workflow zeebe
type Workflow interface {
	// StartProcess Запускает новый процесс в workflow
	//	processId уникальный идентификатор образа процесса. Задается в BPMN как атрибут процесса  (ProcessName)
	//	variables переменные, которые необходимо добавить в скоуп процесса при старте
	StartProcess(ctx context.Context, processName string, variables map[string]interface{}) (*WorkflowInstance, error)
	// SendEvent Отправляет событие в активный workflow.
	//	messageKey именованное событие в схеме workflow
	//	correlationKey ключ корреляции, связанный с событием, для идентификации нужного экземпляра workflow
	//	variables переменные, которые при необходимости нужно добавить в скоуп workflow
	SendEvent(ctx context.Context, messageKey string, correlationKey string, variables map[string]interface{}) error
	// Health проба компонента
	Health(ctx context.Context) error
	// Close Завершает работу компонента
	Close(ctx context.Context) error
}

// WorkflowInstance Информация о экземпляре процесса
type WorkflowInstance struct {
	// ProcessName Наименование процесса
	ProcessName string
	// ProcessVersion Версия процесса
	ProcessVersion int32
	// ProcessInstanceId Уникальный идентификатор экземпляра процесса
	ProcessInstanceId int64
}

type workflowService struct {
	serviceName string
	client      zbc.Client
	taskWorkers []worker.JobWorker
	readyToWork atomic.Bool
}

func (svc *workflowService) StartProcess(
	ctx context.Context,
	processName string,
	variables map[string]interface{}) (*WorkflowInstance, error) {

	if !svc.readyToWork.Load() {
		return nil, ErrWorkflowIsNotReady
	}

	command, commandErr := svc.client.NewCreateInstanceCommand().
		BPMNProcessId(processName).
		LatestVersion().
		VariablesFromMap(variables)

	if commandErr != nil {
		return nil, commandErr
	}

	process, processErr := command.Send(ctx)
	if processErr != nil {
		return nil, processErr
	}

	return &WorkflowInstance{
		ProcessName:       process.BpmnProcessId,
		ProcessVersion:    process.Version,
		ProcessInstanceId: process.ProcessInstanceKey,
	}, nil
}

func (svc *workflowService) SendEvent(
	ctx context.Context,
	messageKey string,
	correlationKey string,
	variables map[string]interface{}) error {

	if !svc.readyToWork.Load() {
		return ErrWorkflowIsNotReady
	}

	ctx = logger.With(ctx,
		logger.String("messageKey", messageKey),
		logger.Int("eventTtlHrs", workflowEventsTTLHours),
		logger.Any("variables", variables))

	command, commandErr := svc.client.NewPublishMessageCommand().
		MessageName(messageKey).
		CorrelationKey(correlationKey).
		TimeToLive(workflowEventsTTLHours * time.Hour).
		VariablesFromMap(variables)

	if commandErr != nil {
		logger.Error(ctx, "Workflow publish command error", logger.Err(commandErr))
		return commandErr
	}

	_, sendErr := command.Send(ctx)
	if sendErr != nil {
		logger.Error(ctx, "Workflow send event error", logger.Err(sendErr))
		return sendErr
	}

	logger.Debug(ctx, "Workflow event has been sent")
	return nil
}

func (svc *workflowService) Health(ctx context.Context) error {
	//	TODO
	return nil
}

func (svc *workflowService) Close(ctx context.Context) error {
	if !svc.readyToWork.Load() {
		return nil
	}

	for _, taskWorker := range svc.taskWorkers {
		taskWorker.Close()
	}

	closeErr := svc.client.Close()
	if closeErr != nil {
		logger.Error(ctx, "Failed to close workflow client", logger.Err(closeErr))
	}

	return closeErr
}

func (svc *workflowService) SetClient(client zbc.Client) {
	svc.client = client
}

func (svc *workflowService) ReadyToWork() {
	svc.readyToWork.Store(true)
}

func (svc *workflowService) DeployBpmn(ctx context.Context, bpmnPath string) {
	command := svc.client.NewDeployResourceCommand().AddResourceFile(bpmnPath)
	deployment, err := command.Send(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to deploy workflow bpmn",
			logger.String("bpmnPath", bpmnPath),
			logger.Err(err))

		panic(err)
	}

	logger.Info(ctx, "Deployment workflow bpmn has been success",
		logger.String("bpmnPath", bpmnPath),
		logger.Int64("deploymentKey", deployment.Key))
}

func (svc *workflowService) Subscribe(
	taskName string,
	taskHandler WorkflowTaskHandler,
	taskHandlerConfig TaskHandlerConfig) {

	// Обертка над хэндлером чтобы получить доступ к скоупу данных
	taskHandlerWrapper := func(client worker.JobClient, job entities.Job) {
		ctx := context.WithValue(context.Background(), logger.CorrelationId, uuid.New().String())
		ctx = logger.With(ctx,
			logger.String("taskName", taskName),
			logger.String("processId", job.BpmnProcessId),
			logger.Int64("taskId", job.Key),
			logger.Int64("instanceId", job.ProcessDefinitionKey),
			logger.Int32("retries", job.Retries))

		defer func() {
			if r := recover(); r != nil {
				recoveryErrData := fmt.Errorf("%v", r)
				logger.Error(ctx, "Workflow task handler thrown panic", logger.Err(recoveryErrData))

				// Постановка процесса на инцидент при любой панике
				go svc.handleIncidentTask(ctx, client, job, recoveryErrData)
			}
		}()

		vars, err := job.GetVariablesAsMap()
		if err != nil {
			logger.Error(ctx, "Workflow failed to get variables for task", logger.Err(err))
			return
		}

		logger.Info(ctx, "Start processing workflow task")
		taskErr := taskHandler(ctx, &workflowTask{
			TaskId:    job.Key,
			TaskName:  taskName,
			ProcessId: job.ProcessDefinitionKey,
			Retries:   job.Retries,
			Scope:     vars,
		})
		if taskErr != nil {
			logger.Error(ctx, "Workflow task failed", logger.Err(taskErr))

			retries := job.Retries + 1
			incident := errors.Is(taskErr, ErrWorkflowIncident) || retries >= taskHandlerConfig.IncidentMaxRetries
			if incident {
				svc.handleIncidentTask(ctx, client, job, taskErr)
			} else {
				svc.handleErrorTask(ctx, client, job, taskHandlerConfig)
			}

			return
		}

		svc.completeTask(ctx, client, job)
	}

	jobWorker := svc.client.NewJobWorker().
		JobType(taskName).
		Handler(taskHandlerWrapper).
		Concurrency(taskHandlerConfig.Concurrency).
		MaxJobsActive(taskHandlerConfig.MaxActiveTasks).
		RequestTimeout(requestTimeoutSec * time.Second).
		PollInterval(taskHandlerConfig.PoolIntervalSec * time.Second).
		Name(svc.serviceName).
		Open()

	svc.taskWorkers = append(svc.taskWorkers, jobWorker)
	logger.Info(context.Background(), "Workflow worker has been started", logger.String("taskName", taskName))
}

func (svc *workflowService) completeTask(ctx context.Context, client worker.JobClient, job entities.Job) {
	var cancelFn context.CancelFunc
	ctx, cancelFn = context.WithTimeout(ctx, cancelOperationTimeoutInSec*time.Second)
	defer cancelFn()

	_, err := client.NewCompleteJobCommand().JobKey(job.Key).Send(ctx)
	if err != nil {
		logger.Error(ctx,
			"failed to send complete job message to camunda. task will be full retry after lock timeout",
			logger.Err(err))

		return
	}

	logger.Info(ctx, "completed send success job info")
}

func (svc *workflowService) handleErrorTask(
	ctx context.Context,
	client worker.JobClient,
	job entities.Job,
	taskHandlerConfig TaskHandlerConfig) {

	var cancelFn context.CancelFunc
	ctx, cancelFn = context.WithTimeout(ctx, cancelOperationTimeoutInSec*time.Second)
	defer cancelFn()

	_, err := client.
		NewFailJobCommand().
		JobKey(job.Key).
		Retries(job.Retries + 1).
		RetryBackoff(taskHandlerConfig.RetryTimeoutSec * time.Second).
		Send(ctx)

	if err != nil {
		logger.Error(ctx, "Failed to send fail job message. task will be retry after timeout", logger.Err(err))
		return
	}

	logger.Info(ctx, "Completed send fail job info")
}

func (svc *workflowService) handleIncidentTask(
	ctx context.Context,
	client worker.JobClient,
	job entities.Job,
	errData error) {

	var cancelFn context.CancelFunc
	ctx, cancelFn = context.WithTimeout(ctx, cancelOperationTimeoutInSec*time.Second)
	defer cancelFn()

	_, err := client.
		NewFailJobCommand().
		JobKey(job.Key).
		Retries(0).
		ErrorMessage(errData.Error()).
		Send(ctx)

	if err != nil {
		logger.Error(ctx, "Failed to send incident job message. task will be retry after timeout", logger.Err(err))
		return
	}

	logger.Info(ctx, "Completed send incident job data")
}
