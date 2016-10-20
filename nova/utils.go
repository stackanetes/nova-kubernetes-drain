package nova

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/stackanetes/kubernetes-entrypoint/logger"
)

func loadConfigs(confPath string) (config map[string]string, err error) {
	absFP, err := filepath.Abs(confPath)
	if err != nil {
		logger.Error.Println()
		return config, fmt.Errorf("Cannot retrieve absolute path for config file.")
	}

	yamlFile, err := ioutil.ReadFile(absFP)
	if err != nil {
		return config, fmt.Errorf("Cannot load config file.")
	}
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		return config, fmt.Errorf("Invalid format of config file.")
	}
	logger.Info.Printf("Configuration loaded from %s.\n", absFP)

	return
}

func getJson(body io.ReadCloser, target interface{}) error {
	defer body.Close()

	return json.NewDecoder(body).Decode(target)
}

func createOpenstackClient(confPath string) (client *gophercloud.ServiceClient, err error) {
	config, err := loadConfigs(confPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot load variables required to create openstack client.")
	}

	ao := gophercloud.AuthOptions{
		IdentityEndpoint: config["IdentityEndpoint"],
		Username:         config["Username"],
		Password:         config["Password"],
		TenantName:       config["TenantName"],
		DomainID:         config["DomainID"],
	}

	provider, err := openstack.AuthenticatedClient(ao)
	if err != nil {
		return client, fmt.Errorf("Cannot create openstack provider: %v", err)
	}
	for a := 1; a < retryNum + 1; a++ {
		// TODO(DTadrzak): Should break the loop if receive status code == 401
		client, err = openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
			Availability: gophercloud.AvailabilityInternal,
		})
		if err == nil {
			return
		}
		logger.Warning.Printf("Attempt: %d. Cannot initalize new client.\n", a)
		time.Sleep(retryInterval * time.Second)
	}

	return client, fmt.Errorf("Cannot create openstack client: %v", err)
}

// Func based on http://stackoverflow.com/questions/32840687/timeout-for-waitgroup-wait
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false
	case <-time.After(timeout):
		return true
	}
}

// GetMyIPAddress returns ip address
// Based on http://stackoverflow.com/questions/23558425
func GetMyIPAddress() (string, error) {
	conn, err := net.Dial("tcp", "keystone-api:5000")
	if err != nil {
		return "", fmt.Errorf("Cannot get my ip address: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")

	return localAddr[0:idx], nil
}

