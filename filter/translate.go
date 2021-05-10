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
	"net/http"
	"strings"

	translate "cloud.google.com/go/translate/apiv3"
	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"golang.org/x/text/language"
	translatepb "google.golang.org/genproto/googleapis/cloud/translate/v3"
)

// Translate translates the media from one language to another.
//
// This function should be called from a lambda that applies desired values for
// translation, leaving only ctx and handle for use as a MediaFilter.
//
// For example:
//   func(ctx context.Context, handle MediaFilterHandle) error {
//   	return Translate(ctx, handle, language.English, language.Spanish, translate.HTML)
//   },
//
// This is an example of a store-and-forward filter, in that it loads the
// entire response to perform its transformation, so it will use memory at least
// equal to the source, and add its processing time to latency.
func Translate(ctx context.Context, handle MediaFilterHandle,
	fromLang language.Tag, toLang language.Tag) error {
	defer handle.input.Close()
	defer handle.output.Close()
	// get Translate client
	translateClient, err := translate.NewTranslationClient(ctx)
	if err != nil {
		return FilterError(handle, http.StatusInternalServerError, "translate filter: %v", err)
	}
	// read the content into a string
	media := new(bytes.Buffer)
	io.Copy(media, handle.input)
	// formulate the request
	projectId, err := common.GetRuntimeProjectId()
	if err != nil {
		return FilterError(handle, http.StatusInternalServerError, "translate filter: %v", err)
	}
	request := translatepb.TranslateTextRequest{
		Parent:             fmt.Sprintf("projects/%v/locations/global", projectId),
		Contents:           []string{media.String()},
		SourceLanguageCode: fromLang.String(),
		TargetLanguageCode: toLang.String(),
		MimeType:           handle.response.Header().Get("Content-Type"),
	}
	// make the request
	response, err := translateClient.TranslateText(ctx, &request)
	if err != nil {
		return FilterError(handle, http.StatusInternalServerError, "translate filter: %v", err)
	}
	// get the translation
	translationString := response.Translations[0].TranslatedText
	// reset content-length header. It is no longer accurate.
	handle.response.Header().Set("Content-Length", fmt.Sprint(len(translationString)))
	// send the translation
	io.Copy(handle.output, strings.NewReader(translationString))
	return nil
}
