/*
Copyright 2022 The Kubernetes Authors.

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

package converters

import (
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/gomega"
)

func TestGetSubnetAddresses(t *testing.T) {
	tests := []struct {
		name   string
		subnet network.Subnet
		want   []string
	}{
		{
			name:   "nil properties subnet",
			subnet: network.Subnet{},
		},
		{
			name: "subnet with single address prefix",
			subnet: network.Subnet{
				SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
					AddressPrefix: to.StringPtr("test-address-prefix"),
				},
			},
			want: []string{"test-address-prefix"},
		},
		{
			name: "subnet with multiple address prefixes",
			subnet: network.Subnet{
				SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
					AddressPrefixes: &[]string{"test-address-prefix-1", "test-address-prefix-2"},
				},
			},
			want: []string{"test-address-prefix-1", "test-address-prefix-2"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			got := GetSubnetAddresses(tt.subnet)
			g.Expect(got).To(Equal(tt.want), fmt.Sprintf("got: %v, want: %v", got, tt.want))
		})
	}
}
