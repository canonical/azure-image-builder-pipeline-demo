package imagedefinition

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
)

func BuildImagePropertiesFromFile(path string) (armcompute.GalleryImageProperties, error) {
    var properties armcompute.GalleryImageProperties
    propertiesData, err := os.ReadFile(path)

    if err != nil {
        fmt.Println("Error reading file:", err)
        return properties, err
    }

    err = json.Unmarshal(propertiesData, &properties)
    if err != nil {
        fmt.Println("Error importing from json:", err)
        return properties, err
    }

    return properties, nil
}

func EnsureImageDefinition(subscriptionId string, cred azcore.TokenCredential, resourceGroup string, galleryName string, imageName string, imageProperties armcompute.GalleryImageProperties, location string) (string, error) {
    clientFactory, err := armcompute.NewClientFactory(subscriptionId, cred, nil)
    if err != nil {
        return "", fmt.Errorf("Failed to create client factory: %w", err)
    }
    client := clientFactory.NewGalleryImagesClient()
    imageId, err := findImageDefinition(*client, resourceGroup, galleryName, imageName)
    if err != nil {
        switch e := err.(type) {
        case *azcore.ResponseError:
            if e.StatusCode == 404 {
                fmt.Println("Creating image definition:", imageName)
                return createImageDefinition(*client, resourceGroup, galleryName, imageName, imageProperties, location)
            }else {
                return "", fmt.Errorf("Error while retrieving image gallery: %w", e)
            }
        default:
            return "", fmt.Errorf("Error while retrieving image gallery: %w", e)
        }
    }

    return imageId, nil
}

func findImageDefinition(client armcompute.GalleryImagesClient, resourceGroup string, galleryName string, imageName string) (string, error) {
    ctx := context.Background()

    resp, err := client.Get(ctx, resourceGroup, galleryName, imageName, nil)
    if err != nil {
        return "", err
    }

    return *resp.ID, nil
}

func createImageDefinition(client armcompute.GalleryImagesClient, resourceGroup string, galleryName string, imageName string, imageProperties armcompute.GalleryImageProperties, location string) (string, error) {
    ctx := context.Background()
    galleryImage := armcompute.GalleryImage{
        Location: &location,
        Properties: &imageProperties,
    }

    poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, galleryName, imageName, galleryImage, nil)
    if err != nil {
        return "", fmt.Errorf("Error creating image definition: %w", err)
    }

    pollCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Minute)
    defer cancel()

    resp, err := poller.PollUntilDone(pollCtx, nil)
    if err != nil {
        if err == context.DeadlineExceeded {
            return "", fmt.Errorf("Polling timeout exceeded: %w", err)
        } else {
            return "", fmt.Errorf("Error while creating image definition: %w", err)
        }
    }

    return *resp.ID, nil
}
