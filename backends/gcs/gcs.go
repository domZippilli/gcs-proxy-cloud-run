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
package gcs

import (
	"context"
	"os"

	storage "cloud.google.com/go/storage"
)

var bucket string
var gcs *storage.Client

// setup performs one-time setup for the GCS backend.
func Setup() error {
	// set the bucket name from environment variable
	bucket = os.Getenv("BUCKET_NAME")

	// initialize the client
	var err error
	gcs, err = storage.NewClient(context.Background())
	if err != nil {
		return err
	}
	return nil
}
