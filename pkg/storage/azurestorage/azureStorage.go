package azurestorage

import (
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type AzureStorage struct {
	ServiceClient   *azblob.ServiceClient
	ContainerClient *azblob.ContainerClient
	Name            string
	Container       string
}

// CreateContainerClient creates a new Azure storage container client.
// If useAzureCliCredentials is true, it will use the Azure CLI credentials.
// Otherwise, it will use the default Azure credentials.
func NewAzureStorage(accountName string, container string, useAzureCliCredentials bool) (*AzureStorage, error) {
	var azureCredential azcore.TokenCredential
	var err error

	// Set Azure storage account connection.
	if useAzureCliCredentials {
		azureCredential, err = azidentity.NewAzureCLICredential(nil)
		if err != nil {
			return nil, fmt.Errorf("NewAzureStorage: failed to create Azure CLI credential: %w", err)
		}
		log.Println("Created Azure client using Azure CLI credentials")
	} else {
		azureCredential, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("NewAzureStorage: failed to create default Azure credential: %w", err)
		}
		log.Println("Created Azure client using NewDefaultAzureCredential")
	}

	serviceClient, err := azblob.NewServiceClient(fmt.Sprintf("https://%s.blob.core.windows.net/", accountName), azureCredential, nil)
	if err != nil {
		return nil, fmt.Errorf("NewAzureStorage: failed to create Azure blob client for storage account %s: %w", accountName, err)
	}

	containerClient, err := serviceClient.NewContainerClient(container)
	if err != nil {
		return nil, fmt.Errorf("NewAzureStorage: failed to create container client: %w", err)
	}

	log.Println("Created Azure service client for storage account", accountName)

	return &AzureStorage{
		ServiceClient:   serviceClient,
		ContainerClient: containerClient,
		Name:            accountName,
		Container:       container,
	}, nil
}

// Uploads the file held in io.Reader to the Blob named fileName
func (a *AzureStorage) UploadArtifact(ctx context.Context, fileName string, data io.Reader) error {
	blobClient, err := a.ContainerClient.NewBlockBlobClient(fileName)
	if err != nil {
		return fmt.Errorf("UploadArtifact: failed to create block blob client: %w", err)
	}

	if _, err := blobClient.UploadStream(ctx, data, azblob.UploadStreamOptions{}); err != nil {
		return fmt.Errorf("UploadArtifact: failed to upload stream to the %s container within the %s Azure Storage Account: %w", err)
	}

	return nil
}

// Find the last date that a blob with a name that follow the specified regexp has been created within the storage account.
func (a *AzureStorage) LastDateOfFileUploaded(ctx context.Context, fileNameRegex *regexp.Regexp) (*time.Time, error) {
	var lastDate *time.Time

	pager := a.ContainerClient.ListBlobsFlat(nil)
	for pager.NextPage(ctx) {
		containers := pager.PageResponse()

		for _, blob := range containers.Segment.BlobItems {
			if fileNameRegex.MatchString(*blob.Name) {
				str := fileNameRegex.FindStringSubmatch(*blob.Name)[1]
				date, err := time.Parse("2006-01-02", str)
				if err != nil {
					continue
				}

				if lastDate != nil && lastDate.Before(date) {
					lastDate = &date
				} else if lastDate == nil {
					lastDate = &date
				}
			}
		}
	}

	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("GetLastDateOfBlobUpload: failed to list blobs: %w", err)
	}

	return lastDate, nil
}
