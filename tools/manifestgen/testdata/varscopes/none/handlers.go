//go:build wfmanifestgen

package handlers

func init() {
	apiregister.RegisterHandler(newOther())
}

func newOther() *otherHandler {
	return &otherHandler{}
}
