/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
  "github.com/steadybit/action-kit/go/action_kit_api/v2"
  "github.com/steadybit/action-kit/go/action_kit_sdk"
  "github.com/steadybit/discovery-kit/go/discovery_kit_api"
  "github.com/steadybit/event-kit/go/event_kit_api"
  "github.com/steadybit/extension-jvm/config"
  "github.com/steadybit/extension-jvm/extjvm"
  "github.com/steadybit/extension-jvm/extjvm/common"
  "github.com/steadybit/extension-jvm/extjvm/controller"
  "github.com/steadybit/extension-jvm/extjvm/hotspot"
  "github.com/steadybit/extension-jvm/extjvm/java_process"
  "github.com/steadybit/extension-kit/extbuild"
  "github.com/steadybit/extension-kit/exthealth"
  "github.com/steadybit/extension-kit/exthttp"
  "github.com/steadybit/extension-kit/extlogging"
)

func main() {
	// Most Steadybit extensions leverage zerolog. To encourage persistent logging setups across extensions,
	// you may leverage the extlogging package to initialize zerolog. Among others, this package supports
	// configuration of active log levels and the log format (JSON or plain text).
	//
	// Example
	//  - to activate JSON logging, set the environment variable STEADYBIT_LOG_FORMAT="json"
	//  - to set the log level to debug, set the environment variable STEADYBIT_LOG_LEVEL="debug"
	extlogging.InitZeroLog()

	// Build information is set at compile-time. This line writes the build information to the log.
	// The information is mostly handy for debugging purposes.
	extbuild.PrintBuildInformation()

	//This will start /health/liveness and /health/readiness endpoints on port 8083 for use with kubernetes
	//The port can be configured using the STEADYBIT_EXTENSION_HEALTH_PORT environment variable
	exthealth.SetReady(false)
	exthealth.StartProbes(int(common.GetOwnHealthPort()))

	// Most extensions require some form of configuration. These calls exist to parse and validate the
	// configuration obtained from environment variables.
	config.ParseConfiguration()
	config.ValidateConfiguration()

	// This call registers a handler for the extension's root path. This is the path initially accessed
	// by the Steadybit agent to obtain the extension's capabilities.
	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	// This is a section you will most likely want to change: The registration of HTTP handlers
	// for your extension. You might want to change these because the names do not fit, or because
	// you do not have a need for all of them.
	extjvm.RegisterDiscoveryHandlers()
	action_kit_sdk.RegisterAction(extjvm.NewLogAction())

	//This will install a signal handlder, that will stop active actions when receiving a SIGURS1, SIGTERM or SIGINT
	action_kit_sdk.InstallSignalHandler()


  // Shutdown Discovery on SIGTERM
  extjvm.InstallSignalHandler()
  //Start Java agent controller
  controller.Start(common.GetOwnJVMAttachmentPort())
  // Start JVM Watcher
  java_process.Start()
  // Start Hotspot JVM Watcher
  hotspot.Start()
  // Start listening for JVM events
  extjvm.Activate(0)
  //Init discover Spring Applications
  extjvm.InitSpringDiscovery()
  //Init discover Datasources
  extjvm.InitDataSourceDiscovery()
  //Start attaching to JVMs
  extjvm.StartAttachment()
  // Start Spring Discovery
  extjvm.StartSpringDiscovery()
  // Start Datasource Discovery
  extjvm.StartDataSourceDiscovery()


	//This will switch the readiness state of the application to true.
	exthealth.SetReady(true)

  common.PrintCaps()

	exthttp.Listen(exthttp.ListenOpts{
		// This is the default port under which your extension is accessible.
		// The port can be configured externally through the
		// STEADYBIT_EXTENSION_PORT environment variable.
		// We suggest that you keep port 8080 as the default.
		Port: int(common.GetOwnPort()),
	})
}

// ExtensionListResponse exists to merge the possible root path responses supported by the
// various extension kits. In this case, the response for ActionKit, DiscoveryKit and EventKit.
type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
	event_kit_api.EventListenerList `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		// See this document to learn more about the action list:
		// https://github.com/steadybit/action-kit/blob/main/docs/action-api.md#action-list
		ActionList: action_kit_sdk.GetActionList(),

		// See this document to learn more about the discovery list:
		// https://github.com/steadybit/discovery-kit/blob/main/docs/discovery-api.md#index-response
		DiscoveryList: extjvm.GetDiscoveryList(),
	}
}