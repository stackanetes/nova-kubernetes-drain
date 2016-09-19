// Copyright 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package node

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

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
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return config, fmt.Errorf("Invalid format of config file.")
	}
	logger.Info.Printf("Configuration loaded from %s.\n", absFP)
	return
}

func createOpenstackClient(confPath string) (client *gophercloud.ServiceClient, err error) {
	config, err := loadConfigs(confPath)
	if err != nil {
		return client, fmt.Errorf("Cannot load variables required to create openstack client.")
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

	client, err = openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return client, fmt.Errorf("Cannot create openstack client: %v", err)
	}
	return
}

// GetMyIPAddress returns ip address
func GetMyIPAddress() (string, error) {
	iface := os.Getenv("INTERFACE_NAME")
	if iface == "" {
		return "", fmt.Errorf("Environment variable INTERFACE_NAME not set")
	}

	intface, err := net.InterfaceByName(iface)
	if err != nil {
		return "", fmt.Errorf("Cannot get iface: %v", err)
	}

	address, err := intface.Addrs()
	if err != nil || len(address) == 0 {
		return "", fmt.Errorf("Cannot get ip: %v", err)
	}

	// Split in order to remove subnet
	return strings.Split(address[0].String(), "/")[0], nil
}

// GetMyHostname returns hostname
func GetMyHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("Environment variable HOSTNAME not set")
	}
	return hostname, nil
}
