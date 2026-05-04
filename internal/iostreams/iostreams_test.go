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

package iostreams

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemReturnsStandardStreams(t *testing.T) {
	io := System()
	require.NotNil(t, io.In)
	require.NotNil(t, io.Out)
	require.NotNil(t, io.ErrOut)
}

func TestIOStreamsPrintHelpers(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	io := Test(nil, &out, &errOut)

	io.Println("hello")
	io.Printf("%s", "world")
	io.Eprintln("bad")
	io.Eprintf("%s", "news")

	assert.Equal(t, "hello\nworld", out.String())
	assert.Equal(t, "bad\nnews", errOut.String())
}
