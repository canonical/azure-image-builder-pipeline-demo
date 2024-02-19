package imagegallery

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
)


func EnsureImageGallery(subscriptionId string, cred azcore.TokenCredential, resourceGroup string, galleryName string, location string) error {
    clientFactory, err := armcompute.NewClientFactory(subscriptionId, cred, nil)
    if err != nil {
        return fmt.Errorf("Failed to create client factory: %w", err)
    }
    client := clientFactory.NewGalleriesClient()

    err = findImageGallery(*client, resourceGroup, galleryName)
    if err != nil {
        switch e := err.(type) {
        case *azcore.ResponseError:
            if e.StatusCode == 404 {
                fmt.Println("Creating image gallery:", galleryName)
                return createImageGallery(*client, resourceGroup, galleryName, location)
            }else {
                return fmt.Errorf("Error while retrieving image gallery: %w", e)
            }
        default:
            return fmt.Errorf("Error while retrieving image gallery: %w", e)
        }
    }

    return nil
}

func createImageGallery(client armcompute.GalleriesClient, resourceGroup string, galleryName string, location string) error {
    ctx := context.Background()
    gallery := armcompute.Gallery{
        Location: &location,
    }
    poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, galleryName, gallery, nil)
    if err != nil {
        return fmt.Errorf("Error creating gallery: %w", err)
    }

    pollCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Minute)
    defer cancel()

    _, err = poller.PollUntilDone(pollCtx, nil)
    if err != nil {
        if err == context.DeadlineExceeded {
            return fmt.Errorf("Polling timeout exceeded: %w", err)
        } else {
            return fmt.Errorf("Error while creating gallery: %w", err)
        }
    }

    return nil
}


func findImageGallery(client armcompute.GalleriesClient, resourceGroup string, galleryName string) error {
    ctx := context.Background()
    _, err := client.Get(ctx, resourceGroup, galleryName, nil)
    if err != nil {
        return err
    }

    return nil
}
