package test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ComponentSuite struct {
	helper.TestSuite
}

func (s *ComponentSuite) TestBasic() {
	const component = "eks/cloudwatch"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	defer s.DestroyAtmosComponent(s.T(), component, stack, nil)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, nil)
	assert.NotNil(s.T(), options)

	// Get the EKS cluster ID from the dependency
	eksOptions := s.GetAtmosOptions("eks/cluster", stack, nil)
	clusterID := atmos.Output(s.T(), eksOptions, "eks_cluster_id")
	require.NotEmpty(s.T(), clusterID, "EKS cluster ID should not be empty")

	// Validate CloudWatch log groups
	logGroupNames := atmos.OutputList(s.T(), options, "cloudwatch_log_group_names")
	require.NotEmpty(s.T(), logGroupNames, "CloudWatch log group names should not be empty")

	// Create AWS clients
	awsConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(awsRegion))
	require.NoError(s.T(), err, "Failed to load AWS config")

	cloudwatchClient := cloudwatchlogs.NewFromConfig(awsConfig)
	iamClient := iam.NewFromConfig(awsConfig)

	// Verify each log group exists
	for _, logGroupName := range logGroupNames {
		_, err := cloudwatchClient.DescribeLogGroups(context.Background(), &cloudwatchlogs.DescribeLogGroupsInput{
			LogGroupNamePrefix: &logGroupName,
		})
		assert.NoError(s.T(), err, "Failed to get log group %s", logGroupName)
	}

	// Get the node group role names from the EKS cluster
	nodeGroupRoleNames := atmos.OutputList(s.T(), eksOptions, "eks_node_group_role_names")
	require.NotEmpty(s.T(), nodeGroupRoleNames, "Node group role names should not be empty")

	// Verify IAM role policy attachments
	for _, roleName := range nodeGroupRoleNames {
		attachedPolicies, err := iamClient.ListAttachedRolePolicies(context.Background(), &iam.ListAttachedRolePoliciesInput{
			RoleName: &roleName,
		})
		assert.NoError(s.T(), err, "Failed to get attached policies for role %s", roleName)

		// Check if CloudWatchAgentServerPolicy is attached
		found := false
		for _, policy := range attachedPolicies.AttachedPolicies {
			if *policy.PolicyName == "CloudWatchAgentServerPolicy" {
				found = true
				break
			}
		}
		assert.True(s.T(), found, "CloudWatchAgentServerPolicy should be attached to role %s", roleName)
	}

	// Verify Helm release metadata
	metadata := atmos.Output(s.T(), options, "metadata")
	assert.NotEmpty(s.T(), metadata, "Helm release metadata should not be empty")

	s.DriftTest(component, stack, nil)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "eks/cloudwatch/disabled"
	const stack = "default-test"
	s.VerifyEnabledFlag(component, stack, nil)
}

func (s *ComponentSuite) SetupSuite() {
	s.TestSuite.InitConfig()
	s.TestSuite.Config.ComponentDestDir = "components/terraform/eks/cloudwatch"
	s.TestSuite.SetupSuite()
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	suite.AddDependency(t, "vpc", "default-test", nil)
	suite.AddDependency(t, "eks/cluster", "default-test", nil)
	helper.Run(t, suite)
}
