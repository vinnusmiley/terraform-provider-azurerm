package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/authorizers"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/file/directories"
)

type Client struct {
	// this is currently unexported since we only use it to look up the account key
	// we could export/use this in the future - but there's no point it being public
	// until that time
	accountsClient storage.AccountsClient
}

// NOTE: this temporarily diverges from the other clients until we move this client in here
// once we have this, can take an Options like everything else
func BuildClient(accountsClient storage.AccountsClient) *Client {
	return &Client{
		accountsClient: accountsClient,
	}
}

func (client Client) FindResourceGroup(ctx context.Context, accountName string) (*string, error) {
	accounts, err := client.accountsClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error listing Storage Accounts (to find Resource Group for %q): %s", accountName, err)
	}

	if accounts.Value == nil {
		return nil, nil
	}

	var resourceGroup *string
	for _, account := range *accounts.Value {
		if account.Name == nil || account.ID == nil {
			continue
		}

		if strings.EqualFold(accountName, *account.Name) {
			id, err := azure.ParseAzureResourceID(*account.ID)
			if err != nil {
				return nil, fmt.Errorf("Error parsing ID for Storage Account %q: %s", accountName, err)
			}

			resourceGroup = &id.ResourceGroup
			break
		}
	}

	return resourceGroup, nil
}

func (client Client) FileShareDirectoriesClient(ctx context.Context, resourceGroup, accountName string) (*directories.Client, error) {
	accountKey, err := client.findAccountKey(ctx, resourceGroup, accountName)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving Account Key: %s", err)
	}

	storageAuth := authorizers.NewSharedKeyLiteAuthorizer(accountName, *accountKey)
	directoriesClient := directories.New()
	directoriesClient.Client.Authorizer = storageAuth
	return &directoriesClient, nil
}

func (client Client) findAccountKey(ctx context.Context, resourceGroup, accountName string) (*string, error) {
	props, err := client.accountsClient.ListKeys(ctx, resourceGroup, accountName)
	if err != nil {
		return nil, fmt.Errorf("Error Listing Keys for Storage Account %q (Resource Group %q): %+v", accountName, resourceGroup, err)
	}

	if props.Keys == nil || len(*props.Keys) == 0 {
		return nil, fmt.Errorf("Keys were nil for Storage Account %q (Resource Group %q): %+v", accountName, resourceGroup, err)
	}

	keys := *props.Keys
	firstKey := keys[0].Value
	return firstKey, nil
}
