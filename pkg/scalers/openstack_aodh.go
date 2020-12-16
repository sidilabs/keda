package scalers

import (
	"context"
	"encoding/json"
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
	defautAodhThreholdAlarmQty = 1

	aodhInsufficientDataState = "insufficient data"
	aodhOKState               = "ok"
	aodhAlarmState            = "alarm"
)

/* expected structure declarations */

type aodhMetadata struct {
	retrieveType  string
	retrieveValue string
	threshold     int
}

type aodhAuthenticationMetadata struct {
	userID                string
	password              string
	authURL               string
	appCredentialSecret   string
	appCredentialSecretID string
}

type aodhScaler struct {
	metadata     *aodhMetadata
	authMetadata *openstack.OpenStackAuthMetadata
}

/*  end of declarations */

var aodhLog = logf.Log.WithName("aodh_scaler")

// NewOpenstackAodhScaler creates new AODH openstack scaler instance
func NewOpenstackAodhScaler(config *ScalerConfig) (Scaler, error) {
	openstackAuth := new(openstack.OpenStackAuthMetadata)

	aodhMetadata, err := parseAodhMetadata(config.TriggerMetadata)

	if err != nil {
		return nil, fmt.Errorf("error parsing AODH metadata: %s", err)
	}

	authMetadata, err := parseAodhAuthenticationMetadata(config.AuthParams)

	if err != nil {
		return nil, fmt.Errorf("error parsing AODH authentication metadata: %s", err)
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
			openstackAuth, err = openstack.NewPasswordAuth(authMetadata.authURL, authMetadata.userID, authMetadata.password, "")

			if err != nil {
				return nil, fmt.Errorf("error getting openstack credentials for password method: %s", err)
			}
		} else {
			return nil, fmt.Errorf("no authentication method was provided for OpenStack")
		}
	}

	return &aodhScaler{
		metadata:     aodhMetadata,
		authMetadata: openstackAuth,
	}, nil
}

func parseAodhMetadata(triggerMetadata map[string]string) (*aodhMetadata, error) {
	meta := aodhMetadata{}

	if val, ok := triggerMetadata["retrieveType"]; ok && val != "" {
		meta.retrieveType = val
	} else {
		return nil, fmt.Errorf("RetrieveType must have one value (id, name or severity)")
	}

	if val, ok := triggerMetadata["retrieveValue"]; ok && val != "" {
		meta.retrieveValue = val
	} else {
		return nil, fmt.Errorf("RetrieveValue must have an integer value assigned to it")
	}

	if val, ok := triggerMetadata["threshold"]; ok && val != "" {
		_threshold, err := strconv.Atoi(val)
		if err != nil {
			aodhLog.Error(err, "Error parsing AODH metadata", "threshold", "threshold")
			return nil, fmt.Errorf("Error parsing AODH metadata : %s", err.Error())
		}
		meta.threshold = _threshold
	}

	return &meta, nil
}

func parseAodhAuthenticationMetadata(authParams map[string]string) (aodhAuthenticationMetadata, error) {
	authMeta := aodhAuthenticationMetadata{}

	if val, ok := authParams["authURL"]; ok && val != "" {
		authMeta.authURL = authParams["authURL"]
	} else {
		return authMeta, fmt.Errorf("authURL doesn't exist in the authParams")
	}

	if val, ok := authParams["userID"]; ok && val != "" {
		authMeta.userID = val

		if val, ok := authParams["password"]; ok && val != "" {
			authMeta.password = val
		} else {
			return authMeta, fmt.Errorf("password doesn't exist in the authParams")
		}

	} else if val, ok := authParams["appCredentialSecretId"]; ok && val != "" {
		authMeta.appCredentialSecretID = val

		if val, ok := authParams["appCredentialSecret"]; ok && val != "" {
			authMeta.appCredentialSecret = val
		}

	} else {
		return authMeta, fmt.Errorf("neither userID or appCredentialSecretID exist in the authParams")
	}

	return authMeta, nil
}

