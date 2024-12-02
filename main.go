/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
	_ "github.com/KimMachineGun/automemlimit" // By default, it sets `GOMEMLIMIT` to 90% of cgroup's memory limit.
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-jvm/config"
	"github.com/steadybit/extension-jvm/extjvm"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/steadybit/extension-kit/extsignals"
	_ "go.uber.org/automaxprocs" // Importing automaxprocs automatically adjusts GOMAXPROCS.
	_ "net/http/pprof"           //allow pprof
	"os"
)

func main() {
	//  - to activate JSON logging, set the environment variable STEADYBIT_LOG_FORMAT="json"
	//  - to set the log level to debug, set the environment variable STEADYBIT_LOG_LEVEL="debug"
	extlogging.InitZeroLog()

	// Build information is set at compile-time. This line writes the build information to the log.
	// The information is mostly handy for debugging purposes.
	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.InfoLevel)

	// Most extensions require some form of configuration. These calls exist to parse and validate the
	// configuration obtained from environment variables.
	config.ParseConfiguration()
	config.ValidateConfiguration()

	//This will start /health/liveness and /health/readiness endpoints on port 8083 for use with kubernetes
	//The port can be configured using the STEADYBIT_EXTENSION_HEALTH_PORT environment variable
	exthealth.SetReady(false)
	exthealth.StartProbes(int(config.Config.HealthPort))

	action_kit_sdk.RegisterCoverageEndpoints()

	// This call registers a handler for the extension's root path. This is the path initially accessed
	// by the Steadybit agent to obtain the extension's capabilities.
	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	// The registration of HTTP handlers for the extension.
	stop, facade, datasource, spring := extjvm.StartJvmInfrastructure()

	//This will install a signal handler, that will stop active actions when receiving a SIGURS1, SIGTERM or SIGINT
	extsignals.AddSignalHandler(extsignals.SignalHandler{
		Handler: func(_ os.Signal) {
			stop()
		},
		Order: extsignals.OrderStopCustom,
		Name:  "extjvm.SignalHandler",
	})
	extsignals.ActivateSignalHandlers()

	discovery_kit_sdk.Register(extjvm.NewJvmDiscovery(facade, datasource, spring))
	action_kit_sdk.RegisterAction(extjvm.NewControllerDelay(facade, spring))
	action_kit_sdk.RegisterAction(extjvm.NewControllerException(facade, spring))
	action_kit_sdk.RegisterAction(extjvm.NewJdbcTemplateException(facade))
	action_kit_sdk.RegisterAction(extjvm.NewJdbcTemplateDelay(facade))
	action_kit_sdk.RegisterAction(extjvm.NewHttpClientStatus(facade))
	action_kit_sdk.RegisterAction(extjvm.NewHttpClientDelay(facade))
	action_kit_sdk.RegisterAction(extjvm.NewJavaMethodDelay(facade))
	action_kit_sdk.RegisterAction(extjvm.NewJavaMethodException(facade))

	//This will switch the readiness state of the application to true.
	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		// This is the default port under which your extension is accessible.
		// The port can be configured externally through the
		// STEADYBIT_EXTENSION_PORT environment variable.
		Port: int(config.Config.Port),
	})
}

// ExtensionListResponse exists to merge the possible root path responses supported by the
// various extension kits. In this case, the response for ActionKit, DiscoveryKit and EventKit.
type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		// See this document to learn more about the action list:
		// https://github.com/steadybit/action-kit/blob/main/docs/action-api.md#action-list
		ActionList: action_kit_sdk.GetActionList(),

		// See this document to learn more about the discovery list:
		// https://github.com/steadybit/discovery-kit/blob/main/docs/discovery-api.md#index-response
		DiscoveryList: discovery_kit_sdk.GetDiscoveryList(),
	}
}
