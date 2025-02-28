package role

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/google/uuid"
)

type DefinitionParams struct {
	Name        string
	Description string
	Scopes      []string
}

func BuildRolePermissionsFromFile(path string) (armauthorization.Permission, error) {
	var permissions armauthorization.Permission
	rolePermissionData, err := os.ReadFile(path)
	if err != nil {
		return permissions, fmt.Errorf("error reading file: %w", err)
	}

	if err = json.Unmarshal(rolePermissionData, &permissions); err != nil {
		return permissions, fmt.Errorf("error setting permissions from input data: %w", err)
	}

	return permissions, nil
}

func BuildRoleProperties(params DefinitionParams, permissions armauthorization.Permission) armauthorization.RoleDefinitionProperties {
	scopePointerSlice := make([]*string, len(params.Scopes))
	for i, str := range params.Scopes {
		scopePointerSlice[i] = &str
	}

	permissionsPointerSlice := make([]*armauthorization.Permission, 1)
	permissionsPointerSlice[0] = &permissions

	roleProperties := armauthorization.RoleDefinitionProperties{
		RoleName:         &params.Name,
		Description:      &params.Description,
		AssignableScopes: scopePointerSlice,
		Permissions:      permissionsPointerSlice,
	}

	return roleProperties
}

func EnsureRoleDefinition(subscriptionID string, cred azcore.TokenCredential, properties armauthorization.RoleDefinitionProperties, scope string) (string, error) {
	clientFactory, err := armauthorization.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create client factory: %w", err)
	}
	roleDefinitionClient := clientFactory.NewRoleDefinitionsClient()

	roleID, err := findRoleDefinition(*roleDefinitionClient, properties, scope)
	if err != nil {
		return "", fmt.Errorf("failed to find existing role: %w", err)
	}

	if roleID == "" {
		log.Println("Creating role")
		return createRoleDefinition(*roleDefinitionClient, properties, scope)
	}

	return roleID, nil
}

func findRoleDefinition(client armauthorization.RoleDefinitionsClient, properties armauthorization.RoleDefinitionProperties, scope string) (string, error) {
	ctx := context.Background()
	roleName := *properties.RoleName
	pager := client.NewListPager(scope, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("error retrieving role definition page: %w", err)
		}

		for _, roleDef := range page.Value {
			if *roleDef.Properties.RoleName == roleName {
				log.Printf("Found role: %s, %s\n", *roleDef.Properties.RoleName, *roleDef.ID)
				return *roleDef.ID, nil
			}
		}
	}

	log.Println("Unable to find role:", roleName)
	return "", nil
}

func createRoleDefinition(client armauthorization.RoleDefinitionsClient, properties armauthorization.RoleDefinitionProperties, scope string) (string, error) {
	ctx := context.Background()
	roleDefinition := armauthorization.RoleDefinition{
		Properties: &properties,
	}

	roleID := uuid.New().String()

	resp, err := client.CreateOrUpdate(ctx, scope, roleID, roleDefinition, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create role: %w", err)
	}

	return *resp.ID, nil
}

func EnsureRoleAssignment(subscriptionID string, cred azcore.TokenCredential, scope, principalID, roleID string) (string, error) {
	clientFactory, err := armauthorization.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create client factory: %w", err)
	}

	client := clientFactory.NewRoleAssignmentsClient()

	assignmentID, err := findRoleAssignment(*client, scope, principalID, roleID)
	if err != nil {
		return "", fmt.Errorf("error finding role assignment: %w", err)
	}

	if assignmentID == "" {
		log.Println("Creating role assignment")
		timeout := 5 * time.Minute
		waitTime := 15 * time.Second
		return createRoleAssignmentWithRetries(*client, scope, principalID, roleID, timeout, waitTime)
	}

	return assignmentID, nil
}

func findRoleAssignment(client armauthorization.RoleAssignmentsClient, scope, principalID, roleID string) (string, error) {
	ctx := context.Background()
	pager := client.NewListForScopePager(scope, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("error retrieving role assignment page: %w", err)
		}

		for _, roleAssignment := range page.Value {
			properties := *roleAssignment.Properties
			if (*properties.PrincipalID == principalID) && (*properties.RoleDefinitionID == roleID) {
				return *roleAssignment.ID, nil
			}
		}
	}

	return "", nil
}

func createRoleAssignmentWithRetries(client armauthorization.RoleAssignmentsClient, scope, principalID, roleID string, timeout, waitTime time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for {
		id, err := createRoleAssignment(client, scope, principalID, roleID)
		if err == nil {
			return id, err
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout after %s while creating role assignment: %w", timeout, err)
		}

		time.Sleep(waitTime)
	}
}

func createRoleAssignment(client armauthorization.RoleAssignmentsClient, scope, principalID, roleID string) (string, error) {
	ctx := context.Background()
	properties := armauthorization.RoleAssignmentProperties{
		PrincipalID:      &principalID,
		RoleDefinitionID: &roleID,
	}
	parameters := armauthorization.RoleAssignmentCreateParameters{
		Properties: &properties,
	}
	assignmentID := uuid.New().String()
	resp, err := client.Create(ctx, scope, assignmentID, parameters, nil)
	if err != nil {
		return "", fmt.Errorf("failed to assign role: %w", err)
	}

	return *resp.ID, nil
}
