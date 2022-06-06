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
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func GetSubnetAddresses(subnet network.Subnet) []string {
	var addresses []string
	if subnet.SubnetPropertiesFormat != nil && subnet.SubnetPropertiesFormat.AddressPrefix != nil {
		addresses = []string{to.String(subnet.SubnetPropertiesFormat.AddressPrefix)}
	} else if subnet.SubnetPropertiesFormat != nil && subnet.SubnetPropertiesFormat.AddressPrefixes != nil {
		addresses = to.StringSlice(subnet.SubnetPropertiesFormat.AddressPrefixes)
	}
	return addresses
}
