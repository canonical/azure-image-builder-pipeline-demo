package resourcegroup

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

type Params struct {
	Name     string
	Location string
}

func EnsureResourceGroup(subscriptionID string, cred azcore.TokenCredential, params Params) (string, error) {
	ctx := context.Background()
	groupsClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)

	if err != nil {
		return "", fmt.Errorf("error creating resource group client: %w", err)
	}

	resp, err := groupsClient.Get(ctx, params.Name, nil)

	if err == nil {
		log.Println("Resource group already exists:", params.Name)
		return *resp.ID, nil
	}

	createResp, err := groupsClient.CreateOrUpdate(ctx, params.Name, armresources.ResourceGroup{Location: &params.Location}, nil)
	if err != nil {
		return "", fmt.Errorf("error while creating group: %w", err)
	}

	return *createResp.ID, nil
}
