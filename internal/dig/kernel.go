// Package dig implements logic for dependency injection using uber-go/dig.

package dig

import (
	"fmt"

	"go.uber.org/dig"
)

type Kernel struct {
	Container *dig.Container
}

func (t *Kernel) Build() (err error) {
	t.Container, err = buildContainer()
	if err != nil {
		err = fmt.Errorf("failed to build container: %w", err)
	}
	return
}

func NewKernel() *Kernel {
	return &Kernel{}
}
