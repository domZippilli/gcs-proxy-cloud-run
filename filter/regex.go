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
	"bytes"
	"context"
	"io"
	"net/http"
	"regexp"

	"github.com/rs/zerolog/log"
)

// BlockRegex will block responses that contain match any of the given regexes.
//
// This function is a streaming function, and will not introduce significant
// TTFB latency. It will use about 4MB of RAM, give or take, for the scanning
// window. The scanning process ends up scanning all bytes twice, in order to
// detect matches that might span a stream chunk.
//
// For example:
// window = [A B]
// window scans OK
// send A
// window = [B C]
// window scans OK
// send B
// and so on.
//
// Partial responses may be sent, but any chunk with a regex match will not be
// sent, and the first one detected cancels the rest of the copy.
//
// Patterns which may match more than 4MB of data are not supported; they will
// not error, but they simply will not be detected.
func BlockRegex(ctx context.Context, handle MediaFilterHandle, regexes []*regexp.Regexp) error {
	defer handle.input.Close()
	defer handle.output.Close()
	// make the buffer
	const chunkSize = int64(1024 * 1024 * 2)
	buffer := bytes.NewBuffer(make([]byte, 0, chunkSize*2))
	// seed the buffer with two chunks
	io.CopyN(buffer, handle.input, chunkSize*2)
	for {
		// scan the buffer
		for _, re := range regexes {
			match := re.Match(buffer.Bytes())
			if match {
				// BLOCK -- not an error, but we stop the response right now
				http.Error(handle.response, "PROHIBITED REGEX PATTERN MATCHED", http.StatusGone)
				log.Warn().Msgf("blockregex: matched %v", re.String())
				return nil
			}
		}
		// send one chunk
		if _, err := io.CopyN(handle.output, buffer, chunkSize); err != nil {
			if err == io.EOF {
				// done
				break
			}
			return FilterError(handle, http.StatusInternalServerError, "block regex: %v", err)
		}
		// refill the buffer
		if _, err := io.CopyN(buffer, handle.input, chunkSize); err != nil {
			if err == io.EOF {
				// continue, to flush buffer
				continue
			}
			return FilterError(handle, http.StatusInternalServerError, "block regex: %v", err)
		}
	}
	return nil
}
