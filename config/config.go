/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Specification is the configuration specification for the extension. Configuration values can be applied
// through environment variables. Learn more through the documentation of the envconfig package.
// https://github.com/kelseyhightower/envconfig
type Specification struct {
	DiscoveryCallInterval          string        `json:"discoveryCallInterval" split_words:"true" required:"false" default:"30s"`
	Port                           uint16        `json:"port" split_words:"true" required:"false" default:"8087"`
	HealthPort                     uint16        `json:"healthPort" split_words:"true" required:"false" default:"8083"`
	DiscoveryAttributesExcludesJVM []string      `json:"discoveryAttributesExcludesJVM" split_words:"true" required:"false"`
	MinProcessAgeBeforeAttach      time.Duration `json:"minProcessAgeBeforeAttach" split_words:"true" required:"false" default:"15s"`
	MinProcessAgeBeforeInspect     time.Duration `json:"MinProcessAgeBeforeInspect" split_words:"true" required:"false" default:"5s"`
	JvmAttachmentEnabled           bool          `json:"jvmAttachmentEnabled" split_words:"true" required:"false" default:"true"`
	JvmAttachmentPort              uint          `json:"jvmAttachmentPort" split_words:"true" required:"false" default:"0"`
	JavaAgentLogLevel              string        `json:"javaAgentLogLevel" split_words:"true" required:"false" default:"INFO"`
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
