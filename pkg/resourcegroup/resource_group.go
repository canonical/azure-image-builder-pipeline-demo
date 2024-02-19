package resourcegroup

import(
    "context"
    "fmt"

    "github.com/Azure/azure-sdk-for-go/sdk/azcore"
    "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

type ResourceGroupParams struct {
    Name string
    Location string
}

func EnsureResourceGroup(subscriptionId string, cred azcore.TokenCredential, params ResourceGroupParams) (string, error) {
    ctx := context.Background()
    groupsClient, err := armresources.NewResourceGroupsClient(subscriptionId, cred, nil)

    if err != nil {
        fmt.Println("Error creating resource group client", err)
        return "", err
    }

    resp, err := groupsClient.Get(ctx, params.Name, nil)

    if err == nil {
        fmt.Println("Resource group already exists:", params.Name)
        return *resp.ID, nil
    }

    createResp, err := groupsClient.CreateOrUpdate(ctx, params.Name, armresources.ResourceGroup{ Location: &params.Location }, nil)
    if err != nil {
        fmt.Println("Error while creating group:", err)
        return "", err
    }

    return *createResp.ID, nil
}
