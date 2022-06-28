// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package common

import (
	"fmt"
	"io"
)

type ReadFFwder struct {
	Media  io.Reader
	Size   int64
	offset int64
}

func (g *ReadFFwder) Read(p []byte) (n int, err error) {
	offset_increment, err := g.Media.Read(p)
	g.offset += int64(offset_increment)
	return offset_increment, err
}

// Seek supports fast-forwarding the GCS media stream.
func (g *ReadFFwder) Seek(offset int64, whence int) (int64, error) {
	var err error

	// first, calculate the new offset
	var newOffset int64
	if whence == 2 {
		// offset from end
		newOffset = g.Size - offset
	} else if whence == 1 {
		// offset from current
		newOffset = g.offset + offset
	} else if whence == 0 {
		// offset from start
		newOffset = offset
	} else {
		err = fmt.Errorf("unsupported seek whence: %v", whence)
	}

	// next, validate we can do new offset
	// is rewind?
	if newOffset < g.offset {
		err = fmt.Errorf("unsupported rewind seek: old_offset %v offset %v, whence %v", g.offset, offset, whence)
	}
	// is past EOF?
	if newOffset > g.Size-1 {
		err = fmt.Errorf("unsupported seek past EOF: size: %v offset %v, whence %v", g.Size, offset, whence)
	}
	if err != nil {
		return g.offset, err
	}

	// finally, fast-forward the media stream
	discarded, err := io.CopyN(io.Discard, g.Media, newOffset-g.offset)
	g.offset += discarded
	if err != nil {
		return g.offset, err
	}
	return g.offset, nil
}