// TODO: improve Normalize string arguments (line 151)
func (a *aodhScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricVal := resource.NewQuantity(int64(a.metadata.threshold), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "openstack-AODH", a.authMetadata.AuthURL)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricVal,
		},
	}

	metricSpec := v2beta2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

func (a *aodhScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := a.getAlarmsMetric()

	if err != nil {
		aodhLog.Error(err, "Error collecting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (a *aodhScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := a.getAlarmsMetric()

	if err != nil {
		return false, err
	}

	return val > 0, nil
}

func (a *aodhScaler) Close() error {
	return nil
}

func (a *aodhScaler) getAlarmsMetric() (int, error) {

	var token string = ""
	var aodhURL string = a.authMetadata.AuthURL

	isValid, validationError := openstack.IsTokenValid(*a.authMetadata)

	if validationError != nil {
		return -1, validationError
	}

	if !isValid {
		var tokenRequestError error
		token, tokenRequestError = a.authMetadata.GetToken()
		a.authMetadata.AuthToken = token
		if tokenRequestError != nil {
			return -1, tokenRequestError
		}
	}

	token = a.authMetadata.AuthToken

	aodhAlarmURL, err := url.Parse(aodhURL)

	if err != nil {
		return -1, fmt.Errorf("the aodhURL is invalid: %s", err.Error())
	}

	aodhAlarmURL.Path = path.Join(aodhAlarmURL.Path, a.metadata.retrieveValue)

	aodhRequest, _ := http.NewRequest("GET", aodhAlarmURL.String(), nil)
	aodhRequest.Header.Set("X-Auth-Token", token)

	resp, requestError := a.authMetadata.HttpClient.Do(aodhRequest)

	if requestError != nil {
		aodhLog.Error(requestError, "Unable to request alarms from URL: %s.", aodhURL)
		return -1, requestError
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		bodyError, readError := ioutil.ReadAll(resp.Body)

		if readError != nil {
			aodhLog.Error(readError, "Request failed with code: %s for URL: %s", resp.StatusCode, a.authMetadata.AuthURL)
			return -1, readError
		}

		return -1, fmt.Errorf(string(bodyError))
	}

	m := map[string]string{}
	body, errConvertJSON := ioutil.ReadAll(resp.Body)

	if body == nil {
		return -1, nil
	}

	if errConvertJSON != nil {
		aodhLog.Error(errConvertJSON, "Failed to convert Body format response to json")
		return -1, err
	}

	errUnMarshall := json.Unmarshal([]byte(body), &m)

	if errUnMarshall != nil {
		aodhLog.Error(errUnMarshall, "Body response failed on parse")
		return -1, errUnMarshall
	}

	if val, ok := m["enabled"]; ok && val != "" {
		isEnabled, errEnabled := strconv.ParseBool(val)

		if errEnabled != nil {
			aodhLog.Error(errEnabled, "Error parsing 'enabled' field from response for alarm with id: %s", a.metadata.retrieveValue)
		}

		if isEnabled {
			aodhLog.Error(nil, "Current Alarm with Id %s is disabled", a.metadata.retrieveValue)
			return -1, fmt.Errorf("Alarm with Id %s is disabled", a.metadata.retrieveValue)
		}
	} else {
		aodhLog.Error(nil, "Couldn't check if alarm with Id %s is enabled", a.metadata.retrieveValue)
		return -1, fmt.Errorf("Couldn't check if alarm with Id %s is enabled", a.metadata.retrieveValue)
	}

	if val, ok := m["state"]; ok && val != "" {
		if val == aodhInsufficientDataState || val == aodhOKState {
			return 0, nil
		} else if val == aodhAlarmState {
			return 1, nil
		} else {
			return -1, fmt.Errorf("Unknown Alarm State detected")
		}
	}

	return -1, fmt.Errorf("Couldn't read state for alarm with id: %s", a.metadata.retrieveValue)

}
