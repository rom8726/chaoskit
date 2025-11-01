package injectors

import "errors"

// ErrFailpointDisabled is returned when gofail runtime is not enabled (missing -tags failpoint).
var ErrFailpointDisabled = errors.New("failpoint runtime is not enabled: build with -tags failpoint and instrument target code with failpoint.Inject") //nolint:lll
