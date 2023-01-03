/*
Copyright 2020 The Kubernetes Authors.

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
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-11-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	infrav1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-azure/azure"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SDKToVMSS converts an Azure SDK VirtualMachineScaleSet to the AzureMachinePool type.
func SDKToVMSS(sdkvmss compute.VirtualMachineScaleSet, sdkinstances []compute.VirtualMachineScaleSetVM) *azure.VMSS {
	vmss := &azure.VMSS{
		ID:    to.String(sdkvmss.ID),
		Name:  to.String(sdkvmss.Name),
		State: infrav1.ProvisioningState(to.String(sdkvmss.ProvisioningState)),
	}

	if sdkvmss.Sku != nil {
		vmss.Sku = to.String(sdkvmss.Sku.Name)
		vmss.Capacity = to.Int64(sdkvmss.Sku.Capacity)
	}

	if sdkvmss.Zones != nil && len(*sdkvmss.Zones) > 0 {
		vmss.Zones = to.StringSlice(sdkvmss.Zones)
	}

	if len(sdkvmss.Tags) > 0 {
		vmss.Tags = MapToTags(sdkvmss.Tags)
	}

	if len(sdkinstances) > 0 {
		vmss.Instances = make([]azure.VMSSVM, len(sdkinstances))
		for i, vm := range sdkinstances {
			vmss.Instances[i] = *SDKToVMSSVM(vm)
		}
	}

	if sdkvmss.VirtualMachineProfile != nil &&
		sdkvmss.VirtualMachineProfile.StorageProfile != nil &&
		sdkvmss.VirtualMachineProfile.StorageProfile.ImageReference != nil {
		imageRef := sdkvmss.VirtualMachineProfile.StorageProfile.ImageReference
		vmss.Image = SDKImageToImage(imageRef, sdkvmss.Plan != nil)
	}

	return vmss
}

// SDKToVMSSVM converts an Azure SDK VirtualMachineScaleSetVM into an infrav1exp.VMSSVM.
func SDKToVMSSVM(sdkInstance compute.VirtualMachineScaleSetVM) *azure.VMSSVM {
	instance := azure.VMSSVM{
		ID:         to.String(sdkInstance.ID),
		InstanceID: to.String(sdkInstance.InstanceID),
	}

	if sdkInstance.VirtualMachineScaleSetVMProperties == nil {
		return &instance
	}

	instance.State = infrav1.Creating
	if sdkInstance.ProvisioningState != nil {
		instance.State = infrav1.ProvisioningState(to.String(sdkInstance.ProvisioningState))
	}

	if sdkInstance.OsProfile != nil && sdkInstance.OsProfile.ComputerName != nil {
		instance.Name = *sdkInstance.OsProfile.ComputerName
	}

	if sdkInstance.StorageProfile != nil && sdkInstance.StorageProfile.ImageReference != nil {
		imageRef := sdkInstance.StorageProfile.ImageReference
		instance.Image = SDKImageToImage(imageRef, sdkInstance.Plan != nil)
	}

	if sdkInstance.Zones != nil && len(*sdkInstance.Zones) > 0 {
		// an instance should only have 1 zone, so we select the first item of the slice
		instance.AvailabilityZone = to.StringSlice(sdkInstance.Zones)[0]
	}

	return &instance
}

// SDKImageToImage converts a SDK image reference to infrav1.Image.
func SDKImageToImage(sdkImageRef *compute.ImageReference, isThirdPartyImage bool) infrav1.Image {
	imgId := to.String(sdkImageRef.ID)
	var infraImg infrav1.Image
	switch imgId {
  case "":
		infraImg = infrav1.Image{
			ID:&imgId,
				Marketplace: &infrav1.AzureMarketplaceImage{
					ImagePlan: infrav1.ImagePlan{
						Publisher: to.String(sdkImageRef.Publisher),
						Offer:     to.String(sdkImageRef.Offer),
						SKU:       to.String(sdkImageRef.Sku),
					},
					Version:         to.String(sdkImageRef.Version),
					ThirdPartyImage: isThirdPartyImage,
				},
		}
	//for now we assume if there is an image id it is custom image from compute gallery
	default:
		si:= strings.Split(imgId, "/")
		if si[6] == "Microsoft.Compute" && si[7] == "galleries"{
			gallery := si[8]
			image := si[10]
			version := si[12]
			subId := si[2]
			rgName := si[4]

			infraImg = infrav1.Image{
				ID:&imgId,
					ComputeGallery: &infrav1.AzureComputeGalleryImage{
						Gallery: gallery,
						Name: image,
						Version: version,
						ResourceGroup: &rgName,
						SubscriptionID: &subId,
					},
			}
    }else {log.Log.Error(errors.New("Failed in vmss.go"), "SDKImageToImage default case did not find Microsoft.Compute or galleries in the id")}
  }
	return infraImg
}
