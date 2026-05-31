package domain

import "errors"

type IntegrationUnavailableError struct {
	Platform DestinationPlatform
	Message  string
}

func (e *IntegrationUnavailableError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return "integration is not connected yet"
}

func IsIntegrationUnavailable(err error) bool {
	var target *IntegrationUnavailableError
	return errors.As(err, &target)
}
