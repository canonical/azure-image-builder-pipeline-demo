package main

import (
	"aib-pipeline-demo/pkg/imagebuilder"
	"aib-pipeline-demo/pkg/imagedefinition"
	"aib-pipeline-demo/pkg/imagegallery"
	"aib-pipeline-demo/pkg/managedidentity"
	"aib-pipeline-demo/pkg/resourcegroup"
	"aib-pipeline-demo/pkg/role"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name:  "create_all_resources",
        Usage: "Create an image template and all required resources to use Azure Image Builder",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "subscriptionID",
                Aliases: []string{"s"},
                Usage:   "Azure subscription ID",
                EnvVars: []string{"AZURE_SUBSCRIPTION_ID"},
            },
            &cli.StringFlag{
                Name:    "resourceGroup",
                Aliases: []string{"g"},
                Usage:   "Azure resource group name",
                Required: true,
            },
            &cli.StringFlag{
                Name:    "location",
                Aliases: []string{"l"},
                Usage:   "Location in which to deploy resources",
                Required: true,
            },
            &cli.StringFlag{
                Name:    "imageTemplateName",
                Usage:   "Name of the image template to create",
                Required: true,
            },
            &cli.StringFlag{
                Name:    "runOutputName",
                Usage:   "The Azure Image Builder output name",
                Value:   "aibDemoOutput",
            },
            &cli.StringFlag{
                Name:    "imageName",
                Usage:   "The name of the image definition to create",
                Value:   "aibDemoImage",
            },
            &cli.StringFlag{
                Name:    "galleryName",
                Usage:   "The name of the image gallery to create",
                Required: true,
            },
            &cli.StringSliceFlag{
                Name:    "targetRegion",
                Aliases: []string{"r"},
                Usage:   "A region to replicate the produced image to.",
                Required: true,
            },
            &cli.PathFlag{
                Name:    "rolePermissions",
                Value:   "./config/aibRolePermissions.json",
                Usage:   "Path to the role permissions file",
            },
            &cli.PathFlag{
                Name:    "imageProperties",
                Value:   "./config/imageDefinitionProperties.json",
                Usage:   "Path to the image definitions properties file",
            },
            &cli.PathFlag{
                Name:    "customizations",
                Value:   "./config/customizations.json",
                Usage:   "Path to the image template customizations file",
            },
            &cli.BoolFlag{
                Name:    "exportTemplate",
                Usage:   "Whether the raw iamge template data should be exported",
                Value:   false,
            },
            &cli.PathFlag{
                Name:    "exportPath",
                Value:   "generatedTemplate.json",
                Usage:   "Path to export the image template to if enabled",
            },

        },
        Before: func(c *cli.Context) error {
            if c.String("subscriptionID") == "" {
                return cli.Exit("Error: the --subscriptionID flag or AZURE_SUBSCRIPTION_ID environment variable is required", 1)
            }
            return nil
        },
        Action: createAllResources,
    }

    err := app.Run(os.Args)
    if err != nil {
        log.Fatal(err)
    }
}

func createAllResources(c *cli.Context) error {
    subscriptionId := c.String("subscriptionID")
    resourceGroupName := c.String("resourceGroup")
    location := c.String("location")
    imageTemplateName := c.String("imageTemplateName")
    galleryName := c.String("galleryName")
    imageName := c.String("imageName")

    targetRegions := c.StringSlice("targetRegion")
    runOutputName := c.String("runOutputName")

    rolePermissionsFile := c.Path("rolePermissions")
    imagePropertiesFile := c.Path("imageProperties")
    customizationsFile := c.Path("customizations")

    exportTemplate := c.Bool("exportTemplate")
    exportPath := c.Path("exportPath")

    cred, err := azidentity.NewEnvironmentCredential(nil)

    if err != nil {
        fmt.Println("Failed to setup credentials:", err)
        return err
    }

    resourceGroupParams := resourcegroup.ResourceGroupParams{
        Name: resourceGroupName,
        Location: location,
    }
    groupId, err := resourcegroup.EnsureResourceGroup(subscriptionId, cred, resourceGroupParams)
    if err != nil {
        fmt.Println("Error ensuring resource group:", err)
        return err
    }

    identityParams := managedidentity.UserAssignedIdentityParams{
        Name: "aibUserIdentity",
        ResourceGroup: resourceGroupName,
        Location: location,
    }
    identityData, err := managedidentity.EnsureUserManagedIdentity(subscriptionId, cred, identityParams)
    if err != nil {
        fmt.Println("Failed to ensure user managed identity exists:", err)
        return err
    }

    permissions, err := role.BuildRolePermissionsFromFile(rolePermissionsFile)
    if err != nil {
        fmt.Println("Error importing role permissions from file:", err)
        return err
    }

    roleParams := role.RoleDefinitionParams{
        Name: "AIB Role Definition",
        Description: "Role to give Azure Image Builder access to the necessary resources.",
        Scopes: []string{groupId},
    }

    roleProperties := role.BuildRoleProperties(roleParams, permissions)

    fmt.Printf("Identity ID: %v\n", identityData)
    roleId, err := role.EnsureRoleDefinition(subscriptionId, cred, roleProperties, groupId)
    if err != nil {
        fmt.Println("Error ensuring role:", err)
        return err
    }
    fmt.Println("Role ID:", roleId)

    _, err = role.EnsureRoleAssignment(subscriptionId, cred, groupId, identityData.PrincipleId, roleId)
    if err != nil {
        fmt.Println("Error assigning role:", err)
        return err
    }

    err = imagegallery.EnsureImageGallery(subscriptionId, cred, resourceGroupName, galleryName, location)
    if err != nil {
        fmt.Println("Error ensuring shared image gallery:", err)
        return err
    }


    imageProperties, err := imagedefinition.BuildImagePropertiesFromFile(imagePropertiesFile)
    if err != nil {
        fmt.Println("Error getting image properties:", err)
        return err
    }

    imageId, err := imagedefinition.EnsureImageDefinition(subscriptionId, cred, resourceGroupName, galleryName, imageName, imageProperties, location)
    if err != nil {
        fmt.Println("Error ensuring image definition:", err)
        return err
    }

    distributeTemplate := imagebuilder.BuildImageTemplateDistributor(imageId, runOutputName, targetRegions)
    sourceTemplate := imagebuilder.BuildImageTemplateSource(*imageProperties.Identifier.Offer, *imageProperties.Identifier.Publisher, *imageProperties.Identifier.SKU, "latest")
    imageTemplateCustomizations, err := imagebuilder.BuildImageTemplateCustomizationsFromFile(customizationsFile)
    if err != nil {
        fmt.Println("Error importing customizations:", err)
        return err
    }

    imageTemplateProperties := imagebuilder.BuildImageTemplateProperties(distributeTemplate, sourceTemplate, imageTemplateCustomizations)
    imageTemplate := imagebuilder.BuildImageTemplate(identityData.Id, location, imageTemplateProperties)

    if exportTemplate {
        imagebuilder.ExportImageTemplateToFile(exportPath, imageTemplate)
        if err != nil {
            fmt.Println("Error exporting image template to file:", err)
            return err
        }
    }

    err = imagebuilder.EnsureImageBuilderTemplate(subscriptionId, cred, resourceGroupName, imageTemplateName, imageTemplate)
    if err != nil {
        fmt.Println("Error ensuring image builder template:", err)
        return err
    }

    return nil
}
