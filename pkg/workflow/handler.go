package workflow

import "context"

// WorkflowTaskHandler Сигнатура обработчика задачи от Workflow
type WorkflowTaskHandler func(ctx context.Context, task WorkflowTask) error

// WorkflowTask Информация о задаче процесса
type WorkflowTask interface {
	// GetId Возвращает идентификатор задачи
	GetId() int64
	// GetName Возвращает имя задачи в рамках процесса
	GetName() string
	// GetProcessId Возвращает идентификатор связанного с задачей процесса
	GetProcessId() int64
	// GetRetriesCount Возвращает количество ранее выполненных ретраев для задачи. 0 если повторных попыток ещё не было
	GetRetriesCount() int32
	// GetScopeVariable Возвращает переменную из скоупа процесса. Возвращает nil если переменная не найдена
	GetScopeVariable(key string) (interface{}, bool)
}

type workflowTask struct {
	TaskId    int64
	TaskName  string
	ProcessId int64
	Retries   int32
	Scope     map[string]interface{}
}

func (task *workflowTask) GetId() int64 {
	return task.TaskId
}

func (task *workflowTask) GetName() string {
	return task.TaskName
}

func (task *workflowTask) GetProcessId() int64 {
	return task.ProcessId
}

func (task *workflowTask) GetRetriesCount() int32 {
	return task.Retries
}

func (task *workflowTask) GetScopeVariable(key string) (interface{}, bool) {
	value, contains := task.Scope[key]
	return value, contains
}
