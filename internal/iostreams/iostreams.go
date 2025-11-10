// Copyright 2025 coScene
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
	"fmt"
	"io"
	"os"
)

// IOStreams provides the standard names for iostreams.
// This is the same pattern used in GitHub CLI.
type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer
}

// System returns the default IOStreams for the system
func System() *IOStreams {
	return &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// Test returns IOStreams for testing
func Test(in io.ReadCloser, out, errOut io.Writer) *IOStreams {
	return &IOStreams{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	}
}

// Println prints to Out with a newline
func (s *IOStreams) Println(a ...interface{}) {
	fmt.Fprintln(s.Out, a...)
}

// Printf prints formatted to Out
func (s *IOStreams) Printf(format string, a ...interface{}) {
	fmt.Fprintf(s.Out, format, a...)
}

// Eprintln prints to ErrOut with a newline
func (s *IOStreams) Eprintln(a ...interface{}) {
	fmt.Fprintln(s.ErrOut, a...)
}

// Eprintf prints formatted to ErrOut
func (s *IOStreams) Eprintf(format string, a ...interface{}) {
	fmt.Fprintf(s.ErrOut, format, a...)
}
