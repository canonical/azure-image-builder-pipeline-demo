package role

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/google/uuid"
)

type RoleDefinitionParams struct {
    Name string
    Description string
    Scopes []string
}

func BuildRolePermissionsFromFile(path string) (armauthorization.Permission, error) {
    var permissions armauthorization.Permission
    rolePermissionData, err := os.ReadFile(path)
    if err != nil {
        fmt.Println("Error reading file:", err)
        return permissions, err
    }

    err = json.Unmarshal(rolePermissionData, &permissions)
    if err != nil {
        fmt.Println("Error setting permissions from input data:", err)
        return permissions, err
    }

    return permissions, nil
}

func BuildRoleProperties(params RoleDefinitionParams, permissions armauthorization.Permission) armauthorization.RoleDefinitionProperties {
    scopePointerSlice := make([]*string, len(params.Scopes))
    for i, str := range params.Scopes {
        scopePointerSlice[i] = &str
    }

    permissionsPointerSlice := make([]*armauthorization.Permission, 1)
    permissionsPointerSlice[0] = &permissions

    roleProperties := armauthorization.RoleDefinitionProperties{
        RoleName: &params.Name,
        Description: &params.Description,
        AssignableScopes: scopePointerSlice,
        Permissions: permissionsPointerSlice,
    }

    return roleProperties
}

func EnsureRoleDefinition(subscriptionId string, cred azcore.TokenCredential, properties armauthorization.RoleDefinitionProperties, scope string) (string, error) {
    clientFactory, err := armauthorization.NewClientFactory(subscriptionId, cred, nil)
    if err != nil {
        fmt.Println("Failed to create client factory:", err)
        return "", err
    }
    roleDefinitionClient := clientFactory.NewRoleDefinitionsClient()

    roleId, err := findRoleDefinition(*roleDefinitionClient, subscriptionId, properties, scope)
    if err != nil {
        fmt.Println("Failed to find existing role:", err)
        return "", err
    }

    if roleId == "" {
        fmt.Println("Creating role")
        return createRoleDefinition(*roleDefinitionClient, subscriptionId, properties, scope)
    } else {
        return roleId, nil
    }
}

func findRoleDefinition(client armauthorization.RoleDefinitionsClient, subscriptionId string, properties armauthorization.RoleDefinitionProperties, scope string) (string, error) {
    ctx := context.Background()
    roleName := *properties.RoleName
    pager := client.NewListPager(scope, nil)
    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            return "", fmt.Errorf("Error retrieving role definition page: %w", err)
        }

        for _, roleDef := range page.Value {
            if *roleDef.Properties.RoleName == roleName {
                fmt.Printf("Found role: %s, %s\n", *roleDef.Properties.RoleName, *roleDef.ID)
                return *roleDef.ID, nil
            }
        }
    }

    fmt.Println("Unable to find role:", roleName)
    return "", nil
}

func createRoleDefinition(client armauthorization.RoleDefinitionsClient, subscriptionId string, properties armauthorization.RoleDefinitionProperties, scope string) (string, error) {
    ctx := context.Background()
    roleDefinition := armauthorization.RoleDefinition{
        Properties: &properties,
    }

    roleId := uuid.New().String()

    resp, err := client.CreateOrUpdate(ctx, scope, roleId, roleDefinition, nil)
    if err != nil {
        return "", fmt.Errorf("Failed to create role: %w", err)
    }

    return *resp.ID, nil
}

func EnsureRoleAssignment(subscriptionId string, cred azcore.TokenCredential, scope, principalId, roleId string) (string, error) {
    clientFactory, err := armauthorization.NewClientFactory(subscriptionId, cred, nil)
    if err != nil {
        return "", fmt.Errorf("Failed to create client factory: %w", err)
    }

    client := clientFactory.NewRoleAssignmentsClient()

    assignmentId, err := findRoleAssignment(*client, scope, principalId, roleId)
    if err != nil {
        return "", fmt.Errorf("Error finding role assignment: %w", err)
    }

    if assignmentId == "" {
        fmt.Println("Creating role assignment")
        timeout := 5 * time.Minute
        waitTime := 15 * time.Second
        return createRoleAssignmentWithRetries(*client, scope, principalId, roleId, timeout, waitTime)
    } else {
        return assignmentId, nil
    }
}

func findRoleAssignment(client armauthorization.RoleAssignmentsClient, scope, principalId, roleId string) (string, error) {
    ctx := context.Background()
    pager := client.NewListForScopePager(scope, nil)
    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            return "", fmt.Errorf("Error retrieving role assignment page: %w", err)
        }

        for _, roleAssignment := range page.Value {
            properties := *roleAssignment.Properties
            if (*properties.PrincipalID == principalId) && (*properties.RoleDefinitionID == roleId) {
                return *roleAssignment.ID, nil
            }
        }
    }

    return "", nil
}

func createRoleAssignmentWithRetries(client armauthorization.RoleAssignmentsClient, scope, principalId, roleId string, timeout, waitTime time.Duration) (string, error) {
    deadline := time.Now().Add(timeout)

    for {
        id, err := createRoleAssignment(client, scope, principalId, roleId)
        if err == nil {
            return id, err
        }

        if time.Now().After(deadline) {
            return "", fmt.Errorf("Timeout after %s while creating role assignment: %w", timeout, err)
        }

        time.Sleep(waitTime)
    }
}


func createRoleAssignment(client armauthorization.RoleAssignmentsClient, scope, principalId, roleId string) (string, error) {
    ctx := context.Background()
    properties := armauthorization.RoleAssignmentProperties{
        PrincipalID: &principalId,
        RoleDefinitionID: &roleId,
    }
    parameters := armauthorization.RoleAssignmentCreateParameters{
        Properties: &properties,
    }
    assignmentId := uuid.New().String()
    resp, err := client.Create(ctx, scope, assignmentId, parameters, nil)
    if err != nil {
        return "", fmt.Errorf("Failed to assign role: %w", err)
    }

    return *resp.ID, nil
}
