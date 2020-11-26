package scalers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/kedacore/keda/v2/pkg/scalers/openstack"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultObjectCount = 2
)

type swiftMetadata struct {
	swiftURL      string
	containerName string
	objectCount   int
}

type swiftAuthenticationMetadata struct {
	userID                string
	password              string
	projectID             string
	authURL               string
	appCredentialSecret   string
	appCredentialSecretID string
}

type swiftScaler struct {
	metadata     *swiftMetadata
	authMetadata *openstack.OpenStackAuthMetadata
}

var swiftLog = logf.Log.WithName("swift_scaler")

func (s *swiftScaler) getSwiftContainerObjectCount() (int, error) {

	var token string = ""
	var swiftURL string = s.metadata.swiftURL
	var containerName string = s.metadata.containerName

	isValid, validationError := openstack.IsTokenValid(*s.authMetadata)

	if validationError != nil {
		return 0, validationError
	}

	if !isValid {
		var tokenRequestError error
		token, tokenRequestError = s.authMetadata.GetToken()
		s.authMetadata.AuthToken = token
		if tokenRequestError != nil {
			return 0, tokenRequestError
		}
	}

	token = s.authMetadata.AuthToken

	swiftContainerURL, err := url.Parse(swiftURL)

	if err != nil {
		return 0, fmt.Errorf("the swiftURL is invalid: %s", err.Error())
	}

	swiftContainerURL.Path = path.Join(swiftContainerURL.Path, containerName)

	swiftRequest, _ := http.NewRequest("GET", swiftContainerURL.String(), nil)
	swiftRequest.Header.Set("X-Auth-Token", token)

	resp, requestError := s.authMetadata.HttpClient.Do(swiftRequest)

	if requestError != nil {
		return 0, requestError
	} else {
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			objectCount, conversionError := strconv.Atoi(resp.Header["X-Container-Object-Count"][0])
			return objectCount, conversionError
		}

		bodyError, readError := ioutil.ReadAll(resp.Body)

		if readError != nil {
			return 0, readError
		}

		return 0, fmt.Errorf(string(bodyError))
	}

}

func NewSwiftScaler(config *ScalerConfig) (Scaler, error) {
	openstackAuth := new(openstack.OpenStackAuthMetadata)

	swiftMetadata, err := parseSwiftMetadata(config.TriggerMetadata)

	if err != nil {
		return nil, fmt.Errorf("error parsing swift metadata: %s", err)
	}

	authMetadata, err := parseSwiftAuthenticationMetadata(config.AuthParams)

	if err != nil {
		return nil, fmt.Errorf("error parsing swift authentication metadata: %s", err)
	}

	// User choose the "application_credentials" authentication method
	if authMetadata.appCredentialSecretID != "" {
		openstackAuth, err = openstack.NewAppCredentialsAuth(authMetadata.authURL, authMetadata.appCredentialSecretID, authMetadata.appCredentialSecret)

		if err != nil {
			return nil, fmt.Errorf("error getting openstack credentials for application credentials method: %s", err)
		}

	} else {
		// User choose the "password" authentication method
		if authMetadata.userID != "" {
			openstackAuth, err = openstack.NewPasswordAuth(authMetadata.authURL, authMetadata.userID, authMetadata.password, authMetadata.projectID)

			if err != nil {
				return nil, fmt.Errorf("error getting openstack credentials for password method: %s", err)
			}
		} else {
			return nil, fmt.Errorf("no authentication method was provided for OpenStack")
		}
	}

	return &swiftScaler{
		metadata:     swiftMetadata,
		authMetadata: openstackAuth,
	}, nil
}

func parseSwiftMetadata(triggerMetadata map[string]string) (*swiftMetadata, error) {
	meta := swiftMetadata{}

	if val, ok := triggerMetadata["swiftURL"]; ok {
		meta.swiftURL = val
	} else {
		return nil, fmt.Errorf("no swiftURL given")
	}

	if val, ok := triggerMetadata["containerName"]; ok {
		meta.containerName = val
	} else {
		return nil, fmt.Errorf("no containerName given")
	}

	if val, ok := triggerMetadata["objectCount"]; ok {
		targetObjectCount, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("objectCount parsing error: %s", err.Error())
		}
		meta.objectCount = targetObjectCount
	} else {
		meta.objectCount = defaultObjectCount
	}

	return &meta, nil

}

func parseSwiftAuthenticationMetadata(authParams map[string]string) (swiftAuthenticationMetadata, error) {
	authMeta := swiftAuthenticationMetadata{}

	if authParams["authURL"] != "" {
		authMeta.authURL = authParams["authURL"]
	} else {
		return authMeta, fmt.Errorf("authURL doesn't exist in the authParams")
	}

	if authParams["userID"] != "" {
		authMeta.userID = authParams["userID"]

		if authParams["password"] != "" {
			authMeta.password = authParams["password"]
		} else {
			return authMeta, fmt.Errorf("password doesn't exist in the authParams")
		}

		if authParams["projectID"] != "" {
			authMeta.projectID = authParams["projectID"]
		} else {
			return authMeta, fmt.Errorf("projectID doesn't exist in the authParams")
		}

	} else {
		if authParams["appCredentialSecretID"] != "" {
			authMeta.appCredentialSecretID = authParams["appCredentialSecretID"]

			if authParams["appCredentialSecret"] != "" {
				authMeta.appCredentialSecret = authParams["appCredentialSecret"]
			} else {
				return authMeta, fmt.Errorf("appCredentialSecret doesn't exist in the authParams")
			}

		} else {
			return authMeta, fmt.Errorf("neither userID or appCredentialSecretID exist in the authParams")
		}
	}

	return authMeta, nil
}

func (s *swiftScaler) IsActive(ctx context.Context) (bool, error) {

	objectCount, err := s.getSwiftContainerObjectCount()

	if err != nil {
		return false, err
	}

	return objectCount > 0, nil
}

func (s *swiftScaler) Close() error {

	return nil
}

func (s *swiftScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {

	objectCount, err := s.getSwiftContainerObjectCount()

	if err != nil {
		swiftLog.Error(err, "error getting object count")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(objectCount), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *swiftScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetObjectCount := resource.NewQuantity(int64(s.metadata.objectCount), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "swift-container", s.metadata.containerName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetObjectCount,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}
