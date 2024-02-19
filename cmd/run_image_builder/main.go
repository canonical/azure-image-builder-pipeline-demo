package main

import (
	"aib-pipeline-demo/pkg/imagebuilder"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name:  "run_image_builder",
        Usage: "Trigger Azure Image Builder with an existing template.",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "templateName",
                Aliases: []string{"n"},
                Usage:   "The image template name",
                Required: true,
            },
            &cli.StringFlag{
                Name:    "subscriptionID",
                Aliases: []string{"s"},
                Usage:   "Azure subscription ID",
                EnvVars: []string{"AZURE_SUBSCRIPTION_ID"},
            },
            &cli.StringFlag{
                Name:    "resourceGroupName",
                Aliases: []string{"g"},
                Usage:   "Azure resource group name",
                Required: true,
            },
        },
        Before: func(c *cli.Context) error {
            if c.String("subscriptionID") == "" {
                return cli.Exit("Error: the --subscriptionID flag or AZURE_SUBSCRIPTION_ID environment variable is required", 1)
            }
            return nil
        },
        Action: runImageBuilder,
    }

    err := app.Run(os.Args)
    if err != nil {
        log.Fatal(err)
    }
}

func runImageBuilder(c *cli.Context) error {
    imageTemplateName := c.String("templateName")
    subscriptionId := c.String("subscriptionID")
    resourceGroupName := c.String("resourceGroupName")

    cred, err := azidentity.NewEnvironmentCredential(nil)

    if err != nil {
        fmt.Println("Failed to setup credentials:", err)
        return err
    }

    fmt.Println("Starting image builder for template:", imageTemplateName)
    err = imagebuilder.StartImageBuilder(subscriptionId, cred, resourceGroupName, imageTemplateName)
    if err != nil {
        fmt.Println("Error running image builder:", err)
        return err
    }

    fmt.Println("Completed image build", imageTemplateName)

    return nil
}
