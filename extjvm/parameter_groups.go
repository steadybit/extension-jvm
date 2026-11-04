// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extjvm

import "github.com/steadybit/action-kit/go/action_kit_api/v2"

// groupTrafficFilter holds what decides which calls an attack hits. Unchanged, every matching call is hit: the
// filters default to "any" and the erroneous call rate to 100%.
const groupTrafficFilter = "Traffic Filter"

var (
	outgoingRequestsFilterGroup = action_kit_api.ParameterGroup{
		Name:        groupTrafficFilter,
		Description: "Leave unchanged to affect all outgoing requests.",
	}
	jdbcFilterGroup = action_kit_api.ParameterGroup{
		Name:        groupTrafficFilter,
		Description: "Leave unchanged to affect all operations on all JDBC connections.",
	}
	controllerFilterGroup = action_kit_api.ParameterGroup{
		Name:        groupTrafficFilter,
		Description: "Leave unchanged to affect all requests to the request mapping.",
	}
	methodCallsFilterGroup = action_kit_api.ParameterGroup{
		Name:        groupTrafficFilter,
		Description: "Leave unchanged to affect every call of the method.",
	}
)
