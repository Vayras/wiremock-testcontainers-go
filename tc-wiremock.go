package testcontainers_wiremock

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const defaultWireMockImage = "docker.io/wiremock/wiremock"
const defaultWireMockVersion = "2.35.0-1"
const defaultPort = "8080"

type WireMockContainer struct {
	testcontainers.Container
	version string
}

type WireMockExtension struct {
	testcontainers.Container
	id        string
	classname string
	jarPath   string
}

// RunContainer creates an instance of the WireMockContainer type
func RunContainer(ctx context.Context, opts ...testcontainers.ContainerCustomizer) (*WireMockContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        defaultWireMockImage + ":" + defaultWireMockVersion,
		ExposedPorts: []string{defaultPort + "/tcp"},
		Cmd:          []string{""},
		WaitingFor:   wait.ForHTTP("/__admin").WithPort(nat.Port(defaultPort)),
	}

	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	for _, opt := range opts {
		opt.Customize(&genericContainerReq)
	}

	req.Cmd = append(req.Cmd, "--disable-banner")

	container, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	if err != nil {
		return nil, err
	}

	return &WireMockContainer{Container: container}, nil
}

func WithMappingFile(id string, filePath string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) {
		cfgFile := testcontainers.ContainerFile{
			HostFilePath:      filePath,
			ContainerFilePath: filepath.Join("/home/wiremock/mappings", id+".json"),
			FileMode:          0755,
		}

		req.Files = append(req.Files, cfgFile)
	}

}

func WithFile(name string, filePath string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) {
		cfgFile := testcontainers.ContainerFile{
			HostFilePath:      filePath,
			ContainerFilePath: "/home/wiremock/__files/" + name,
			FileMode:          0755,
		}

		req.Files = append(req.Files, cfgFile)
	}
}

func WithImage(image string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) {
		req.Image = image
	}
}

func GetURI(ctx context.Context, container testcontainers.Container) (string, error) {
	hostIP, err := container.Host(ctx)
	if err != nil {
		return "", err
	}

	mappedPort, err := container.MappedPort(ctx, nat.Port(defaultPort))
	if err != nil {
		return "", err
	}

	return "http://" + hostIP + ":" + mappedPort.Port(), nil
}

// SendHttpGet sends Http GET request to the container passed as an argument.
// 'queryParams' parameter is optional and can be passed as a nil. Query parameters also work when hardcoded in the endpoint argument.
func SendHttpGet(container testcontainers.Container, endpoint string, queryParams map[string]string) (int, string, error) {
	if queryParams != nil {
		var err error
		endpoint, err = addQueryParamsToURL(endpoint, queryParams)
		if err != nil {
			return -1, "", err
		}
	}

	return sendHttpRequest(http.MethodGet, container, endpoint, nil)
}

// SendHttpDelete sends Http DELETE request to the container passed as an argument.
func SendHttpDelete(container testcontainers.Container, endpoint string) (int, string, error) {
	return sendHttpRequest(http.MethodDelete, container, endpoint, nil)
}

// SendHttpPost sends Http POST request to the container passed as an argument.
func SendHttpPost(container testcontainers.Container, endpoint string, body io.Reader) (int, string, error) {
	return sendHttpRequest(http.MethodPost, container, endpoint, body)
}

// SendHttpPatch sends Http PATCH request to the container passed as an argument.
func SendHttpPatch(container testcontainers.Container, endpoint string, body io.Reader) (int, string, error) {
	return sendHttpRequest(http.MethodPatch, container, endpoint, body)
}

// SendHttpPut sends Http PUT request to the container passed as an argument.
func SendHttpPut(container testcontainers.Container, endpoint string, body io.Reader) (int, string, error) {
	return sendHttpRequest(http.MethodPut, container, endpoint, body)
}

func sendHttpRequest(httpMethod string, container testcontainers.Container, endpoint string, body io.Reader) (int, string, error) {
	ctx := context.Background()

	uri, err := GetURI(ctx, container)
	if err != nil {
		return -1, "", err
	}

	req, err := http.NewRequest(httpMethod, uri+endpoint, body)
	if err != nil {
		return -1, "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, "", err
	}

	out, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, "", err
	}

	return res.StatusCode, string(out), nil
}

func addQueryParamsToURL(endpoint string, queryParams map[string]string) (string, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	existingQueryParams, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return "", err
	}

	for key, value := range queryParams {
		existingQueryParams.Set(key, value)
	}

	parsedURL.RawQuery = existingQueryParams.Encode()

	return parsedURL.String(), nil
}
