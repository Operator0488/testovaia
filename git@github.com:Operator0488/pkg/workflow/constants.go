package workflow

import "time"

const (
	workflowEventsTTLHours      = 1800 // 30 дней
	workflowRetryTimeoutSeconds = 60
	maxRetryCountBeforeIncident = 5
	cancelOperationTimeoutInSec = 5
	requestTimeoutSec           = 1
	poolIntervalSec             = 1
	maxActiveJobs               = 10
	tasksConcurrency            = 3
	pollThreshold               = 0.3
	initializeErrBackoff        = 5 * time.Second
)
