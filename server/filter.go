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
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
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

// ===== EXAMPLE FILTERS BELOW =====

// NoOpFilter does nothing to the media.
func NoOpFilter(ctx context.Context, handle MediaFilterHandle) error {
	defer handle.input.Close()
	defer handle.output.Close()
	_, err := io.Copy(handle.output, handle.input)
	return err
}

// LowerFilter applies bytes.ToLower to the media.
//
// This is an example of a streaming filter. This will use very little memory
// and add very little latency to responses.
func LowerFilter(ctx context.Context, handle MediaFilterHandle) error {
	defer handle.input.Close()
	defer handle.output.Close()
	buf := make([]byte, 4096)
	for {
		_, err := handle.input.Read(buf)
		buf = bytes.ToLower(buf)
		handle.output.Write(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("lower filter: %v", err)
		}
	}
	return nil
}

// IntercalateFilter will split the media, insert a value between elements,
// and return the concatenated result.
//
// This function should be called from a lambda that applies desired values for
// intercalation, leaving only ctx and handle for use as a MediaFilter.
//
// For example:
//   func(ctx context.Context, handle MediaFilterHandle) error {
//   	return IntercalateFilter(ctx, handle, "\n", "f")
//   },
//
// This is an example of a store-and-forward filter, in that it loads the
// entire response to perform its transformation, so it will use memory at least
// equal to the source, and add its processing time to latency.
func IntercalateFilter(ctx context.Context, handle MediaFilterHandle, separator string, insertValue string) error {
	defer handle.input.Close()
	defer handle.output.Close()

	media := new(bytes.Buffer)
	buf := make([]byte, 4096)
	for {
		_, err := handle.input.Read(buf)
		buf = bytes.ToLower(buf)
		media.Write(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("intercalate filter: %v", err)
		}
	}
	mediaString := media.String()
	tokens := strings.Split(mediaString, separator)
	outputLines := []string{}
	for i, token := range tokens {
		outputLines = append(outputLines, token)
		if i <= len(tokens)-2 {
			outputLines = append(outputLines, separator+insertValue)
		}
	}
	for _, line := range outputLines {
		handle.output.Write([]byte(string(line)))
	}
	return nil
}

// DO NOT USE -- Broken.
// TranslateFilter translates the media from one language to another.
//
// This function should be called from a lambda that applies desired values for
// translation, leaving only ctx and handle for use as a MediaFilter.
//
// For example:
//   func(ctx context.Context, handle MediaFilterHandle) error {
//   	return TranslateFilter(ctx, handle, language.English, language.Spanish, translate.HTML)
//   },
//
// This is an example of a store-and-forward filter, in that it loads the
// entire response to perform its transformation, so it will use memory at least
// equal to the source, and add its processing time to latency.
func TranslateFilter(ctx context.Context, handle MediaFilterHandle,
	fromLang language.Tag, toLang language.Tag, format translate.Format) error {
	defer handle.input.Close()
	defer handle.output.Close()

	// get Translate client
	translateClient, err := translate.NewClient(ctx)
	if err != nil {
		log.Fatalf("translate filter: %v", err)
	}

	// read the content into a string
	media := new(bytes.Buffer)
	buf := make([]byte, 4096)
	for {
		_, err := handle.input.Read(buf)
		buf = bytes.ToLower(buf)
		media.Write(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("translate filter: %v", err)
		}
	}
	mediaString := media.String()

	// get the translation
	// TODO(domz): POST instead of GET?
	translations, err := translateClient.Translate(ctx, []string{mediaString}, toLang,
		&translate.Options{
			Source: fromLang,
			Format: format,
		})
	if err != nil {
		log.Fatalf("translate filter: %v", err)
	}

	// write the translation
	for _, t := range translations {
		handle.output.Write([]byte(t.Text))
	}
	return nil
}
