// Copyright 2026 coScene
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package upload_utils

import (
	"os"
	"testing"

	"github.com/coscene-io/cocli/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestUploadDBLifecycle(t *testing.T) {
	oldUploaderDir := constants.DefaultUploaderDirPath
	constants.DefaultUploaderDirPath = t.TempDir()
	t.Cleanup(func() {
		constants.DefaultUploaderDirPath = oldUploaderDir
	})

	db, err := NewUploadDB("data.bin", "record-a", "sha256", 64)
	require.NoError(t, err)
	dbPath := db.Path()

	require.NoError(t, db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(multipartUploadsBucket)).Put(
			[]byte("checkpoint"),
			[]byte(`{"upload_id":"upload-1","uploaded_size":42}`),
		)
	}))

	var checkpoint MultipartCheckpointInfo
	require.NoError(t, db.Get("checkpoint", &checkpoint))
	assert.Equal(t, "upload-1", checkpoint.UploadId)
	assert.Equal(t, int64(42), checkpoint.UploadedSize)

	require.NoError(t, db.Reset())
	require.NoError(t, db.View(func(tx *bolt.Tx) error {
		got := tx.Bucket([]byte(multipartUploadsBucket)).Get([]byte("checkpoint"))
		assert.Nil(t, got)
		return nil
	}))

	require.NoError(t, db.Delete())
	_, err = os.Stat(dbPath)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
