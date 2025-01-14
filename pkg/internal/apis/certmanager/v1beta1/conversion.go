/*
Copyright 2020 The cert-manager Authors.

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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/conversion"

	"github.com/jetstack/cert-manager/pkg/apis/certmanager/v1beta1"
	"github.com/jetstack/cert-manager/pkg/internal/apis/certmanager"
)

func Convert_certmanager_X509Subject_To_v1beta1_X509Subject(in *certmanager.X509Subject, out *v1beta1.X509Subject, s conversion.Scope) error {
	return autoConvert_certmanager_X509Subject_To_v1beta1_X509Subject(in, out, s)
}
