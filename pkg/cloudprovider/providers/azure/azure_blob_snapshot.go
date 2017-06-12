/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"fmt"
	"time"
)

// CreateSnapshot creates a blob snapshot
func (az *Cloud) CreateSnapshot(uri string) (*time.Time, error) {
	accountName, blobName, err := az.getBlobNameAndAccountFromURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vhd URI %v", err)
	}
	accountKey, err := az.getStorageAccesskey(accountName)
	if err != nil {
		return nil, fmt.Errorf("no key for storage account %s, err %v", accountName, err)
	}
	blobClient, err := az.getBlobClient(accountName, accountKey)
	if err != nil {
		return nil, err
	}
	return blobClient.SnapshotBlob(vhdContainerName, blobName, 0, nil)
}
