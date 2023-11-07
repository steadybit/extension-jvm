// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extjvm

import (
	"embed"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
)

const HttpCallCircuitBreakerID = targetID + ".advice.http-call-circuit-breaker"
const HttpCallTimeoutID = targetID + ".advice.http-call-timeout"

func RegisterAdviceHandlers() {
	exthttp.RegisterHttpHandler("/jvm/advice/http-call-circuit-breaker", exthttp.GetterAsHandler(getAdviceDescriptionHttpCallCircuitBreaker))
	exthttp.RegisterHttpHandler("/jvm/advice/http-call-timeout", exthttp.GetterAsHandler(getAdviceDescriptionHttpCallTimeout))
}

//go:embed advice_templates/http_call_circuit_breaker/*
var httpCallCircuitBreakerContent embed.FS

//go:embed advice_templates/http_call_timeout/*
var httpCallTimeoutContent embed.FS

func getAdviceDescriptionHttpCallCircuitBreaker() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          HttpCallCircuitBreakerID,
		Label:                       "Circuit Breaker",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        targetIcon,
		Tags:                        &[]string{"java", "jvm", "spring", "http", "circuit-breaker"},
		AssessmentQueryApplicable:   "target.type=\"" + targetID + "\" and application.http-outgoing-calls IS PRESENT",
		AssessmentQueryActionNeeded: "application.http-outgoing-calls.missing-circuit-breaker IS PRESENT",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetID + ".advice.http-call-circuit-breaker.experiment-1",
			Name:       "Backend Service Issues",
			Experiment: readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/experiment_backend_service_issues.json"),
		}},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/instructions.md"),
				Motivation:  readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/motivation.md"),
				Summary:     readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/implemented.md"),
			},
		},
	}
}

func getAdviceDescriptionHttpCallTimeout() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                          HttpCallTimeoutID,
		Label:                       "Timeouts",
		Version:                     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                        targetIcon,
		Tags:                         &[]string{"java", "jvm", "spring", "http", "timeout"},
		AssessmentQueryApplicable:   "target.type=\"" + targetID + "\" and application.http-outgoing-calls IS PRESENT",
		AssessmentQueryActionNeeded: "application.http-outgoing-calls.missing-timeout IS PRESENT",
		Experiments: &[]advice_kit_api.ExperimentTemplate{{
			Id:         targetID + ".advice.http-call-timeout.experiment-1",
			Name:       "Response Time Issues",
			Experiment: readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/experiment_response_time_issues.json"),
		}},
		Description: advice_kit_api.AdviceDefinitionDescription{
			ActionNeeded: advice_kit_api.AdviceDefinitionDescriptionActionNeeded{
				Instruction: readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/instructions.md"),
				Motivation:  readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/motivation.md"),
				Summary:     readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/action_needed_summary.md"),
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionDescriptionValidationNeeded{
				Summary: readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/validation_needed.md"),
			},
			Implemented: advice_kit_api.AdviceDefinitionDescriptionImplemented{
				Summary: readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/implemented.md"),
			},
		},
	}
}

func readLocalFile(fs embed.FS, fileName string) string {
	fileContent, err := fs.ReadFile(fileName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read file: %s", fileName)
	}
	return string(fileContent)
}
