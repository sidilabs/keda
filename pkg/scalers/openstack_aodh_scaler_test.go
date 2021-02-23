package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseOpenstackAodhMetadataTestData struct {
	metadata map[string]string
}

type parseOpenstackAodhAuthMetadatesData struct {
	authMetadata map[string]string
}

type openstackAodhtMetricIdentifier struct {
	resolvedEnv          map[string]string
	metadataTestData     *parseOpenstackAodhMetadataTestData
	authMetadataTestData *parseOpenstackAodhAuthMetadatesData
	name                 string
}

var opentsackAodhMetadataTestData = []parseOpenstackAodhMetadataTestData{
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "sum", "granularity": "300", "threshold": "1250"}},
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "max", "granularity": "300", "threshold": "1250"}},
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250", "timeout": "30"}},
}

var openstackAodhAuthMetadataTestData = []parseOpenstackAodhAuthMetadatesData{
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password", "authURL": "http://localhost:5000/v3/"}},
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret", "authURL": "http://localhost:5000/v3/"}},
}

var invalidOpenstackAodhMetadaTestData = []parseOpenstackAodhMetadataTestData{

	// Missing metrics url
	{metadata: map[string]string{"metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},

	// Empty metrics url
	{metadata: map[string]string{"metricsUrl": "", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},

	// Missing metricID
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250", "timeout": "30"}},

	//Empty metricID
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "", "aggregationMethod": "mean", "granularity": "300", "threshold": "1250"}},

	//Missing aggregation method
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "granularity": "300", "threshold": "1250", "timeout": "30"}},

	//Missing granularity
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "threshold": "1250", "timeout": "30"}},

	//Missing threshold
	{metadata: map[string]string{"metricsUrl": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "timeout": "30"}},

	//granularity 0
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "avc", "threshold": "1250"}},

	//threshold 0
	{metadata: map[string]string{"metricsURL": "http://localhost:8041/v1/metric", "metricID": "003bb589-166d-439d-8c31-cbf098d863de", "aggregationMethod": "mean", "granularity": "300", "threshold": "0z"}},
}

var invalidOpenstackAodhAuthMetadataTestData = []parseOpenstackAodhAuthMetadatesData{
	// Using Password method:

	// Missing userID
	{authMetadata: map[string]string{"password": "my-password", "authURL": "http://localhost:5000/v3/"}},
	// Missing password
	{authMetadata: map[string]string{"userID": "my-id", "authURL": "http://localhost:5000/v3/"}},

	// Missing authURL
	{authMetadata: map[string]string{"userID": "my-id", "password": "my-password"}},

	// Using Application Credentials method:

	// Missing appCredentialID
	{authMetadata: map[string]string{"appCredentialSecret": "my-app-credential-secret", "authURL": "http://localhost:5000/v3/"}},
	// Missing appCredentialSecret
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "authURL": "http://localhost:5000/v3/"}},
	// Missing authURL
	{authMetadata: map[string]string{"appCredentialID": "my-app-credential-id", "appCredentialSecret": "my-app-credential-secret"}},
}

func TestOpenstackMetricsGetMetricsForSpecScaling(t *testing.T) {
	// first, test cases with authentication based on password
	testCases := []openstackAodhtMetricIdentifier{
		{nil, &opentsackAodhMetadataTestData[0], &openstackAodhAuthMetadataTestData[0], "openstack-aodh-mean"},
		{nil, &opentsackAodhMetadataTestData[1], &openstackAodhAuthMetadataTestData[0], "openstack-aodh-sum"},
		{nil, &opentsackAodhMetadataTestData[2], &openstackAodhAuthMetadataTestData[0], "openstack-aodh-max"},
		{nil, &opentsackAodhMetadataTestData[3], &openstackAodhAuthMetadataTestData[0], "openstack-aodh-mean"},

		{nil, &opentsackAodhMetadataTestData[0], &openstackAodhAuthMetadataTestData[1], "openstack-aodh-mean"},
		{nil, &opentsackAodhMetadataTestData[1], &openstackAodhAuthMetadataTestData[1], "openstack-aodh-sum"},
		{nil, &opentsackAodhMetadataTestData[2], &openstackAodhAuthMetadataTestData[1], "openstack-aodh-max"},
		{nil, &opentsackAodhMetadataTestData[3], &openstackAodhAuthMetadataTestData[1], "openstack-aodh-mean"},
	}

	for _, testData := range testCases {
		testData := testData
		meta, err := parseAodhMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})

		if err != nil {
			t.Fatal("Could not parse metadata from openstack metrics scaler")
		}

		_, err = parseAodhAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})

		mockMetricsScaler := aodhScaler{meta, nil}
		metricsSpec := mockMetricsScaler.GetMetricSpecForScaling()
		metricName := metricsSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestOpenstackMetricsGetMetricsForSpecScalingInvalidMetaData(t *testing.T) {
	testCases := []openstackAodhtMetricIdentifier{
		{nil, &invalidOpenstackAodhMetadaTestData[0], &openstackAodhAuthMetadataTestData[0], "Missing metrics url"},
		{nil, &invalidOpenstackAodhMetadaTestData[1], &openstackAodhAuthMetadataTestData[0], "Empty metrics url"},
		{nil, &invalidOpenstackAodhMetadaTestData[2], &openstackAodhAuthMetadataTestData[0], "Missing metricID"},
		{nil, &invalidOpenstackAodhMetadaTestData[3], &openstackAodhAuthMetadataTestData[0], "Empty metricID"},
		{nil, &invalidOpenstackAodhMetadaTestData[4], &openstackAodhAuthMetadataTestData[0], "Missing aggregation method"},
		{nil, &invalidOpenstackAodhMetadaTestData[5], &openstackAodhAuthMetadataTestData[0], "Missing granularity"},
		{nil, &invalidOpenstackAodhMetadaTestData[6], &openstackAodhAuthMetadataTestData[0], "Missing threshold"},
		{nil, &invalidOpenstackAodhMetadaTestData[7], &openstackAodhAuthMetadataTestData[0], "Missing threshold"},
		{nil, &invalidOpenstackAodhMetadaTestData[8], &openstackAodhAuthMetadataTestData[0], "Missing threshold"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(pt *testing.T) {
			_, err := parseAodhMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}

func TestOpenstackAodhAuthenticationInvalidAuthMetadata(t *testing.T) {
	testCases := []openstackAodhtMetricIdentifier{
		{nil, &opentsackAodhMetadataTestData[0], &invalidOpenstackAodhAuthMetadataTestData[0], "Missing userID"},
		{nil, &opentsackAodhMetadataTestData[0], &invalidOpenstackAodhAuthMetadataTestData[1], "Missing password"},
		{nil, &opentsackAodhMetadataTestData[0], &invalidOpenstackAodhAuthMetadataTestData[2], "Missing authURL"},
		{nil, &opentsackAodhMetadataTestData[0], &invalidOpenstackAodhAuthMetadataTestData[3], "Missing appCredentialID"},
		{nil, &opentsackAodhMetadataTestData[0], &invalidOpenstackAodhAuthMetadataTestData[4], "Missing appCredentialSecret"},
		{nil, &opentsackAodhMetadataTestData[0], &invalidOpenstackAodhAuthMetadataTestData[5], "Missing authURL - application credential"},
	}

	for _, testData := range testCases {
		testData := testData
		t.Run(testData.name, func(ptr *testing.T) {
			_, err := parseAodhAuthenticationMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authMetadataTestData.authMetadata})
			assert.NotNil(t, err)
		})
	}
}
