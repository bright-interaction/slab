package deploy

import (
	"context"
	"errors"
)

// DockyardDeployer pushes a built site to a Dockyard-managed container.
//
// This is a stub. Real implementation lands in a later phase. It satisfies
// the Deployer interface so deploy.New("dockyard") returns a typed value
// instead of nil and the rest of the wiring (handlers, validation) can be
// exercised end to end.
type DockyardDeployer struct{}

// Kind reports the deployer kind.
func (d *DockyardDeployer) Kind() string { return "dockyard" }

// Validate is a placeholder until the full Dockyard schema is finalised.
func (d *DockyardDeployer) Validate(target Target) error {
	if target.Config == nil {
		return errors.New("dockyard deploy: config is empty")
	}
	return nil
}

// Deploy is not implemented yet.
func (d *DockyardDeployer) Deploy(_ context.Context, _ string, _ Target) (Result, error) {
	return Result{}, errors.New("dockyard deployer: not implemented")
}
