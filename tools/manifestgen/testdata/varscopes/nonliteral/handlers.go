//go:build wfmanifestgen

package handlers

func init() {
	apiregister.RegisterHandler(newTenant())
}

func newTenant() *varsHandler {
	return &varsHandler{scopeName: name}
}
