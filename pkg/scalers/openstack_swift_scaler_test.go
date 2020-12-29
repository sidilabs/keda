package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseSwiftMetadataTestData struct {
	metadata map[string]string
}

type parseSwiftAuthMetadataTestData struct {
	authMetadata map[string]string
}

type swiftMetricIdentifier struct {
	resolvedEnv          map[string]string
	metadataTestData     *parseSwiftMetadataTestData
	authMetadataTestData *parseSwiftAuthMetadataTestData
	name                 string
}

var swiftMetadataTestData = []parseSwiftMetadataTestData{
	// Only required parameters
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container"}},
	// Adding objectCount
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5"}},
	// Adding objectPrefix
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "objectPrefix": "my-prefix"}},
	// Adding objectDelimiter
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "objectDelimiter": "/"}},
	// Adding objectLimit
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "objectLimit": "1000"}},
	// Adding timeout
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "objectCount": "5", "timeout": "2"}},
	// Adding onlyFiles
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "containerName": "my-container", "onlyFiles": "true"}},
}

var swiftAuthMetadataTestData = []parseSwiftAuthMetadataTestData{
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "projectID": "my-project-id", "authURL": "http://localhost:5000/v3/"}},
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret", "authURL": "http://localhost:5000/v3/"}},
}

var invalidMetadataTestData = []parseSwiftMetadataTestData{
	// Missing swiftURL
	{metadata: map[string]string{"containerName": "my-container", "objectCount": "5"}},
	// Missing containerName
	{metadata: map[string]string{"swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5"}},
	// objectCount is not an integer value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5.5"}},
	// timeout is not an integer value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5", "timeout": "2.5"}},
	// onlyFiles is not a boolean value
	{metadata: map[string]string{"containerName": "my-container", "swiftURL": "http://localhost:8080/v1/my-account-id", "objectCount": "5", "onlyFiles": "yes"}},
}

var invalidSwiftAuthMetadataTestData = []parseSwiftAuthMetadataTestData{
	// Using Password method:

	// Missing userID
	{authMetadata: map[string]string{"password": "my-password", "projectID": "my-project-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing password
	{authMetadata: map[string]string{"userID": "my-id", "projectID": "my-project-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing projectID
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "authURL": "http://localhost:5000/v3/"}},
	// Missing authURL
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "projectID": "my-project-id"}},

	// Using Application Credentials method:

	// Missing appCredentialID
	{authMetadata: map[string]string{"appCredentialSecret": "my-app-credential-secret", "authURL": "http://localhost:5000/v3/"}},
	// Missing appCredentialSecret
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing authURL
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret"}},
}

func TestSwiftGetMetricSpecForScaling(t *testing.T) {
	testCases := []swiftMetricIdentifier{
		{nil, &swiftMetadataTestData[0], &swiftAuthMetadataTestData[0], "swift-my-container"},
		{nil, &swiftMetadataTestData[1], &swiftAuthMetadataTestData[0], "swift-my-container"},
		{nil, &swiftMetadataTestData[2], &swiftAuthMetadataTestData[0], "swift-my-container"},
		{nil, &swiftMetadataTestData[3], &swiftAuthMetadataTestData[0], "swift-my-container"},
		{nil, &swiftMetadataTestData[4], &swiftAuthMetadataTestData[0], "swift-my-container"},
		{nil, &swiftMetadataTestData[5], &swiftAuthMetadataTestData[0], "swift-my-container"},
		{nil, &swiftMetadataTestData[6], &swiftAuthMetadataTestData[0], "swift-my-container"},

		{nil, &swiftMetadataTestData[0], &swiftAuthMetadataTestData[1], "swift-my-container"},
		{nil, &swiftMetadataTestData[1], &swiftAuthMetadataTestData[1], "swift-my-container"},
		{nil, &swiftMetadataTestData[2], &swiftAuthMetadataTestData[1], "swift-my-container"},
		{nil, &swiftMetadataTestData[3], &swiftAuthMetadataTestData[1], "swift-my-container"},
		{nil, &swiftMetadataTestData[4], &swiftAuthMetadataTestData[1], "swift-my-container"},
		{nil, &swiftMetadataTestData[5], &swiftAuthMetadataTestData[1], "swift-my-container"},
		{nil, &swiftMetadataTestData[6], &swiftAuthMetadataTestData[1], "swift-my-container"},
	}

	for _, testData := range testCases {
		testData := testData
		meta, err := parseSwiftMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		_, err = parseSwiftAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
		if err != nil {
			t.Fatal("Could not parse auth metadata:", err)
		}

		mockSwiftScaler := swiftScaler{meta, nil}

		metricSpec := mockSwiftScaler.GetMetricSpecForScaling()

		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestParseSwiftMetadataForInvalidCases(t *testing.T) {
	testCases := []swiftMetricIdentifier{
		{nil, &invalidMetadataTestData[0], &parseSwiftAuthMetadataTestData{}, "missing swiftURL"},
		{nil, &invalidMetadataTestData[1], &parseSwiftAuthMetadataTestData{}, "missing containerName"},
		{nil, &invalidMetadataTestData[2], &parseSwiftAuthMetadataTestData{}, "objectCount is not an integer value"},
		{nil, &invalidMetadataTestData[3], &parseSwiftAuthMetadataTestData{}, "onlyFiles is not a boolean value"},
		{nil, &invalidMetadataTestData[4], &parseSwiftAuthMetadataTestData{}, "timeout is not an integer value"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseSwiftMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}

func TestParseSwiftAuthenticationMetadataForInvalidCases(t *testing.T) {
	testCases := []swiftMetricIdentifier{
		{nil, &parseSwiftMetadataTestData{}, &invalidSwiftAuthMetadataTestData[0], "missing userID"},
		{nil, &parseSwiftMetadataTestData{}, &invalidSwiftAuthMetadataTestData[1], "missing password"},
		{nil, &parseSwiftMetadataTestData{}, &invalidSwiftAuthMetadataTestData[2], "missing projectID"},
		{nil, &parseSwiftMetadataTestData{}, &invalidSwiftAuthMetadataTestData[3], "missing authURL for password method"},
		{nil, &parseSwiftMetadataTestData{}, &invalidSwiftAuthMetadataTestData[4], "missing appCredentialID"},
		{nil, &parseSwiftMetadataTestData{}, &invalidSwiftAuthMetadataTestData[5], "missing appCredentialSecret"},
		{nil, &parseSwiftMetadataTestData{}, &invalidSwiftAuthMetadataTestData[6], "missing authURL for application credentials method"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseSwiftAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}
