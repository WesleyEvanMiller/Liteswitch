package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"liteswitch/handler"
)

type nothing struct{}

type NodePoolLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type rawConfig struct {
	NsToOmit                    map[string]string        `json:"nsToOmit"`
	NodePoolLabel               NodePoolLabel            `json:"nodePoolLabel"`
	NodePoolStateHandlerEnabled bool                     `json:"nodePoolStateHandlerEnabled"`
	CloudType                   string                   `json:"cloudType"`
	CloudGroupID                string                   `json:"cloudGroupID"`
	CloudCredentials            handler.CloudCredentials `json:"cloudCredentials"`
	NodePool                    handler.NodePool         `json:"nodePool"`
}

type config struct {
	nsToOmit      map[string]string
	nodePoolLabel NodePoolLabel
	handlerConfig handler.Config
}

// initDefaults reads from a config file/env and inits our default vars
// such as nsToOmit, resourceQuotaValues
func initDefaults(handle io.Reader) (*config, error) {
	bytes, _ := ioutil.ReadAll(handle)

	var rc rawConfig

	err := json.Unmarshal(bytes, &rc)
	if err != nil {
		return &config{}, err
	}

	c := &config{
		nsToOmit:      rc.NsToOmit,
		nodePoolLabel: rc.NodePoolLabel,
		handlerConfig: handler.Config{
			NodePoolStateHandlerEnabled: rc.NodePoolStateHandlerEnabled,
			CloudType:                   rc.CloudType,
			CloudGroupID:                rc.CloudGroupID,
			CloudCredentials:            rc.CloudCredentials,
			NodePool:                    rc.NodePool,
		},
	}

	return c, err
}
