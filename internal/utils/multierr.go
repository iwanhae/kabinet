package utils

import "fmt"

type MultiError struct {
	Errors []error
}

func (m *MultiError) Error() string {
	errString := ""
	for _, err := range m.Errors {
		errString += fmt.Sprintf(": %s", err.Error())
	}
	return errString
}

func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}
