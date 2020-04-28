/*
Copyright 2020 The Jetstack cert-manager contributors.

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

package requestmanager

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	coretesting "k8s.io/client-go/testing"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	controllerpkg "github.com/jetstack/cert-manager/pkg/controller"
	testpkg "github.com/jetstack/cert-manager/pkg/controller/test"
	"github.com/jetstack/cert-manager/pkg/util/pki"
	"github.com/jetstack/cert-manager/test/unit/gen"
)

func mustGenerateRSA(t *testing.T, keySize int) []byte {
	pk, err := pki.GenerateRSAPrivateKey(keySize)
	if err != nil {
		t.Fatal(err)
	}
	d, err := pki.EncodePKCS8PrivateKey(pk)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustGenerateECDSA(t *testing.T, keySize int) []byte {
	pk, err := pki.GenerateECPrivateKey(keySize)
	if err != nil {
		t.Fatal(err)
	}
	d, err := pki.EncodePKCS8PrivateKey(pk)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func relaxedCertificateRequestMatcher(l coretesting.Action, r coretesting.Action) error {
	objL := l.(coretesting.CreateAction).GetObject().(*cmapi.CertificateRequest).DeepCopy()
	objR := r.(coretesting.CreateAction).GetObject().(*cmapi.CertificateRequest).DeepCopy()
	objL.Spec.CSRPEM = nil
	objR.Spec.CSRPEM = nil
	if !reflect.DeepEqual(objL, objR) {
		return fmt.Errorf("unexpected difference between actions: %s", pretty.Diff(objL, objR))
	}
	return nil
}

func TestProcessItem(t *testing.T) {
	bundle1 := mustCreateCryptoBundle(t, &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testns",
			Name:      "test",
			UID:       "test",
		},
		Spec: cmapi.CertificateSpec{CommonName: "test-bundle-1"}},
	)
	bundle2 := mustCreateCryptoBundle(t, &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testns",
			Name:      "test",
			UID:       "test",
		},
		Spec: cmapi.CertificateSpec{CommonName: "test-bundle-2"}},
	)
	tests := map[string]struct {
		// key that should be passed to ProcessItem.
		// if not set, the 'namespace/name' of the 'Certificate' field will be used.
		// if neither is set, the key will be ""
		key string

		// Certificate to be synced for the test.
		// if not set, the 'key' will be passed to ProcessItem instead.
		certificate *cmapi.Certificate

		secrets []runtime.Object

		// Request, if set, will exist in the apiserver before the test is run.
		requests []runtime.Object

		expectedActions []testpkg.Action

		expectedEvents []string

		// err is the expected error text returned by the controller, if any.
		err string
	}{
		"do nothing if an empty 'key' is used": {},
		"do nothing if an invalid 'key' is used": {
			key: "abc/def/ghi",
		},
		"do nothing if a key references a Certificate that does not exist": {
			key: "namespace/name",
		},
		"do nothing if Certificate has 'Issuing' condition set to 'false'": {
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionFalse}),
			),
		},
		"do nothing if Certificate has no 'Issuing' condition": {
			certificate: bundle1.certificate,
		},
		"do nothing if status.nextPrivateKeySecretName is not set": {
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
		},
		"do nothing if status.nextPrivateKeySecretName does not exist": {
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("does-not-exist"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
		},
		"do nothing if status.nextPrivateKeySecretName contains no data": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists-but-empty"},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists-but-empty"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
		},
		"do nothing if status.nextPrivateKeySecretName contains invalid data": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists-but-invalid"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: []byte("invalid")},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists-but-invalid"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
		},
		"create a CertificateRequest if none exists": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: bundle1.certificate.Namespace, Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: bundle1.privateKeyBytes},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "1",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"delete the owned CertificateRequest and create a new one if existing one does not have the annotation": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: mustGenerateRSA(t, 2048)},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "",
					}),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewAction(coretesting.NewDeleteAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns", "test")),
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "1",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"delete the owned CertificateRequest and create a new one if existing one contains invalid annotation": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: mustGenerateRSA(t, 2048)},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "invalid",
					}),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewAction(coretesting.NewDeleteAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns", "test")),
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "1",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"do nothing if existing CertificateRequest is valid for the spec": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: bundle1.privateKeyBytes},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "1",
					}),
				),
			},
		},
		"should delete requests that contain invalid CSR data": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: bundle1.privateKeyBytes},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "1",
					}),
					gen.SetCertificateRequestCSR([]byte("invalid")),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewAction(coretesting.NewDeleteAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns", "test")),
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "1",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"should ignore requests that do not have a revision of 'current + 1' and create a new one": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: mustGenerateRSA(t, 2048)},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "3",
					}),
				),
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestName("testing-number-2"),
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "4",
					}),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "1",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"should delete request for the current revision if public keys do not match": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: mustGenerateRSA(t, 2048)},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "1",
					}),
				),
				// included here just to ensure it does not get deleted as it is not for the
				// 'next' revision that is being requested
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestName("testing-number-2"),
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "4",
					}),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewAction(coretesting.NewDeleteAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns", "test")),
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "1",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"should delete request for the current revision if public keys do not match (with explicit revision)": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: bundle2.privateKeyBytes},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
				gen.SetCertificateRevision(5),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "6",
					}),
				),
				// included here just to ensure it does not get deleted as it is not for the
				// 'next' revision that is being requested
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestName("testing-number-2"),
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "5",
					}),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewAction(coretesting.NewDeleteAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns", "test")),
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle2.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "6",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"should recreate the CertificateRequest if the CSR is not signed by the stored private key": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: mustGenerateRSA(t, 2048)},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
				gen.SetCertificateRevision(5),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "6",
					}),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewAction(coretesting.NewDeleteAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns", "test")),
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "6",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"should recreate the CertificateRequest if the CSR does not match requirements on spec": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: bundle1.privateKeyBytes},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateCommonName("something-different"),
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
				gen.SetCertificateRevision(5),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "6",
					}),
				),
			},
			expectedEvents: []string{`Normal Requested Created new CertificateRequest resource "test-notrandom"`},
			expectedActions: []testpkg.Action{
				testpkg.NewAction(coretesting.NewDeleteAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns", "test")),
				testpkg.NewCustomMatch(coretesting.NewCreateAction(cmapi.SchemeGroupVersion.WithResource("certificaterequests"), "testns",
					gen.CertificateRequestFrom(bundle1.certificateRequest,
						gen.SetCertificateRequestAnnotations(map[string]string{
							cmapi.CRPrivateKeyAnnotationKey:               "exists",
							cmapi.CertificateRequestRevisionAnnotationKey: "6",
						}),
					)), relaxedCertificateRequestMatcher),
			},
		},
		"should do nothing if request has an up to date CSR and it is still pending": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: bundle1.privateKeyBytes},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
				gen.SetCertificateRevision(5),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "6",
					}),
				),
			},
		},
		"should do nothing if multiple owned and up to date CertificateRequests for the current revision exist": {
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: "testns", Name: "exists"},
					Data:       map[string][]byte{corev1.TLSPrivateKeyKey: bundle1.privateKeyBytes},
				},
			},
			certificate: gen.CertificateFrom(bundle1.certificate,
				gen.SetCertificateNextPrivateKeySecretName("exists"),
				gen.SetCertificateStatusCondition(cmapi.CertificateCondition{Type: cmapi.CertificateConditionIssuing, Status: cmmeta.ConditionTrue}),
				gen.SetCertificateRevision(5),
			),
			requests: []runtime.Object{
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "6",
					}),
				),
				gen.CertificateRequestFrom(bundle1.certificateRequest,
					gen.SetCertificateRequestName("another-name-2"),
					gen.SetCertificateRequestAnnotations(map[string]string{
						cmapi.CRPrivateKeyAnnotationKey:               "exists",
						cmapi.CertificateRequestRevisionAnnotationKey: "6",
					}),
				),
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Create and initialise a new unit test builder
			builder := &testpkg.Builder{
				T:               t,
				ExpectedEvents:  test.expectedEvents,
				ExpectedActions: test.expectedActions,
				StringGenerator: func(i int) string { return "notrandom" },
			}
			if test.certificate != nil {
				builder.CertManagerObjects = append(builder.CertManagerObjects, test.certificate)
			}
			if test.secrets != nil {
				builder.KubeObjects = append(builder.KubeObjects, test.secrets...)
			}
			for _, req := range test.requests {
				builder.CertManagerObjects = append(builder.CertManagerObjects, req)
			}
			builder.Init()

			// Register informers used by the controller using the registration wrapper
			w := &controllerWrapper{}
			_, _, err := w.Register(builder.Context)
			if err != nil {
				t.Fatal(err)
			}
			// Start the informers and begin processing updates
			builder.Start()
			defer builder.Stop()

			key := test.key
			if key == "" && test.certificate != nil {
				key, err = controllerpkg.KeyFunc(test.certificate)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Call ProcessItem
			err = w.controller.ProcessItem(context.Background(), key)
			switch {
			case err != nil:
				if test.err != err.Error() {
					t.Errorf("error text did not match, got=%s, exp=%s", err.Error(), test.err)
				}
			default:
				if test.err != "" {
					t.Errorf("got no error but expected: %s", test.err)
				}
			}

			if err := builder.AllEventsCalled(); err != nil {
				builder.T.Error(err)
			}
			if err := builder.AllActionsExecuted(); err != nil {
				builder.T.Error(err)
			}
			if err := builder.AllReactorsCalled(); err != nil {
				builder.T.Error(err)
			}
		})
	}
}