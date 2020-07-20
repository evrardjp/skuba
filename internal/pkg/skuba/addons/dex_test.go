/*
 * Copyright (c) 2019 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package addons

import (
	"fmt"
	"regexp"
	"testing"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/SUSE/skuba/internal/pkg/skuba/kubernetes"
	"github.com/SUSE/skuba/internal/pkg/skuba/skuba"
	img "github.com/SUSE/skuba/pkg/skuba"
)

func TestGetDexImage(t *testing.T) {
	tests := []struct {
		name     string
		imageTag string
		want     string
	}{
		{
			name:     "get dex init image without revision",
			imageTag: "2.16.0",
			want:     "caasp-dex:2.16.0",
		},
		{
			name:     "get dex init image with revision",
			imageTag: "2.16.0-rev2",
			want:     "caasp-dex:2.16.0-rev2",
		},
	}

	for _, ver := range kubernetes.AvailableVersions() {
		for _, tt := range tests {
			tt := tt // Parallel testing
			t.Run(tt.name, func(t *testing.T) {
				imageUri := fmt.Sprintf("%s/%s", img.ImageRepository(ver), tt.want)

				if got := GetDexImage(ver, tt.imageTag); got != imageUri {
					t.Errorf("GetDexImage() = %v, want %v", got, imageUri)
				}
			})
		}
	}
}

func Test_renderContext_DexImage(t *testing.T) {
	type test struct {
		name          string
		renderContext renderContext
		want          string
	}
	for _, ver := range kubernetes.AvailableVersions() {
		tt := test{
			name: "render dex Image URL when cluster version is " + ver.String(),
			renderContext: renderContext{
				config: AddonConfiguration{
					ClusterVersion: ver,
					ControlPlane:   "",
					ClusterName:    "",
				},
			},
			want: img.ImageRepository(ver) + "/caasp-dex:([[:digit:]]{1,}.){2}[[:digit:]]{1,}(-rev[:digit:]{1,})?",
		}
		t.Run(tt.name, func(t *testing.T) {
			got := tt.renderContext.DexImage()
			matched, err := regexp.Match(tt.want, []byte(got))
			if err != nil {
				t.Error(err)
				return
			}
			if !matched {
				t.Errorf("renderContext.DexImage() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func Test_dexCallbacks_beforeApply(t *testing.T) {
	type test struct {
		name               string
		addonConfiguration AddonConfiguration
		skubaConfiguration *skuba.SkubaConfiguration
		wantErr            bool
	}
	for _, ver := range kubernetes.AvailableVersions() {
		tt := test{
			name:               "test dex callbacks beforeApply when cluster version is " + ver.String(),
			addonConfiguration: AddonConfiguration{},
			wantErr:            true,
		}
		t.Run(tt.name, func(t *testing.T) {
			d := dexCallbacks{}
			if err := d.beforeApply(fake.NewSimpleClientset(), tt.addonConfiguration, tt.skubaConfiguration); (err != nil) != tt.wantErr {
				t.Errorf("dexCallbacks.beforeApply() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
