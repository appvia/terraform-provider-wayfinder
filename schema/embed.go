// Package schema embeds the generated Wayfinder manifest into the provider binary.
package schema

import _ "embed"

// Manifest is the raw wayfinder-manifest.json document.
//
//go:embed wayfinder-manifest.json
var Manifest []byte
