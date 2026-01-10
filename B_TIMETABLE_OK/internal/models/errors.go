package models

import "fmt"

type ErrorUnprocessableEntity struct {
	Message string `default:""`
}

func (e ErrorUnprocessableEntity) Error() string {
	return e.Message
}

type ErrorNotFound struct {
	Message string `default:""`
}

func (e ErrorNotFound) Error() string {
	return fmt.Sprintf("Not found - %s", e.Message)
}

type ErrorGeneric struct {
	Message string `default:""`
}

func (e ErrorGeneric) Error() string {
	return e.Message
}
