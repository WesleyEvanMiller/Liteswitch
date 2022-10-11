package handler

import (
	"context"
	"errors"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
	"k8s.io/klog"
)

var parsedCloudType cloud.Configuration

func StopNodePool(config Config) error {

	os.Setenv("AZURE_CLIENT_ID", config.CloudCredentials.Username)
	os.Setenv("AZURE_CLIENT_SECRET", config.CloudCredentials.Password)
	os.Setenv("AZURE_TENANT_ID", config.CloudCredentials.Tenant)

	defaultCloudType := cloud.AzurePublic

	if config.CloudType == "AzureGovernment" {
		parsedCloudType = cloud.AzureGovernment
	} else if config.CloudType == "AzureChina" {
		parsedCloudType = cloud.AzureChina
	} else {
		parsedCloudType = defaultCloudType
	}

	credOptions := &azidentity.EnvironmentCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: parsedCloudType,
		},
	}

	cred, err := azidentity.NewEnvironmentCredential(credOptions)
	if err != nil {
		klog.Infof("failed to obtain a credential: %v", err)
		return err
	}

	clientOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: parsedCloudType,
		},
		DisableRPRegistration: false,
	}

	agentPoolClient, err := armcontainerservice.NewAgentPoolsClient(config.CloudGroupID, cred, clientOptions)
	if err != nil {
		return err
	}

	nodePool, err := agentPoolClient.Get(context.TODO(), config.NodePool.ClusterResourceGroup, config.NodePool.Cluster, config.NodePool.Name, nil)
	if err != nil {
		return err
	}

	if nodePool.Properties.PowerState.Code == &armcontainerservice.PossibleCodeValues()[1] {
		return nil
	}

	nodePool.Properties.PowerState.Code = &armcontainerservice.PossibleCodeValues()[1]

	currentProvisioningState := *nodePool.Properties.ProvisioningState

	if currentProvisioningState != "Succeeded" {
		provisioningErr := errors.New("Current State is" + currentProvisioningState)
		return provisioningErr
	}

	result, err := agentPoolClient.BeginCreateOrUpdate(context.TODO(), config.NodePool.ClusterResourceGroup, config.NodePool.Cluster, config.NodePool.Name, nodePool.AgentPool, nil)
	if err != nil {
		return err
	}

	klog.Info(result.Result(context.TODO()))

	return nil
}

func StartNodePool(config Config) error {

	os.Setenv("AZURE_CLIENT_ID", config.CloudCredentials.Username)
	os.Setenv("AZURE_CLIENT_SECRET", config.CloudCredentials.Password)
	os.Setenv("AZURE_TENANT_ID", config.CloudCredentials.Tenant)

	defaultCloudType := cloud.AzurePublic

	if config.CloudType == "AzureGovernment" {
		parsedCloudType = cloud.AzureGovernment
	} else if config.CloudType == "AzureChina" {
		parsedCloudType = cloud.AzureChina
	} else {
		parsedCloudType = defaultCloudType
	}

	credOptions := &azidentity.EnvironmentCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: parsedCloudType,
		},
	}

	cred, err := azidentity.NewEnvironmentCredential(credOptions)
	if err != nil {
		klog.Infof("failed to obtain a credential: %v", err)
		return err
	}

	clientOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: parsedCloudType,
		},
		DisableRPRegistration: false,
	}

	agentPoolClient, err := armcontainerservice.NewAgentPoolsClient(config.CloudGroupID, cred, clientOptions)
	if err != nil {
		return err
	}

	nodePool, err := agentPoolClient.Get(context.TODO(), config.NodePool.ClusterResourceGroup, config.NodePool.Cluster, config.NodePool.Name, nil)
	if err != nil {
		return err
	}

	if nodePool.Properties.PowerState.Code == &armcontainerservice.PossibleCodeValues()[0] {
		return nil
	}

	nodePool.Properties.PowerState.Code = &armcontainerservice.PossibleCodeValues()[0]

	currentProvisioningState := *nodePool.Properties.ProvisioningState

	if currentProvisioningState != "Succeeded" {
		provisioningErr := errors.New("Current State is" + currentProvisioningState)
		return provisioningErr
	}

	result, err := agentPoolClient.BeginCreateOrUpdate(context.TODO(), config.NodePool.ClusterResourceGroup, config.NodePool.Cluster, config.NodePool.Name, nodePool.AgentPool, nil)
	if err != nil {
		return err
	}

	klog.Info(result.Result(context.TODO()))

	return nil
}
