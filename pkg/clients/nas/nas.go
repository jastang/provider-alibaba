/*
Copyright 2021 The Crossplane Authors.

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

package nas

import (
	"context"

	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	sdk "github.com/alibabacloud-go/nas-20170626/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/pkg/errors"

	"github.com/crossplane/provider-alibaba/apis/nas/v1alpha1"
)

// ErrCodeNoSuchNASFileSystem is the error code "NoSuchNASFileSystem" returned by SDK
const (
	errFailedToCreateNASClient = "failed to crate NAS client"
	errCodeFileSystemNotExist  = "InvalidFileSystem.NotFound"
)

// ClientInterface create a client inferface
type ClientInterface interface {
	DescribeFileSystems(fileSystemID, fileSystemType, vpcID *string) (*sdk.DescribeFileSystemsResponse, error)
	CreateFileSystem(fs v1alpha1.NASFileSystemParameter) (*sdk.CreateFileSystemResponse, error)
	DeleteFileSystem(fileSystemID string) error
}

// SDKClient is the SDK client for NASFileSystem
type SDKClient struct {
	Client *sdk.Client
}

// NewClient will create NAS client
func NewClient(ctx context.Context, endpoint string, accessKeyID string, accessKeySecret string, securityToken string) (*SDKClient, error) {
	config := &openapi.Config{
		AccessKeyId:     &accessKeyID,
		AccessKeySecret: &accessKeySecret,
		SecurityToken:   &securityToken,
		Endpoint:        &endpoint,
	}
	client, err := sdk.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, errFailedToCreateNASClient)
	}
	return &SDKClient{Client: client}, nil
}

// DescribeFileSystems describes NAS FileSystem
func (c *SDKClient) DescribeFileSystems(fileSystemID, fileSystemType, vpcID *string) (*sdk.DescribeFileSystemsResponse, error) {
	describeFileSystemsRequest := &sdk.DescribeFileSystemsRequest{}
	if fileSystemID != nil {
		describeFileSystemsRequest.FileSystemId = tea.String(*fileSystemID)
	}
	if fileSystemType != nil {
		describeFileSystemsRequest.FileSystemType = tea.String(*fileSystemType)
	}
	if vpcID != nil {
		describeFileSystemsRequest.VpcId = tea.String(*vpcID)
	}
	fs, err := c.Client.DescribeFileSystems(describeFileSystemsRequest)
	if err != nil {
		return nil, err
	}
	return fs, nil
}

// CreateFileSystem creates NASFileSystem
func (c *SDKClient) CreateFileSystem(fs v1alpha1.NASFileSystemParameter) (*sdk.CreateFileSystemResponse, error) {
	createFileSystemRequest := &sdk.CreateFileSystemRequest{
		FileSystemType: fs.FileSystemType,
		ChargeType:     fs.ChargeType,
		VpcId:          fs.VpcID,
		VSwitchId:      fs.VSwitchID,
		StorageType:    fs.StorageType,
		ProtocolType:   fs.ProtocolType,
	}
	res, err := c.Client.CreateFileSystem(createFileSystemRequest)
	return res, err
}

// DeleteFileSystem deletes NASFileSystem
func (c *SDKClient) DeleteFileSystem(fileSystemID string) error {
	deleteFileSystemRequest := &sdk.DeleteFileSystemRequest{
		FileSystemId: tea.String(fileSystemID),
	}
	_, err := c.Client.DeleteFileSystem(deleteFileSystemRequest)
	return err
}

// GenerateObservation generates NASFileSystemObservation from fileSystem information
// When vpcID and vSwitchID are set, descriptionResponse.Body.FileSystems.FileSystem becomes 0, so we need to set fileSystemID
// first, not from descriptionResponse
func GenerateObservation(fileSystemID *string, descriptionResponse *sdk.DescribeFileSystemsResponse) v1alpha1.NASFileSystemObservation {
	observation := v1alpha1.NASFileSystemObservation{
		FileSystemID: *fileSystemID,
	}
	var domain string
	if len(descriptionResponse.Body.FileSystems.FileSystem) == 0 {
		return observation
	}
	if len(descriptionResponse.Body.FileSystems.FileSystem[0].MountTargets.MountTarget) == 0 {
		domain = ""
	} else {
		domain = *descriptionResponse.Body.FileSystems.FileSystem[0].MountTargets.MountTarget[0].MountTargetDomain
	}
	observation.MountTargetDomain = domain
	return observation
}

// IsUpdateToDate checks whether cr is up to date
func IsUpdateToDate(cr *v1alpha1.NASFileSystem, fsResponse *sdk.DescribeFileSystemsResponse) bool {
	if *fsResponse.Body.TotalCount == 0 {
		return false
	}
	fs := fsResponse.Body.FileSystems.FileSystem[0]

	if *cr.Spec.StorageType == *fs.StorageType && *cr.Spec.ProtocolType == *fs.ProtocolType {
		return true
	}
	return false
}

// IsNotFoundError helper function to test for SLS project not found error
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := errors.Cause(err).(*tea.SDKError); ok && (*e.Code == errCodeFileSystemNotExist) {
		return true
	}
	return false
}