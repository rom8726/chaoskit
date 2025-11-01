//go:build failpoint

package injectors

import (
	failpoint "github.com/pingcap/failpoint"
)

func enableFailpoint(name, action string) error {
	return failpoint.Enable(name, action)
}

func disableFailpoint(name string) error {
	return failpoint.Disable(name)
}
