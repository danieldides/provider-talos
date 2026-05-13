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

package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane-contrib/provider-talos/apis/machine/v1alpha1"
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

func TestBuildBootstrapTLSConfig(t *testing.T) {
	caCert, clientCert, clientKey := generateTestCertificates(t)

	tests := map[string]struct {
		clientConfig v1alpha1.ClientConfiguration
		node         string
		check        func(t *testing.T, cfg *tls.Config)
		wantErr      string
	}{
		"SecureWithCA": {
			clientConfig: v1alpha1.ClientConfiguration{
				CACertificate:     caCert,
				ClientCertificate: clientCert,
				ClientKey:         clientKey,
			},
			node: "127.0.0.1",
			check: func(t *testing.T, cfg *tls.Config) {
				t.Helper()
				if len(cfg.Certificates) != 1 {
					t.Fatalf("buildBootstrapTLSConfig(...): got %d certificates, want 1", len(cfg.Certificates))
				}
				if cfg.RootCAs == nil {
					t.Fatal("buildBootstrapTLSConfig(...): RootCAs = nil")
				}
				if diff := cmp.Diff("127.0.0.1", cfg.ServerName); diff != "" {
					t.Errorf("buildBootstrapTLSConfig(...).ServerName: -want, +got:\n%s", diff)
				}
				if diff := cmp.Diff(uint16(tls.VersionTLS12), cfg.MinVersion); diff != "" {
					t.Errorf("buildBootstrapTLSConfig(...).MinVersion: -want, +got:\n%s", diff)
				}
			},
		},
		"InsecureCACertificateSkipsRootCAs": {
			clientConfig: v1alpha1.ClientConfiguration{
				CACertificate:     "insecure",
				ClientCertificate: clientCert,
				ClientKey:         clientKey,
			},
			check: func(t *testing.T, cfg *tls.Config) {
				t.Helper()
				if cfg.RootCAs != nil {
					t.Fatal("buildBootstrapTLSConfig(...): RootCAs != nil")
				}
			},
		},
		"InvalidCAErrors": {
			clientConfig: v1alpha1.ClientConfiguration{
				CACertificate:     "invalid",
				ClientCertificate: clientCert,
				ClientKey:         clientKey,
			},
			wantErr: "failed to parse CA certificate",
		},
		"InvalidClientCertificateErrors": {
			clientConfig: v1alpha1.ClientConfiguration{
				CACertificate:     caCert,
				ClientCertificate: "invalid",
				ClientKey:         clientKey,
			},
			wantErr: "failed to create client certificate",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := buildBootstrapTLSConfig(tc.clientConfig, tc.node)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("buildBootstrapTLSConfig(...): expected error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("buildBootstrapTLSConfig(...): got error %q, want to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildBootstrapTLSConfig(...): unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func generateTestCertificates(t *testing.T) (string, string, string) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(...): %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("x509.CreateCertificate(...): %v", err)
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(...): %v", err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("x509.CreateCertificate(...): %v", err)
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})

	return string(caPEM), string(clientCertPEM), string(clientKeyPEM)
}
