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
	"encoding/base64"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"

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
	customEndpoint := "10.0.0.11:7445"

	cases := map[string]struct {
		cr   *v1alpha1.ConfigurationApply
		want string
	}{
		"DefaultFromNode": {
			cr: &v1alpha1.ConfigurationApply{
				Spec: v1alpha1.ConfigurationApplySpec{ForProvider: v1alpha1.ConfigurationApplyParameters{Node: "10.0.0.11"}},
			},
			want: "10.0.0.11:50000",
		},
		"ExplicitEndpoint": {
			cr: &v1alpha1.ConfigurationApply{
				Spec: v1alpha1.ConfigurationApplySpec{ForProvider: v1alpha1.ConfigurationApplyParameters{Node: "10.0.0.11", Endpoint: &customEndpoint}},
			},
			want: customEndpoint,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := getConfigurationApplyEndpoint(tc.cr)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("getConfigurationApplyEndpoint(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetConfigurationApplyMode(t *testing.T) {
	auto := "auto"
	noReboot := "no_reboot"
	invalid := "bad"

	cases := map[string]struct {
		cr      *v1alpha1.ConfigurationApply
		want    machine.ApplyConfigurationRequest_Mode
		wantErr string
	}{
		"DefaultReboot": {
			cr:   &v1alpha1.ConfigurationApply{},
			want: machine.ApplyConfigurationRequest_REBOOT,
		},
		"Auto": {
			cr:   &v1alpha1.ConfigurationApply{Spec: v1alpha1.ConfigurationApplySpec{ForProvider: v1alpha1.ConfigurationApplyParameters{ApplyMode: &auto}}},
			want: machine.ApplyConfigurationRequest_AUTO,
		},
		"NoReboot": {
			cr:   &v1alpha1.ConfigurationApply{Spec: v1alpha1.ConfigurationApplySpec{ForProvider: v1alpha1.ConfigurationApplyParameters{ApplyMode: &noReboot}}},
			want: machine.ApplyConfigurationRequest_NO_REBOOT,
		},
		"Invalid": {
			cr:      &v1alpha1.ConfigurationApply{Spec: v1alpha1.ConfigurationApplySpec{ForProvider: v1alpha1.ConfigurationApplyParameters{ApplyMode: &invalid}}},
			wantErr: "unsupported applyMode \"bad\"",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := getConfigurationApplyMode(tc.cr)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("getConfigurationApplyMode(...): expected error")
				}

				if diff := cmp.Diff(tc.wantErr, err.Error()); diff != "" {
					t.Fatalf("getConfigurationApplyMode(...): -want error, +got error:\n%s", diff)
				}

				return
			}

			if err != nil {
				t.Fatalf("getConfigurationApplyMode(...): unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("getConfigurationApplyMode(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetConfigurationApplyClientContext(t *testing.T) {
	endpoint := "10.0.0.11:7445"
	cr := &v1alpha1.ConfigurationApply{
		Spec: v1alpha1.ConfigurationApplySpec{ForProvider: v1alpha1.ConfigurationApplyParameters{
			Node:     "10.0.0.11",
			Endpoint: &endpoint,
			ClientConfiguration: v1alpha1.ClientConfiguration{
				CACertificate:     "ca-pem",
				ClientCertificate: "-----BEGIN CERTIFICATE-----\ninvalid\n-----END CERTIFICATE-----",
				ClientKey:         "-----BEGIN PRIVATE KEY-----\ninvalid\n-----END PRIVATE KEY-----",
			},
		}},
	}

	want := &clientconfig.Context{
		Endpoints: []string{endpoint},
		CA:        base64.StdEncoding.EncodeToString([]byte(cr.Spec.ForProvider.ClientConfiguration.CACertificate)),
		Crt:       base64.StdEncoding.EncodeToString([]byte(cr.Spec.ForProvider.ClientConfiguration.ClientCertificate)),
		Key:       base64.StdEncoding.EncodeToString([]byte(cr.Spec.ForProvider.ClientConfiguration.ClientKey)),
	}

	if diff := cmp.Diff(want, getConfigurationApplyClientContext(cr)); diff != "" {
		t.Fatalf("getConfigurationApplyClientContext(...): -want, +got:\n%s", diff)
	}
}
