package openstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

const tokensEndpoint = "/auth/tokens"

var httpClient = &http.Client{}

type OpenStackAuthMetadata struct {
	AuthURL    string      `json:"-"`
	AuthToken  string      `json:"-"`
	HttpClient http.Client `json:"-"`
	Properties *AuthProps  `json:"auth"`
}

type AuthProps struct {
	Identity *IdentityProps `json:"identity"`
	Scope    *ScopeProps    `json:"scope,omitempty"`
}

type IdentityProps struct {
	Methods       []string            `json:"methods"`
	Password      *PasswordProps      `json:"password,omitempty"`
	AppCredential *AppCredentialProps `json:"application_credential,omitempty"`
}

type PasswordProps struct {
	User *UserProps `json:"user"`
}

type AppCredentialProps struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

type ScopeProps struct {
	Project *ProjectProps `json:"project"`
}

type UserProps struct {
	ID       string `json:"id"`
	Password string `json:"password"`
}

type ProjectProps struct {
	ID string `json:"id"`
}

func (authProps *OpenStackAuthMetadata) GetToken() (string, error) {

	jsonBody, jsonError := json.Marshal(authProps)

	if jsonError != nil {
		return "", jsonError
	}

	body := bytes.NewReader(jsonBody)

	tokenURL, err := url.Parse(authProps.AuthURL)

	if err != nil {
		return "", fmt.Errorf("the authURL is invalid: %s", err.Error())
	}

	tokenURL.Path = path.Join(tokenURL.Path, tokensEndpoint)

	resp, requestError := http.Post(tokenURL.String(), "application/json", body)

	if requestError != nil {
		return "", requestError
	} else {
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			authProps.AuthToken = resp.Header["X-Subject-Token"][0]
			return resp.Header["X-Subject-Token"][0], nil
		}

		errBody, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf(string(errBody))
	}
}

func IsTokenValid(authProps OpenStackAuthMetadata) (bool, error) {
	token := authProps.AuthToken

	tokenURL, err := url.Parse(authProps.AuthURL)

	if err != nil {
		return false, fmt.Errorf("the authURL is invalid: %s", err.Error())
	}

	tokenURL.Path = path.Join(tokenURL.Path, tokensEndpoint)

	checkTokenRequest, checkRequestError := http.NewRequest("HEAD", tokenURL.String(), nil)
	checkTokenRequest.Header.Set("X-Subject-Token", token)
	checkTokenRequest.Header.Set("X-Auth-Token", token)

	if checkRequestError != nil {
		return false, checkRequestError
	}

	checkResp, requestError := authProps.HttpClient.Do(checkTokenRequest)

	if requestError != nil {
		return false, requestError
	}

	if checkResp.StatusCode >= 400 {
		return false, nil
	}

	return true, nil
}

func NewPasswordAuth(authURL string, userID string, userPassword string, projectID string) (*OpenStackAuthMetadata, error) {
	var tokenError error

	passAuth := new(OpenStackAuthMetadata)

	passAuth.Properties = new(AuthProps)

	passAuth.Properties.Scope = new(ScopeProps)
	passAuth.Properties.Scope.Project = new(ProjectProps)

	passAuth.Properties.Identity = new(IdentityProps)
	passAuth.Properties.Identity.Password = new(PasswordProps)
	passAuth.Properties.Identity.Password.User = new(UserProps)

	url, err := url.Parse(authURL)

	if err != nil {
		return nil, fmt.Errorf("authURL is invalid: %s", err.Error())
	}

	url.Path = path.Join(url.Path, "")

	passAuth.AuthURL = url.String()

	passAuth.HttpClient = *httpClient

	passAuth.Properties.Identity.Methods = []string{"password"}
	passAuth.Properties.Identity.Password.User.ID = userID
	passAuth.Properties.Identity.Password.User.Password = userPassword

	passAuth.Properties.Scope.Project.ID = projectID

	passAuth.AuthToken, tokenError = passAuth.GetToken()

	return passAuth, tokenError
}

func NewAppCredentialsAuth(authURL string, id string, secret string) (*OpenStackAuthMetadata, error) {
	var tokenError error

	appAuth := new(OpenStackAuthMetadata)

	appAuth.Properties = new(AuthProps)

	appAuth.Properties.Identity = new(IdentityProps)

	url, err := url.Parse(authURL)

	if err != nil {
		return nil, fmt.Errorf("authURL is invalid: %s", err.Error())
	}

	url.Path = path.Join(url.Path, "")

	appAuth.AuthURL = url.String()

	appAuth.HttpClient = *httpClient

	appAuth.Properties.Identity.AppCredential = new(AppCredentialProps)
	appAuth.Properties.Identity.Methods = []string{"application_credential"}
	appAuth.Properties.Identity.AppCredential.ID = id
	appAuth.Properties.Identity.AppCredential.Secret = secret

	appAuth.AuthToken, tokenError = appAuth.GetToken()

	return appAuth, tokenError
}
