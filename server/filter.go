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
package main

import (
	"bytes"
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
	input    io.Reader
	output   io.Writer
	request  *http.Request
	response http.ResponseWriter
}

// FilterChain creates a chain of goroutine filters, with buffers between them
// to connect their inputs and outputs. The given input will start the chain,
// and the returned output will be at the end of the chain.
func FilterChain(ctx context.Context, input io.Reader, request *http.Request, response http.ResponseWriter, filters []MediaFilter) (*bytes.Buffer, error) {
	var i io.Reader
	var o *bytes.Buffer
	i = input
	o = new(bytes.Buffer)
	for _, filter := range filters {
		go filter(ctx, MediaFilterHandle{
			input:    i,
			output:   o,
			request:  request,
			response: response,
		})
		i = o
		o = new(bytes.Buffer)
	}
	return o, nil
}

// NoOpFilter does nothing to the bytes, just copies them from input to
// output.
func NoOpFilter(ctx context.Context, handle MediaFilterHandle) error {
	io.Copy(handle.output, handle.input)
	return nil
}
