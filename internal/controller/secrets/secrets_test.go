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

package secrets

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/google/go-cmp/cmp"

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
		service *TalosSecretsService
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

func TestGenerateMachineSecretsUsesAdminClientCertificate(t *testing.T) {
	generated, err := (&external{}).generateMachineSecrets(nil)
	if err != nil {
		t.Fatalf("generateMachineSecrets(...): unexpected error: %v", err)
	}

	if generated.ClientCertificate == generated.CACertificate {
		t.Fatal("ClientCertificate matches CACertificate, want generated leaf certificate")
	}

	clientCert := parseCertificate(t, generated.ClientCertificate)
	if clientCert.IsCA {
		t.Fatal("ClientCertificate IsCA = true, want false")
	}
	if diff := cmp.Diff([]string{"os:admin"}, clientCert.Subject.Organization); diff != "" {
		t.Errorf("ClientCertificate Subject.Organization: -want, +got:\n%s", diff)
	}
	if !hasClientAuthUsage(clientCert) {
		t.Fatal("ClientCertificate missing ExtKeyUsageClientAuth")
	}

	caCert := parseCertificate(t, generated.CACertificate)
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	if _, err := clientCert.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		t.Fatalf("ClientCertificate did not verify against CACertificate: %v", err)
	}

	var talosConfig struct {
		Contexts map[string]struct {
			CA  string `json:"ca"`
			Crt string `json:"crt"`
			Key string `json:"key"`
		} `json:"contexts"`
	}
	if err := json.Unmarshal(generated.TalosConfig, &talosConfig); err != nil {
		t.Fatalf("TalosConfig JSON unmarshal failed: %v", err)
	}
	ctx, ok := talosConfig.Contexts["default"]
	if !ok {
		t.Fatal("TalosConfig missing default context")
	}
	if diff := cmp.Diff(generated.CACertificate, ctx.CA); diff != "" {
		t.Errorf("TalosConfig CA: -want, +got:\n%s", diff)
	}
	if diff := cmp.Diff(generated.ClientCertificate, ctx.Crt); diff != "" {
		t.Errorf("TalosConfig client certificate: -want, +got:\n%s", diff)
	}
	if diff := cmp.Diff(generated.ClientKey, ctx.Key); diff != "" {
		t.Errorf("TalosConfig client key: -want, +got:\n%s", diff)
	}
}

func parseCertificate(t *testing.T, certPEM string) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		t.Fatal("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}
	return cert
}

func hasClientAuthUsage(cert *x509.Certificate) bool {
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageClientAuth {
			return true
		}
	}
	return false
}
