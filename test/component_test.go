package test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	awsHelper "github.com/cloudposse/test-helpers/pkg/aws"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ComponentSuite struct {
	helper.TestSuite
	KubernetesClient *kubernetes.Clientset
}

func (s *ComponentSuite) TestBasic() {
	const component = "eks/cloudwatch/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	// Get the EKS cluster ID from the dependency
	eksOptions := s.GetAtmosOptions("eks/cluster", stack, nil)
	clusterID := atmos.Output(s.T(), eksOptions, "eks_cluster_id")
	require.NotEmpty(s.T(), clusterID, "EKS cluster ID should not be empty")

	// Get the cluster and create Kubernetes client
	cluster := awsHelper.GetEksCluster(s.T(), context.Background(), awsRegion, clusterID)
	k8sConfig, err := awsHelper.NewK8SClientConfig(cluster)
	require.NoError(s.T(), err, "Failed to create Kubernetes config")
	require.NotNil(s.T(), k8sConfig)

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	require.NoError(s.T(), err, "Failed to create Kubernetes client")
	s.KubernetesClient = clientset

	options, _ := s.DeployAtmosComponent(s.T(), component, stack, nil)
	assert.NotNil(s.T(), options)

	// Get the priority class name from the output
	priorityClassName := atmos.Output(s.T(), options, "priority_class_name")
	require.NotEmpty(s.T(), priorityClassName, "Priority class name should not be empty")

	// Verify priority class exists and has correct name
	priorityClass, err := s.KubernetesClient.SchedulingV1().PriorityClasses().Get(context.Background(), priorityClassName, metav1.GetOptions{})
	assert.NoError(s.T(), err, "Failed to get priority class %s", priorityClassName)
	assert.Equal(s.T(), priorityClassName, priorityClass.Name, "Priority class name should match Terraform output")

	// Verify Fluent Bit uses the correct priority class
	fluentBit, err := s.KubernetesClient.AppsV1().DaemonSets("amazon-cloudwatch").Get(context.Background(), "fluent-bit", metav1.GetOptions{})
	assert.NoError(s.T(), err, "Failed to get Fluent Bit daemonset")
	assert.Equal(s.T(), priorityClassName, fluentBit.Spec.Template.Spec.PriorityClassName, "Fluent Bit should use the correct priority class")

	// Verify CloudWatch Agent uses the correct priority class
	cloudwatchAgent, err := s.KubernetesClient.AppsV1().DaemonSets("amazon-cloudwatch").Get(context.Background(), "cloudwatch-agent", metav1.GetOptions{})
	assert.NoError(s.T(), err, "Failed to get CloudWatch Agent daemonset")
	assert.Equal(s.T(), priorityClassName, cloudwatchAgent.Spec.Template.Spec.PriorityClassName, "CloudWatch Agent should use the correct priority class")

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
	defer s.DestroyAtmosComponent(s.T(), component, stack, nil)
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
