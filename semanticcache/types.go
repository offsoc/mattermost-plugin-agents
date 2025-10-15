// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semanticcache

import (
	"database/sql"
)

// SimpleCache provides semantic caching with pgvector
type SimpleCache struct {
	db        *sql.DB
	embedder  func(string) ([]float32, error)
	threshold float32
	enabled   bool
}
