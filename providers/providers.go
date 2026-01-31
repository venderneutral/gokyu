// Package providers imports all available gokyu providers.
// Use this package when you want to support multiple providers
// and switch between them via configuration.
//
// Usage:
//
//	import _ "github.com/venderneutral/gokyu/providers"
package providers

import (
	_ "github.com/venderneutral/gokyu/providers/amazonmq"
	_ "github.com/venderneutral/gokyu/providers/azure"
)
