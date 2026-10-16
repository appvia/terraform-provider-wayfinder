//go:build wfmanifestgen

package handlers

func init() {
	apiregister.RegisterHandler(newTenant())
	other.RegisterHandler(newWrongPackage())
	_ = func() { apiregister.RegisterHandler(newDeferred()) }
	apiregister.RegisterHandler(owner.newMethod())
}

func registerLater() {
	apiregister.RegisterHandler(newInstance())
}

func newTenant() *varsHandler {
	return &varsHandler{scopeName: "tenant"}
}

func newInstance() *varsHandler {
	return &varsHandler{scopeName: "instance"}
}

func newWrongPackage() *varsHandler {
	return &varsHandler{scopeName: "wrongpackage"}
}

func newDeferred() *varsHandler {
	return &varsHandler{scopeName: "deferred"}
}

func (owner receiver) newMethod() *varsHandler {
	return &varsHandler{scopeName: "method"}
}

func unregisteredMalformed() *varsHandler {
	return &varsHandler{scopeName: name}
}

func closureFactory() func() *varsHandler {
	return func() *varsHandler { return &varsHandler{scopeName: "closure"} }
}
