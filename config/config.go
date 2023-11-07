/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	ActiveAdviceList               []string `required:"false" split_words:"true" default:"*"`
	DiscoveryCallInterval          string   `json:"discoveryCallInterval" split_words:"true" required:"false" default:"1m"`
	Port                           uint16   `json:"port" split_words:"true" required:"false" default:"8087"`
	HealthPort                     uint16   `json:"healthPort" split_words:"true" required:"false" default:"8083"`
	JavaAgentAttachmentPort        uint16   `json:"javaAgentAttachmentPort" split_words:"true" required:"false" default:"8095"`
	DiscoveryAttributesExcludesJVM []string `json:"discoveryAttributesExcludesJVM" split_words:"true" required:"false"`
}

var (
	Config Specification
)

func ParseConfiguration() {
	err := envconfig.Process("steadybit_extension", &Config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse configuration from environment.")
	}
}

func ValidateConfiguration() {
	// You may optionally validate the configuration here.
}
