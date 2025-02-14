package kubecost

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Client represents a connection to the Kubecost API.
type Client struct {
	baseURL     *url.URL
	Client      *http.Client
	bearerToken string
}

// Decorator is a function which modifies a request to the Kubecost API.
type Decorator func(request *http.Request)

// AllocationResponse is the response to an allocation request.
type AllocationResponse struct {
	Code int                     `json:"code"`
	Data []map[string]Allocation `json:"data"`
}

// AllocationProperties are the properties of an allocation.
type AllocationProperties struct {
	Annotations    map[string]string `json:"annotation,omitempty"`
	Cluster        string            `json:"cluster,omitempty"`
	Container      string            `json:"container,omitempty"`
	Controller     string            `json:"controller,omitempty"`
	ControllerKind string            `json:"controllerKind,omitempty"`
	Labels         map[string]string `json:"label,omitempty"`
	Namespace      string            `json:"namespace,omitempty"`
	Node           string            `json:"node,omitempty"`
	Pod            string            `json:"pod,omitempty"`
	Services       []string          `json:"service,omitempty"`
}

// AllocationWindow is the window of time the allocation is effective for.
type AllocationWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Allocation is an allocation of costs.
type Allocation struct {
	Name       string               `json:"name"`
	Properties AllocationProperties `json:"properties"`
	Window     AllocationWindow     `json:"window"`
	Start      time.Time            `json:"start"`
	End        time.Time            `json:"end"`

	Minutes float32 `json:"minutes"`

	CPUCores              float32 `json:"cpuCores"`
	CPUCoreRequestAverage float32 `json:"cpuCoreRequestAverage"`
	CPUCoreUsageAverage   float32 `json:"cpuCoreUsageAverage"`
	CPUCoreHours          float32 `json:"cpuCoreHours"`
	CPUCost               float32 `json:"cpuCost"`
	CPUEfficiency         float32 `json:"cpuEfficiency"`

	GPUHours float32 `json:"gpuHours"`
	GPUCost  float32 `json:"gpuCost"`

	NetworkCost float32 `json:"networkCost"`

	PVBytes     float32 `json:"pvBytes"`
	PVByteHours float32 `json:"pvByteHours"`
	PVCost      float32 `json:"pvCost"`

	RAMBytes              float32 `json:"ramBytes"`
	RAMByteRequestAverage float32 `json:"ramByteRequestAveroage"`
	RAMByteUsageAverage   float32 `json:"ramByteUsageAverage"`
	RAMByteHours          float32 `json:"ramByteHours"`
	RAMCost               float32 `json:"ramCost"`
	RAMEfficiency         float32 `json:"ramEfficiency"`

	SharedCost float32 `json:"sharedCost"`

	ExternalCost float32 `json:"externalCost"`

	TotalCost       float32 `json:"totalCost"`
	TotalEfficiency float32 `json:"totalEfficiency"`
}

// NewClient returns a new Kubecost API client.
func NewClient(baseURL *url.URL, attachBearerToken bool) (*Client, error) {
	token := ""
	if attachBearerToken {
		var err error
		token, err = GetAccessToken()
		if err != nil {
			return nil, fmt.Errorf("Error creating kubecost client: %w", err)
		}
	}

	return &Client{
		baseURL:     baseURL,
		Client:      http.DefaultClient,
		bearerToken: token,
	}, nil
}

// Allocation fetches allocation information from the Kubecost API.
//
// **NOTE**: At least on decorator which provides the `window` argument must be provided,
// otherwise the API will return a 400 Invalid Request response.
func (c *Client) Allocation(ctx context.Context, decorators ...Decorator) (*AllocationResponse, error) {
	url, err := c.baseURL.Parse("model/allocation")
	if err != nil {
		return nil, fmt.Errorf("Allocation: failed to parse url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Allocation:: failed to create request: %w", err)
	}

	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	for _, decorator := range decorators {
		decorator(req)
	}

	log.Printf("requesting %s", req.URL.String())
	res, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Allocation: error sending request: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Allocation: unexpected status code %d (%s)", res.StatusCode, res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Allocation: failed to read body: %w", err)

	}
	allocation := AllocationResponse{}
	err = json.Unmarshal(body, &allocation)
	if err != nil {
		return nil, fmt.Errorf("Allocation: failed to unmarshal response: %w", err)
	}

	if allocation.Code != http.StatusOK {
		return nil, fmt.Errorf("Allocation: unexpected code %d received", allocation.Code)
	}

	return &allocation, nil
}
