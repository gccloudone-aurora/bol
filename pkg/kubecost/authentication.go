package kubecost

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	metadataURL = "http://169.254.169.254/metadata/identity/oauth2/token"
	apiVersion  = "2018-02-01"
	resource    = "https://management.azure.com/"
)

// TokenResponse represents the response from the Managed Identity endpoint.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	ExpiresOn   string `json:"expires_on"`
	NotBefore   string `json:"not_before"`
	Resource    string `json:"resource"`
	TokenType   string `json:"token_type"`
}

// GetAccessToken obtains an access token using aad-pod-identity.
func GetAccessToken() (string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return "", err
	}

	// Set required headers and query parameters
	req.Header.Set("Metadata", "true")
	query := req.URL.Query()
	query.Add("api-version", apiVersion)
	query.Add("resource", resource)
	req.URL.RawQuery = query.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}

	return tokenResponse.AccessToken, nil
}

func main() {
	token, err := GetAccessToken()
	if err != nil {
		fmt.Printf("Error obtaining access token: %v\n", err)
		return
	}
	fmt.Printf("Access Token: %s\n", token)
}
