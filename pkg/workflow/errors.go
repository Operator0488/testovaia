package workflow

import (
	"errors"
	"fmt"
)

var (
	ErrWorkflowIsNotReady = fmt.Errorf("workflow client is not ready to work")
	ErrWorkflowIncident   = fmt.Errorf("workflow incident")
)

func MakeIncidentErr(err error) error {
	return errors.Join(ErrWorkflowIncident, err)
}

func MakeIncident(err string) error {
	return errors.Join(ErrWorkflowIncident, errors.New(err))
}
