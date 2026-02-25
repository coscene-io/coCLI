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
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHeap(t *testing.T) {
	h := NewHeap([]int{5, 3, 1, 4, 2})
	assert.Equal(t, 5, h.Len())
	assert.Equal(t, 1, h.Peek())
}

func TestHeap_PushPop(t *testing.T) {
	h := NewHeap(nil)
	heap.Push(h, 3)
	heap.Push(h, 1)
	heap.Push(h, 2)

	assert.Equal(t, 1, heap.Pop(h))
	assert.Equal(t, 2, heap.Pop(h))
	assert.Equal(t, 3, heap.Pop(h))
}

func TestHeap_Peek_Empty(t *testing.T) {
	h := NewHeap(nil)
	assert.Equal(t, 0, h.Peek())
}

func TestHeap_Remove(t *testing.T) {
	h := NewHeap([]int{1, 2, 3, 4, 5})
	h.Remove(3)
	assert.Equal(t, 4, h.Len())
	assert.Equal(t, 1, h.Peek())
}

func TestHeap_Remove_NonExistent(t *testing.T) {
	h := NewHeap([]int{1, 2, 3})
	h.Remove(99)
	assert.Equal(t, 3, h.Len())
}

func TestFindMinMissingInteger(t *testing.T) {
	tests := []struct {
		name string
		arr  []int
		want int
	}{
		{"empty", []int{}, 1},
		{"sequential from 1", []int{1, 2, 3}, 4},
		{"gap at start", []int{2, 3, 4}, 1},
		{"gap in middle", []int{1, 2, 4, 5}, 3},
		{"single element 1", []int{1}, 2},
		{"single element 2", []int{2}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FindMinMissingInteger(tt.arr))
		})
	}
}
