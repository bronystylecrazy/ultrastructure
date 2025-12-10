package main

import "github.com/bronystylecrazy/flexinfra/logging"

type ExampleService struct {
	logging.Log
}

func NewExampleService() *ExampleService {
	return &ExampleService{}
}

func (e *ExampleService) Print() {
	e.L().Info("🚀 Example Service Print!")
}
