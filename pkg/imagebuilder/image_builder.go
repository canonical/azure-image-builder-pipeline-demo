package imagebuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/virtualmachineimagebuilder/armvirtualmachineimagebuilder/v2"
)

func StartImageBuilder(subscriptionId string, cred azcore.TokenCredential, resourceGroup string, imageTemplateName string) error {
    clientFactory, err := armvirtualmachineimagebuilder.NewClientFactory(subscriptionId, cred, nil)
    if err != nil {
        return fmt.Errorf("Failed to create client factory: %w", err)
    }

    ctx := context.Background()
    client := clientFactory.NewVirtualMachineImageTemplatesClient()
    poller, err := client.BeginRun(ctx, resourceGroup, imageTemplateName, nil)
    if err != nil {
        return fmt.Errorf("Error running image template: %w", err)
    }

    _, err = poller.PollUntilDone(ctx, nil)
    if err != nil {
        return fmt.Errorf("Error creating image template: %w", err)
    }

    return nil
}

func EnsureImageBuilderTemplate(subscriptionId string, cred azcore.TokenCredential, resourceGroup string, imageTemplateName string, imageTemplate armvirtualmachineimagebuilder.ImageTemplate) error {
    clientFactory, err := armvirtualmachineimagebuilder.NewClientFactory(subscriptionId, cred, nil)
    if err != nil {
        return fmt.Errorf("Failed to create client factory: %w", err)
    }

    client := clientFactory.NewVirtualMachineImageTemplatesClient()

    ctx := context.Background()
    _, err = client.Get(ctx, resourceGroup, imageTemplateName, nil)
    if err != nil {
        switch e := err.(type) {
        case *azcore.ResponseError:
            if e.StatusCode == 404 {
                fmt.Println("Creating image template:", imageTemplateName)
                return createImageBuilderTemplate(*client, resourceGroup, imageTemplateName, imageTemplate)
            }else {
                return fmt.Errorf("Error while retrieving image template: %w", e)
            }
        default:
            return fmt.Errorf("Error while retrieving image template: %w", e)
        }
    }

    return nil
}

func createImageBuilderTemplate(client armvirtualmachineimagebuilder.VirtualMachineImageTemplatesClient, resourceGroup string, imageTemplateName string, imageTemplate armvirtualmachineimagebuilder.ImageTemplate) error {
    ctx := context.Background()
    poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, imageTemplateName, imageTemplate, nil)
    if err != nil {
        return fmt.Errorf("Error creating image template: %w", err)
    }

    resp, err := poller.PollUntilDone(ctx, nil)
    if err != nil {
        return fmt.Errorf("Error creating image template: %w", err)
    }

    fmt.Println("Created image template:", *resp.ID, *resp.Name)

    return nil
}

func BuildImageTemplate(identityId string, location string, properties armvirtualmachineimagebuilder.ImageTemplateProperties) armvirtualmachineimagebuilder.ImageTemplate {
    identityTemplate := BuildImageIdentityTemplate(identityId)
    template := armvirtualmachineimagebuilder.ImageTemplate{
        Identity: &identityTemplate,
        Location: &location,
        Properties: &properties,
    }

    return template
}

func BuildImageTemplateProperties(distribute armvirtualmachineimagebuilder.ImageTemplateDistributorClassification, source armvirtualmachineimagebuilder.ImageTemplateSourceClassification, customizations []armvirtualmachineimagebuilder.ImageTemplateCustomizerClassification) armvirtualmachineimagebuilder.ImageTemplateProperties {
    var distributeSlice []armvirtualmachineimagebuilder.ImageTemplateDistributorClassification
    distributeSlice = append(distributeSlice, distribute)

    properties := armvirtualmachineimagebuilder.ImageTemplateProperties{
        Distribute: distributeSlice,
        Source: source,
        Customize: customizations,
    }

    return properties
}

func BuildImageIdentityTemplate(identityId string) armvirtualmachineimagebuilder.ImageTemplateIdentity {
    identityType := armvirtualmachineimagebuilder.ResourceIdentityType("UserAssigned")
    identities := make(map[string]*armvirtualmachineimagebuilder.UserAssignedIdentity)
    identities[identityId] = &armvirtualmachineimagebuilder.UserAssignedIdentity{}

    identity := armvirtualmachineimagebuilder.ImageTemplateIdentity{
        Type: &identityType,
        UserAssignedIdentities: identities,
    }


    return identity
}

func BuildImageTemplateDistributor(imageId string, runOutputName string, targetRegionNames []string) armvirtualmachineimagebuilder.ImageTemplateDistributorClassification {
    var targetRegions []*armvirtualmachineimagebuilder.TargetRegion
    for _, regionName := range targetRegionNames {
        regionNameCopy := regionName
        region := armvirtualmachineimagebuilder.TargetRegion{Name: &regionNameCopy}
        targetRegions = append(targetRegions, &region)
    }
    distribute := armvirtualmachineimagebuilder.ImageTemplateSharedImageDistributor{
        GalleryImageID: &imageId,
        RunOutputName: &runOutputName,
        TargetRegions: targetRegions,
    }

    return &distribute
}

func BuildImageTemplateSource(offer string, publisher string, sku string, version string) armvirtualmachineimagebuilder.ImageTemplateSourceClassification {
    purchasePlanInfo := armvirtualmachineimagebuilder.PlatformImagePurchasePlan{
        PlanName: &sku,
        PlanProduct: &offer,
        PlanPublisher: &publisher,
    }
    source := armvirtualmachineimagebuilder.ImageTemplatePlatformImageSource{
        Offer: &offer,
        Publisher: &publisher,
        SKU: &sku,
        Version: &version,
        PlanInfo: &purchasePlanInfo,
    }

    return &source
}

func BuildImageTemplateCustomizationsFromFile(path string) ([]armvirtualmachineimagebuilder.ImageTemplateCustomizerClassification, error) {
    var customizations []armvirtualmachineimagebuilder.ImageTemplateCustomizerClassification
    data, err := os.ReadFile(path)
    if err != nil {
        fmt.Println("Error reading file:", err)
        return customizations, err
    }

    var items []json.RawMessage
    err = json.Unmarshal(data, &items)
    if err != nil {
        fmt.Println("Error importing from json:", err)
        return customizations, err
    }

    for _, item := range items {
        var tempMap map[string]interface{}
        err = json.Unmarshal(item, &tempMap)
        if err != nil {
            fmt.Println("Error importing from json:", err)
            return customizations, err
        }

        switch tempMap["type"] {
        case "Shell":
            var obj armvirtualmachineimagebuilder.ImageTemplateShellCustomizer
            err = json.Unmarshal(item, &obj)
            if err != nil {
                fmt.Println("Error importing from json:", err)
                return customizations, err
            }
            customizations = append(customizations, &obj)
        case "File":
            var obj armvirtualmachineimagebuilder.ImageTemplateFileCustomizer
            err = json.Unmarshal(item, &obj)
            if err != nil {
                fmt.Println("Error importing from json:", err)
                return customizations, err
            }
            customizations = append(customizations, &obj)
        }
    }

    return customizations, nil
}

func ExportImageTemplateToFile(path string, template armvirtualmachineimagebuilder.ImageTemplate) error {
    jsonData, err := json.MarshalIndent(template, "", "    ")
    if err != nil {
        fmt.Println("Error marshaling data:", err)
        return err
    }
    file, err := os.Create("generated.json")
    if err != nil {
        fmt.Println("Error creating file", err)
        return err
    }

    _, err = file.Write(jsonData)
    if err != nil {
        fmt.Println("Error writing to file:", err)
        return err
    }

    return nil
}
