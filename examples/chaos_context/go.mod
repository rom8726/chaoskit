module chaos-context-example

go 1.25

require github.com/rom8726/chaoskit v0.0.0

require (
	github.com/Shopify/toxiproxy/v2 v2.12.0 // indirect
	github.com/pingcap/errors v0.11.4 // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
)

replace github.com/rom8726/chaoskit => ../../
