/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package configurationapply

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/crossplane-contrib/provider-talos/apis/machine/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

func TestObserve(t *testing.T) {
	type fields struct {
		service interface{}
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		// TODO: Add test cases.
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{service: tc.fields.service}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestGetConfigurationApplyEndpoint(t *testing.T) {
	node := "127.0.0.2"
	customEndpoint := "127.0.0.1:50000"
	emptyEndpoint := ""

	tests := map[string]struct {
		cr   *v1alpha1.ConfigurationApply
		want string
	}{
		"DefaultsToNodePort": {
			cr: &v1alpha1.ConfigurationApply{
				Spec: v1alpha1.ConfigurationApplySpec{
					ForProvider: v1alpha1.ConfigurationApplyParameters{Node: node},
				},
			},
			want: node + ":50000",
		},
		"UsesProvidedEndpoint": {
			cr: &v1alpha1.ConfigurationApply{
				Spec: v1alpha1.ConfigurationApplySpec{
					ForProvider: v1alpha1.ConfigurationApplyParameters{
						Node:     node,
						Endpoint: &customEndpoint,
					},
				},
			},
			want: customEndpoint,
		},
		"IgnoresEmptyEndpoint": {
			cr: &v1alpha1.ConfigurationApply{
				Spec: v1alpha1.ConfigurationApplySpec{
					ForProvider: v1alpha1.ConfigurationApplyParameters{
						Node:     node,
						Endpoint: &emptyEndpoint,
					},
				},
			},
			want: node + ":50000",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := getConfigurationApplyEndpoint(tc.cr)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("getConfigurationApplyEndpoint(...): -want, +got:\n%s", diff)
			}
		})
	}
}
