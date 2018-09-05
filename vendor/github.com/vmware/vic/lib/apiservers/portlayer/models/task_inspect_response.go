package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// TaskInspectResponse task inspect response
// swagger:model TaskInspectResponse
type TaskInspectResponse struct {

	// exit code
	ExitCode int64 `json:"exitCode,omitempty"`

	// id
	// Required: true
	ID string `json:"id"`

	// open stderr
	OpenStderr bool `json:"openStderr,omitempty"`

	// open stdin
	OpenStdin bool `json:"openStdin,omitempty"`

	// open stdout
	OpenStdout bool `json:"openStdout,omitempty"`

	// pid
	Pid int64 `json:"pid,omitempty"`

	// process config
	ProcessConfig *ProcessConfig `json:"processConfig,omitempty"`

	// state
	State string `json:"state,omitempty"`

	// tty
	Tty bool `json:"tty,omitempty"`

	// user
	User string `json:"user,omitempty"`
}

// Validate validates this task inspect response
func (m *TaskInspectResponse) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateID(formats); err != nil {
		// prop
		res = append(res, err)
	}

	if err := m.validateProcessConfig(formats); err != nil {
		// prop
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *TaskInspectResponse) validateID(formats strfmt.Registry) error {

	if err := validate.RequiredString("id", "body", string(m.ID)); err != nil {
		return err
	}

	return nil
}

func (m *TaskInspectResponse) validateProcessConfig(formats strfmt.Registry) error {

	if swag.IsZero(m.ProcessConfig) { // not required
		return nil
	}

	if m.ProcessConfig != nil {

		if err := m.ProcessConfig.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("processConfig")
			}
			return err
		}
	}

	return nil
}
