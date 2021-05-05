// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package filter

import (
	"context"
	"io"
	"net/http"
)

// MediaFilter functions can transform bytes from input to output.
type MediaFilter func(context.Context, MediaFilterHandle) error

// MediaFilterHandle is a pair of input and output for the filter to read and write.
// Request and response are also included in case the filter needs to refer to
// or modify those.
type MediaFilterHandle struct {
	input    *io.PipeReader
	output   *io.PipeWriter
	request  *http.Request
	response http.ResponseWriter
}

// Performs a copy to response with filters applied to the input.
func FilteredResponse(ctx context.Context, response http.ResponseWriter, input io.Reader, request *http.Request, filters []MediaFilter) (int64, error) {
	inputReader, inputWriter := io.Pipe()
	// prime the pump by writing the input to the first pipe
	go func() {
		io.Copy(inputWriter, input)
		inputWriter.Close()
	}()
	// variable for last pipe's reader (output) in outer scope
	var lastFilterReader *io.PipeReader
	for i, filter := range filters {
		// make a new pipe
		filterReader, filterWriter := io.Pipe()
		// decide whether to read from input, or the last filter
		var inputSource *io.PipeReader
		if i == 0 {
			inputSource = inputReader
		} else {
			inputSource = lastFilterReader
		}
		// run filter goroutine
		go filter(ctx, MediaFilterHandle{
			input:    inputSource,
			output:   filterWriter,
			request:  request,
			response: response,
		})
		// update last filter pipereader for next filter or output
		lastFilterReader = filterReader
	}
	return io.Copy(response, lastFilterReader)
}

// NoOpFilter does nothing to the media.
func NoOpFilter(ctx context.Context, handle MediaFilterHandle) error {
	defer handle.input.Close()
	defer handle.output.Close()
	_, err := io.Copy(handle.output, handle.input)
	return err
}
