package imagebuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/virtualmachineimagebuilder/armvirtualmachineimagebuilder/v2"
)

func StartImageBuilder(subscriptionID string, cred azcore.TokenCredential, resourceGroup string, imageTemplateName string) error {
	clientFactory, err := armvirtualmachineimagebuilder.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create client factory: %w", err)
	}

	ctx := context.Background()
	client := clientFactory.NewVirtualMachineImageTemplatesClient()
	poller, err := client.BeginRun(ctx, resourceGroup, imageTemplateName, nil)
	if err != nil {
		return fmt.Errorf("error running image template: %w", err)
	}

	if _, err = poller.PollUntilDone(ctx, nil); err != nil {
		return fmt.Errorf("error creating image template: %w", err)
	}

	return nil
}

func EnsureImageBuilderTemplate(subscriptionID string, cred azcore.TokenCredential, resourceGroup string, imageTemplateName string, imageTemplate armvirtualmachineimagebuilder.ImageTemplate) error {
	clientFactory, err := armvirtualmachineimagebuilder.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create client factory: %w", err)
	}

	client := clientFactory.NewVirtualMachineImageTemplatesClient()

	ctx := context.Background()
	_, err = client.Get(ctx, resourceGroup, imageTemplateName, nil)
	if err != nil {
		switch e := err.(type) {
		case *azcore.ResponseError:
			if e.StatusCode == 404 {
				log.Print("Creating image template: ", imageTemplateName)
				return createImageBuilderTemplate(*client, resourceGroup, imageTemplateName, imageTemplate)
			} else {
				return fmt.Errorf("error while retrieving image template: %w", e)
			}
		default:
			return fmt.Errorf("error while retrieving image template: %w", e)
		}
	}

	return nil
}

func createImageBuilderTemplate(client armvirtualmachineimagebuilder.VirtualMachineImageTemplatesClient, resourceGroup string, imageTemplateName string, imageTemplate armvirtualmachineimagebuilder.ImageTemplate) error {
	ctx := context.Background()
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, imageTemplateName, imageTemplate, nil)
	if err != nil {
		return fmt.Errorf("error creating image template: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("error creating image template: %w", err)
	}

	log.Printf("Created image template: %s %s", *resp.ID, *resp.Name)

	return nil
}

func BuildImageTemplate(identityID string, location string, properties armvirtualmachineimagebuilder.ImageTemplateProperties) armvirtualmachineimagebuilder.ImageTemplate {
	identityTemplate := BuildImageIdentityTemplate(identityID)
	template := armvirtualmachineimagebuilder.ImageTemplate{
		Identity:   &identityTemplate,
		Location:   &location,
		Properties: &properties,
	}

	return template
}

func BuildImageTemplateProperties(distribute armvirtualmachineimagebuilder.ImageTemplateDistributorClassification, source armvirtualmachineimagebuilder.ImageTemplateSourceClassification, customizations []armvirtualmachineimagebuilder.ImageTemplateCustomizerClassification) armvirtualmachineimagebuilder.ImageTemplateProperties {
	var distributeSlice []armvirtualmachineimagebuilder.ImageTemplateDistributorClassification
	distributeSlice = append(distributeSlice, distribute)

	properties := armvirtualmachineimagebuilder.ImageTemplateProperties{
		Distribute: distributeSlice,
		Source:     source,
		Customize:  customizations,
	}

	return properties
}

func BuildImageIdentityTemplate(identityID string) armvirtualmachineimagebuilder.ImageTemplateIdentity {
	identityType := armvirtualmachineimagebuilder.ResourceIdentityType("UserAssigned")
	identities := make(map[string]*armvirtualmachineimagebuilder.UserAssignedIdentity)
	identities[identityID] = &armvirtualmachineimagebuilder.UserAssignedIdentity{}

	identity := armvirtualmachineimagebuilder.ImageTemplateIdentity{
		Type:                   &identityType,
		UserAssignedIdentities: identities,
	}

	return identity
}

func BuildImageTemplateDistributor(imageID string, runOutputName string, targetRegionNames []string) armvirtualmachineimagebuilder.ImageTemplateDistributorClassification {
	var targetRegions []*armvirtualmachineimagebuilder.TargetRegion
	for _, regionName := range targetRegionNames {
		regionNameCopy := regionName
		region := armvirtualmachineimagebuilder.TargetRegion{Name: &regionNameCopy}
		targetRegions = append(targetRegions, &region)
	}
	distribute := armvirtualmachineimagebuilder.ImageTemplateSharedImageDistributor{
		GalleryImageID: &imageID,
		RunOutputName:  &runOutputName,
		TargetRegions:  targetRegions,
	}

	return &distribute
}

func BuildImageTemplateSource(offer string, publisher string, sku string, version string) armvirtualmachineimagebuilder.ImageTemplateSourceClassification {
	purchasePlanInfo := armvirtualmachineimagebuilder.PlatformImagePurchasePlan{
		PlanName:      &sku,
		PlanProduct:   &offer,
		PlanPublisher: &publisher,
	}
	source := armvirtualmachineimagebuilder.ImageTemplatePlatformImageSource{
		Offer:     &offer,
		Publisher: &publisher,
		SKU:       &sku,
		Version:   &version,
		PlanInfo:  &purchasePlanInfo,
	}

	return &source
}

func BuildImageTemplateCustomizationsFromFile(path string) ([]armvirtualmachineimagebuilder.ImageTemplateCustomizerClassification, error) {
	var customizations []armvirtualmachineimagebuilder.ImageTemplateCustomizerClassification
	data, err := os.ReadFile(path)
	if err != nil {
		return customizations, fmt.Errorf("error reading file: %s, %w", path, err)
	}

	var items []json.RawMessage
	err = json.Unmarshal(data, &items)
	if err != nil {
		return customizations, fmt.Errorf("error importing from json: %w", err)
	}

	for _, item := range items {
		var tempMap map[string]interface{}
		if err = json.Unmarshal(item, &tempMap); err != nil {
			return customizations, fmt.Errorf("error importing from json: %w", err)
		}

		switch tempMap["type"] {
		case "Shell":
			var obj armvirtualmachineimagebuilder.ImageTemplateShellCustomizer
			if err = json.Unmarshal(item, &obj); err != nil {
				return customizations, fmt.Errorf("error importing from json: %w", err)
			}
			customizations = append(customizations, &obj)
		case "File":
			var obj armvirtualmachineimagebuilder.ImageTemplateFileCustomizer
			if err = json.Unmarshal(item, &obj); err != nil {
				return customizations, fmt.Errorf("error importing from json: %w", err)
			}
			customizations = append(customizations, &obj)
		}
	}

	return customizations, nil
}

func ExportImageTemplateToFile(path string, template armvirtualmachineimagebuilder.ImageTemplate) error {
	jsonData, err := json.MarshalIndent(template, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	if _, err = file.Write(jsonData); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}
