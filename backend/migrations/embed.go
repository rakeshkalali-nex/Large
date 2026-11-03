// Package migrations contains the ordered, immutable baseline SQL migrations.
package migrations

import "embed"

// SQL is consumed by the platform migration runner in lexical filename order.
//go:embed *.sql
var SQL embed.FS
