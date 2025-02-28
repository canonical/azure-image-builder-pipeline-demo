package managedidentity

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
)

type UserAssignedIdentityParams struct {
	ResourceGroup string
	Name          string
	Location      string
}

type IdentityData struct {
	ID          string
	PrincipleID string
}

func EnsureUserManagedIdentity(subscriptionID string, cred azcore.TokenCredential, identityParams UserAssignedIdentityParams) (IdentityData, error) {
	ctx := context.Background()
	identityData := IdentityData{}

	clientFactory, err := armmsi.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return identityData, fmt.Errorf("failed to create client factory: %w", err)
	}
	identityClient := clientFactory.NewUserAssignedIdentitiesClient()

	getResponse, err := identityClient.Get(ctx, identityParams.ResourceGroup, identityParams.Name, nil)
	if err == nil {
		log.Println("Identity already exists:", identityParams.Name)
		identityData.ID = *getResponse.ID
		identityData.PrincipleID = *getResponse.Properties.PrincipalID
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
		return identityData, fmt.Errorf("failed to finish the request: %w", err)
	}

	identityData.ID = *response.ID
	identityData.PrincipleID = *response.Properties.PrincipalID
	return identityData, nil
}
