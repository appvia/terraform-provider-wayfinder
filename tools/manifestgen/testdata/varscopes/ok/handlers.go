//go:build wfmanifestgen

package handlers

func init() {
	apiregister.RegisterHandler(newWorkspace())
	apiregister.RegisterHandler(newInstance())
	apiregister.RegisterHandler(newTenant())
	apiregister.RegisterHandler(newEnvironment())
	apiregister.RegisterHandler(newTenant())
}

func newTenant() *varsHandler {
	return &varsHandler{scopeName: `tenant`}
}

func newWorkspace() *varsHandler {
	return &varsHandler{scopeName: "workspace"}
}

func newEnvironment() varsHandler {
	return varsHandler{scopeName: "\x65nvironment"}
}

func newInstance() *varsHandler {
	return &varsHandler{scopeName: "instance"}
}
