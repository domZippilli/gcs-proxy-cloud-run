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
package common

import (
	"testing"
)

// TestNormalizePath calls NormalizePath with a few different URLs, checking
// the result against expected outcomes.
func TestNormalizePath(t *testing.T) {
	type TestCase struct {
		path string
		want string
	}

	tcs := []TestCase{
		// it should append index.html to end-in-slash paths
		{"/", "index.html"},
		{"somepath/", "somepath/index.html"},
		// it should remove leading slashes
		{"/somepath/", "somepath/index.html"},
		{"/something", "something"},
		// it should leave other paths alone
		{"somefile", "somefile"},
	}

	for _, tc := range tcs {
		got := NormalizePath(tc.path)
		if tc.want != got {
			t.Fatalf("got: %v, want: %v", got, tc.want)
		}
	}

}
