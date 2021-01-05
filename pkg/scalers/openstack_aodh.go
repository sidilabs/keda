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
	"time"

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
	defaultValueWhenError      = 0
	aodhInsufficientDataState  = "insufficient data"
	aodhOKState                = "ok"
	aodhAlarmState             = "alarm"
)

/* expected structure declarations */

type aodhMetadata struct {
	metricsURL        string
	metricID          string
	aggregationMethod string
	granularity       int
	threshold         float64
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

type measureResult struct {
	measures [][]interface{}
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

	if val, ok := triggerMetadata["metricsURL"]; ok && val != "" {
		meta.metricsURL = val
	} else {
		return nil, fmt.Errorf("No metricsURL was declared")
	}

	if val, ok := triggerMetadata["metricID"]; ok && val != "" {
		meta.metricID = val
	} else {
		return nil, fmt.Errorf("No metricID was declared")
	}

	if val, ok := triggerMetadata["aggregationMethod"]; ok && val != "" {
		meta.metricID = val
	} else {
		return nil, fmt.Errorf("No aggregationMethod found")
	}

	if val, ok := triggerMetadata["granularity"]; ok && val != "" {
		meta.metricID = val
	} else {
		return nil, fmt.Errorf("No granularity found")
	}

	if val, ok := triggerMetadata["threshold"]; ok && val != "" {
		// converts the string to float64 but its value is convertible to float32 without changing
		_threshold, err := strconv.ParseFloat(val, 32)
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

// Gets measureament from API as float64, converts it to int and return the value.
func (a *aodhScaler) getAlarmsMetric() (float64, error) {

	var token string = ""
	var metricURL string = a.metadata.metricsURL

	isValid, validationError := openstack.IsTokenValid(*a.authMetadata)

	if validationError != nil {
		aodhLog.Error(validationError, "Unable to check token validity.")
		return 0, validationError
	}

	if !isValid {
		var tokenRequestError error
		token, tokenRequestError = a.authMetadata.GetToken()
		a.authMetadata.AuthToken = token
		if tokenRequestError != nil {
			aodhLog.Error(tokenRequestError, "The token being used is invalid")
			return defaultValueWhenError, tokenRequestError
		}
	}

	token = a.authMetadata.AuthToken

	aodhAlarmURL, err := url.Parse(metricURL)

	if err != nil {
		aodhLog.Error(err, "The metrics URL provided is invalid")
		return defaultValueWhenError, fmt.Errorf("The metrics URL is invalid: %s", err.Error())
	}

	aodhAlarmURL.Path = path.Join(aodhAlarmURL.Path, a.metadata.metricID+"/measures")
	queryParameter := aodhAlarmURL.Query()
	granularity := "2"
	if a.metadata.granularity > 1 {
		granularity = strconv.Itoa(a.metadata.granularity)
	}
	queryParameter.Set("granularity", granularity)
	queryParameter.Set("aggregation", a.metadata.aggregationMethod)

	currTimeWithWindow := time.Now().Add(time.Minute + time.Duration(a.metadata.granularity-1)).Format(time.RFC3339)
	queryParameter.Set("start", string(currTimeWithWindow)[:17]+"00")

	aodhAlarmURL.RawQuery = queryParameter.Encode()

	aodhRequest, newReqErr := http.NewRequest("GET", aodhAlarmURL.String(), nil)
	if newReqErr != nil {
		aodhLog.Error(newReqErr, "Could not build metrics request", nil)
	}
	aodhRequest.Header.Set("X-Auth-Token", token)

	resp, requestError := a.authMetadata.HttpClient.Do(aodhRequest)

	if requestError != nil {
		aodhLog.Error(requestError, "Unable to request Metrics from URL: %s.", a.metadata.metricsURL)
		return defaultValueWhenError, requestError
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		bodyError, readError := ioutil.ReadAll(resp.Body)

		if readError != nil {
			aodhLog.Error(readError, "Request failed with code: %s for URL: %s", resp.StatusCode, a.metadata.metricsURL)
			return defaultValueWhenError, readError
		}

		return defaultValueWhenError, fmt.Errorf(string(bodyError))
	}

	m := measureResult{}
	body, errConvertJSON := ioutil.ReadAll(resp.Body)

	if errConvertJSON != nil {
		aodhLog.Error(errConvertJSON, "Failed to convert Body format response to json")
		return defaultValueWhenError, err
	}

	if body == nil {
		return defaultValueWhenError, nil
	}

	errUnMarshall := json.Unmarshal([]byte(body), &m.measures)

	if errUnMarshall != nil {
		aodhLog.Error(errUnMarshall, "Failed converting json format Body structure.")
		return defaultValueWhenError, errUnMarshall
	}

	var targetMeasure []interface{} = nil

	if len(m.measures) > 1 {
		targetMeasure = m.measures[len(m.measures)-1]
	}

	if len(targetMeasure) != 3 {
		aodhLog.Error(fmt.Errorf("Unexpected json response"), "Unexpected json tuple, expected structure is [string, float, float].")
		return defaultValueWhenError, fmt.Errorf("Unexpected json response")
	}

	if val, ok := targetMeasure[2].(float64); ok {
		return val, nil
	}

	aodhLog.Error(fmt.Errorf("Failed to convert interface type to flaot64"), "Unable to convert targetMeasure to expected format float64")
	return defaultValueWhenError, fmt.Errorf("Failed to convert interface type to flaot64")

}
