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
	"fmt"
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
	var marketImg infrav1.AzureMarketplaceImage
	var computeImg infrav1.AzureComputeGalleryImage

  if imgId == "" {
		marketImg = infrav1.AzureMarketplaceImage{
			ImagePlan:       infrav1.ImagePlan{
				Publisher: to.String(sdkImageRef.Publisher),
				Offer:     to.String(sdkImageRef.Offer),
				SKU:       to.String(sdkImageRef.Sku),
			},
			Version:         to.String(sdkImageRef.Version),
			ThirdPartyImage: isThirdPartyImage,
		}
	}else{  //shared galleries are depricated only use compute gallery images with no image plan
		parts, err := ParseImageID(imgId)
		if err != nil {
			log.Log.Error(err, "Failed to parse image id")
		}

		for i := range(parts) {
			if strings.EqualFold(parts[i], "subscriptions"){
				computeImg.SubscriptionID = &parts[i + 1]
			}
			if strings.EqualFold(parts[i], "resourcegroups"){
				computeImg.ResourceGroup = &parts[i + 1]
			}
			if strings.EqualFold(parts[i], "galleries"){
				computeImg.Gallery = parts[i + 1]
			}
			if strings.EqualFold(parts[i], "images"){
				computeImg.Name = parts[i + 1]
			}
			if strings.EqualFold(parts[i], "versions"){
				computeImg.Version = parts[i + 1]
			}
		}
	}

	infraImg = infrav1.Image{
		ID:             &imgId,
		SharedGallery:  &infrav1.AzureSharedGalleryImage{},
		Marketplace:    &marketImg,
		ComputeGallery: &computeImg,
	}

		return infraImg
}

// ParseImageID parses a string to an instance of Image
func ParseImageID(id string) ([]string, error) {
	if len(id) == 0 {
		return nil, fmt.Errorf("invalid resource ID: id cannot be empty")
	}

	if !strings.HasPrefix(id, "/") {
		return nil, fmt.Errorf("invalid resource ID: resource id '%s' must start with '/'", id)
	}

	parts := splitStringAndOmitEmpty(id, "/")

	if len(parts) < 12 {
		return nil, fmt.Errorf("invalid resource ID: %s", id)
	}

	if !strings.EqualFold(parts[5], "Microsoft.Compute") || !strings.EqualFold(parts[6], "galleries"){
		return nil, fmt.Errorf("invalid image id type we only accept Microsoft.Compute/galleries %s", id)
	}

	if !strings.EqualFold(parts[0], "subscriptions") || parts[1] == "" {
		return nil, fmt.Errorf("invalid image ID subscription keyword or subscription is empty: %s", id)
	}

	if !strings.EqualFold(parts[2], "resourcegroups") || parts[3] == "" {
		return nil, fmt.Errorf("invalid image ID rg keyword missing or rg is empty: %s", id)
	}

	if !strings.EqualFold(parts[4], "providers"){
		return nil, fmt.Errorf("invalid image ID providers keyword missing: %s", id)
	}

	if !strings.EqualFold(parts[10], "versions") || parts[11] == "" {
		return nil, fmt.Errorf("invalid image ID versions keyword missing or version is empty %s", id)
	}

	return parts, nil
}

func splitStringAndOmitEmpty(v, sep string) []string {
	r := make([]string, 0)
	for _, s := range strings.Split(v, sep) {
		if len(s) == 0 {
			continue
		}
		r = append(r, s)
	}

	return r
}
