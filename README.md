# Azure Image Builder with GitHub Actions demo
Provides a sample project which uses the Azure Go SDK to setup and run Azure Image Builder. A sample GitHub Action is also provided to show how this process could be automated to run regularly.

Setup of necessary resources and creating the image template is done with the `create_all_resources` command. Running AIB is done with the `run_image_builder` command.

## What you'll need
* Create an Azure service principal with a secret
    * Take down `client id`, `client secret`, `tenant id`, `subscription id`
    * The service principal will need sufficient permissions to create/assign roles if running `create_all_resources`
* GitHub account
* Golang (if you want to run locally)

## Local setup
The only dependency is Golang. After installing Golang you can run the following to install all the necessary modules and compile the two sample commands.

```sh
# Install dependencies
go mod tidy

# Build the commands you need
go build ./cmd/run_image_builder
go build ./cmd/create_all_resources
```

The following environment variables must be set to run any of the commands:
* AZURE_SUBSCRIPTION_ID
* AZURE_TENANT_ID
* AZURE_CLIENT_ID
* AZURE_CLIENT_SECRET

## GitHub Action setup
The following secrets must be defined:
* AZURE_SUBSCRIPTION_ID
* AZURE_TENANT_ID
* AZURE_CLIENT_ID
* AZURE_CLIENT_SECRET

See [Using secrets in GitHub Actions](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions) for help with adding secrets.

The repository comes with a sample GitHub Action that can be used.

## Commands
Setup of necessary resources and creating the image template is done with the `create_all_resources` command. This will essentially do all the steps manually done in the [golden image tutorial](https://ubuntu.com/tutorials/how-to-create-a-golden-image-of-ubuntu-pro-20-04-fips-with-azure-image-builder#1-overview).

Once an image template is produced you can reuse that same template to keep creating updated versions of your golden image. To run Azure Image Builder with your image template use the `run_image_builder` command.

## Usage

### Configuration
Configuration is done via the flags passed to the commands. Not all possible options/configuration for Azure Image Builder are exposed, edit the code as needed to configure what is necessary for you.

To see what options are available
```sh
./create_all_resources --help
./run_image_builder --help
```

Along with the string flags, `create_all_resources` takes in three path flags which contain additional required configuration. In `config/` you'll find the corresponding configuration files which should be edited:
* `config/imageDefinitionProperties.json`: Defines the base image you are basing your golden image on.
* `config/customizations.json`: Defines what customizations you want done to your base image.
* `config/aibRolePermissions.json`: Defines the permissions of the managed identity used by Azure Image Builder. These permissions are scoped to the resource group created by `create_all_resources`. You most likely won't need to change this.

Note: these are the default paths but you can provide paths to different files via the `--imageProperties`, `--customizations` and `--rolePermissions` flags. See `./create_all_resources --help` for more information.

### Sample usage
```sh
./create_all_resources \
    --resourceGroup "aib-pipeline" \
    --galleryName "aibGallery" \
    --imageTemplateName "ubuntu_22_04" \
    --location "eastus" \
    --targetRegion "eastus" --targetRegion "westus" \
    --exportTemplate true
```

*Important*: it is not possible to accept the image terms with the SDK, so this step must be done with the Azure CLI. Just make sure to use the same sku, offer and publisher you set in `config/imageDefinitionProperties.json`
```sh
az vm image terms accept --plan <sku> --offer <offer> --publisher <publisher> --subscription <subID>
```

```sh
./run_image_builder \
    --templateName "ubuntu_22_04" \
    --resourceGroupName "aib-pipeline"
```
