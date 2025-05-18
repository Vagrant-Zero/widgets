package di

import "fmt"

const (
	initStatus = 1
	runStatus  = 2

	injectTag = "inject"
)

var (
	interfaceNilError = fmt.Errorf("interface is nil")
)
