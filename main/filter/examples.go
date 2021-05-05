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
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
)

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
			return fmt.Errorf("lower filter: %v", err)
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
			return fmt.Errorf("intercalate filter: %v", err)
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
		return fmt.Errorf("translate filter: %v", err)
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
			return fmt.Errorf("translate filter: %v", err)
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
		return fmt.Errorf("translate filter: %v", err)
	}

	// write the translation
	for _, t := range translations {
		handle.output.Write([]byte(t.Text))
	}
	return nil
}
