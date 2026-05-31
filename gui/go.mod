module github.com/nycjv321/dnsctl/gui

go 1.24.5

require (
	github.com/nycjv321/dnsctl v0.0.0
	github.com/wailsapp/wails/v2 v2.10.1
)

// Use the sibling main module for the guiapi/service/hosts packages.
replace github.com/nycjv321/dnsctl => ../

// NOTE: run `go mod tidy` inside this directory (requires network + the Wails
// dependency tree) to populate the full require list and go.sum. It is kept
// minimal here because the Wails toolchain is needed to build the GUI.
