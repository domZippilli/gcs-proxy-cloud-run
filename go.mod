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
module github.com/DomZippilli/gcs-proxy-cloud-function

go 1.16

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.14.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rs/zerolog v1.21.0
	golang.org/x/text v0.3.6
	google.golang.org/genproto v0.0.0-20210506142907-4a47615972c2
)
