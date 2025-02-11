package kubernetes

import (
	"github.com/tsuru/tsuru/provision/provisiontest"
	provTypes "github.com/tsuru/tsuru/types/provision"
	check "gopkg.in/check.v1"
)

func (s *S) TestGetAppResourceRequirements(c *check.C) {
	a := provisiontest.NewFakeApp("myapp", "plat", 1)
	a.Memory = 10 * 1024
	a.MilliCPU = 1000
	clusterClient := &ClusterClient{
		Cluster: &provTypes.Cluster{},
	}

	type testCase struct {
		factors                requirementsFactors
		expectedLimitMemory    string
		expectedRequestsMemory string
		expectedLimitCPU       string
		expectedRequestsCPU    string
	}

	testsCases := []testCase{
		{
			factors: requirementsFactors{
				overCommit: 1,
			},
			expectedLimitMemory:    "10Ki",
			expectedRequestsMemory: "10Ki",
			expectedLimitCPU:       "1",
			expectedRequestsCPU:    "1",
		},
		{
			factors: requirementsFactors{
				overCommit: 2,
			},
			expectedLimitMemory:    "10Ki",
			expectedRequestsMemory: "5Ki",
			expectedLimitCPU:       "1",
			expectedRequestsCPU:    "500m",
		},

		{
			factors: requirementsFactors{
				overCommit:       1,
				cpuOverCommit:    3,
				memoryOverCommit: 2,
			},
			expectedLimitMemory:    "10Ki",
			expectedRequestsMemory: "5Ki",
			expectedLimitCPU:       "1",
			expectedRequestsCPU:    "333m",
		},

		{
			factors: requirementsFactors{
				cpuBurst:   1.1,
				overCommit: 1,
			},
			expectedLimitMemory:    "10Ki",
			expectedRequestsMemory: "10Ki",
			expectedLimitCPU:       "1100m",
			expectedRequestsCPU:    "1",
		},

		{
			factors: requirementsFactors{
				cpuBurst:   2,
				overCommit: 1,
			},
			expectedLimitMemory:    "10Ki",
			expectedRequestsMemory: "10Ki",
			expectedLimitCPU:       "2",
			expectedRequestsCPU:    "1",
		},

		{
			factors: requirementsFactors{
				cpuBurst:   2,
				overCommit: 2,
			},
			expectedLimitMemory:    "10Ki",
			expectedRequestsMemory: "5Ki",
			expectedLimitCPU:       "2",
			expectedRequestsCPU:    "500m",
		},
	}

	for _, testCase := range testsCases {
		requirements, err := appResourceRequirements(a, clusterClient, testCase.factors)
		c.Assert(err, check.IsNil)

		memoryLimits := requirements.Limits["memory"]
		c.Assert(memoryLimits.String(), check.Equals, testCase.expectedLimitMemory)

		memoryRequests := requirements.Requests["memory"]
		c.Assert(memoryRequests.String(), check.Equals, testCase.expectedRequestsMemory)

		cpuLimits := requirements.Limits["cpu"]
		c.Assert(cpuLimits.String(), check.Equals, testCase.expectedLimitCPU)

		cpuRequests := requirements.Requests["cpu"]
		c.Assert(cpuRequests.String(), check.Equals, testCase.expectedRequestsCPU)
	}
}
