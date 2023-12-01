// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extjvm

import (
	"embed"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/advice-kit/go/advice_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
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
		Id:                        HttpCallCircuitBreakerID,
		Label:                     "Request Endpoints via Circuit Breaker",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      targetIcon,
		Tags:                      &[]string{"java", "jvm", "spring", "http", "circuit-breaker"},
		AssessmentQueryApplicable: "target.type=\"" + targetID + "\" and application.http-outgoing-calls IS PRESENT",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "application.http-outgoing-calls.missing-circuit-breaker IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/instructions.md"),
					Motivation:  readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/motivation.md"),
					Summary:     readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:          targetID + ".advice.http-call-circuit-breaker.experiment-1",
						Name:        "Backend Service Issues",
						ShortDescription: "When calling external services, problems with response time, errors, etc. can occur again and again. Your service should be able to handle this well. An experiment can be used to simulate an incorrect response behavior in order to check what effects this has on the affected component. Also, the correct functionality of an implemented circuit breaker should always be validated with an experiment.",
						Type:        "EXPERIMENT",
						Experiment:  extutil.Ptr(advice_kit_api.Experiment(readLocalFile(httpCallCircuitBreakerContent, "advice_templates/http_call_circuit_breaker/experiment_backend_service_issues.json"))),
					},
				}),
			},
		},
	}
}

func getAdviceDescriptionHttpCallTimeout() advice_kit_api.AdviceDefinition {

	return advice_kit_api.AdviceDefinition{
		Id:                        HttpCallTimeoutID,
		Label:                     "Request Endpoints with Timeouts",
		Version:                   extbuild.GetSemverVersionStringOrUnknown(),
		Icon:                      targetIcon,
		Tags:                      &[]string{"java", "jvm", "spring", "http", "timeout"},
		AssessmentQueryApplicable: "target.type=\"" + targetID + "\" and application.http-outgoing-calls IS PRESENT",
		Status: advice_kit_api.AdviceDefinitionStatus{
			ActionNeeded: advice_kit_api.AdviceDefinitionStatusActionNeeded{
				AssessmentQuery: "application.http-outgoing-calls.missing-timeout IS PRESENT",
				Description: advice_kit_api.AdviceDefinitionStatusActionNeededDescription{
					Instruction: readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/instructions.md"),
					Motivation:  readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/motivation.md"),
					Summary:     readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/action_needed_summary.md"),
				},
			},
			Implemented: advice_kit_api.AdviceDefinitionStatusImplemented{
				Description: advice_kit_api.AdviceDefinitionStatusImplementedDescription{
					Summary: readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/implemented.md"),
				},
			},
			ValidationNeeded: advice_kit_api.AdviceDefinitionStatusValidationNeeded{
				Description: advice_kit_api.AdviceDefinitionStatusValidationNeededDescription{
					Summary: readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/validation_needed.md"),
				},
				Validation: extutil.Ptr([]advice_kit_api.Validation{
					{
						Id:          targetID + ".advice.http-call-timeout.experiment-1",
						Name:        "Response Time Issues",
						Type:        "EXPERIMENT",
						ShortDescription: "Validate with an experiment if the service can handle longer response times well and also check if the set timeout has the desired effect.",
						Experiment:  extutil.Ptr(advice_kit_api.Experiment(readLocalFile(httpCallTimeoutContent, "advice_templates/http_call_timeout/experiment_response_time_issues.json"))),
					},
				}),
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
