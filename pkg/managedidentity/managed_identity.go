package managedidentity

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
)

type UserAssignedIdentityParams struct {
    ResourceGroup string
    Name string
    Location string
}

type IdentityData struct {
    Id string
    PrincipleId string
}

func EnsureUserManagedIdentity(subscriptionId string, cred azcore.TokenCredential, identityParams UserAssignedIdentityParams) (IdentityData, error) {
    ctx := context.Background()
    identityData := IdentityData{}

    clientFactory, err := armmsi.NewClientFactory(subscriptionId, cred, nil)
    if err != nil {
        fmt.Println("Failed to create client factory:", err)
        return identityData, err
    }
    identityClient := clientFactory.NewUserAssignedIdentitiesClient()

    getResponse, err := identityClient.Get(ctx, identityParams.ResourceGroup, identityParams.Name, nil)
    if err == nil {
        fmt.Println("Identity already exists:", identityParams.Name)
        identityData.Id = *getResponse.ID
        identityData.PrincipleId = *getResponse.Properties.PrincipalID
        return identityData, nil
    }

    return createUserManagedIdentity(*identityClient, identityParams)
}

func createUserManagedIdentity(client armmsi.UserAssignedIdentitiesClient, identityParams UserAssignedIdentityParams) (IdentityData, error) {
    ctx := context.Background()
    identityData := IdentityData{}
    identity := armmsi.Identity{
        Location: &identityParams.Location,
    }
    response, err := client.CreateOrUpdate(ctx, identityParams.ResourceGroup, identityParams.Name, identity, nil)
    if err != nil {
        fmt.Println("failed to finish the request:", err)
        return identityData, err
    }

    identityData.Id = *response.ID
    identityData.PrincipleId = *response.Properties.PrincipalID
    return identityData, nil
}
