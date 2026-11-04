// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extjvm

import (
	"testing"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/stretchr/testify/assert"
)

func groupMembers(desc action_kit_api.ActionDescription) map[string][]string {
	members := map[string][]string{}
	for _, p := range desc.Parameters {
		if p.Group != nil {
			members[*p.Group] = append(members[*p.Group], p.Name)
		}
	}
	return members
}

func definedGroups(desc action_kit_api.ActionDescription) []string {
	var names []string
	if desc.ParameterGroups != nil {
		for _, g := range *desc.ParameterGroups {
			names = append(names, g.Name)
		}
	}
	return names
}

func TestParameterGroups(t *testing.T) {
	tests := []struct {
		name    string
		desc    action_kit_api.ActionDescription
		groups  map[string][]string
		defined []string
		visible []string
	}{
		{"controllerDelay", controllerDelayDescribe(),
			map[string][]string{groupTrafficFilter: {"method", "methods"}}, []string{groupTrafficFilter}, []string{"delayJitter"}},
		{"controllerException", controllerExceptionDescribe(),
			map[string][]string{groupTrafficFilter: {"method", "methods", "erroneousCallRate"}}, []string{groupTrafficFilter}, nil},
		{"httpClientDelay", httpClientDelayDescribe(),
			map[string][]string{groupTrafficFilter: {"httpMethods", "hostAddress", "urlPath"}}, []string{groupTrafficFilter}, []string{"delayJitter"}},
		{"httpClientStatus", httpClientStatusDescribe(),
			map[string][]string{groupTrafficFilter: {"httpMethods", "hostAddress", "urlPath", "erroneousCallRate"}}, []string{groupTrafficFilter}, []string{"failureCauses"}},
		{"javaMethodDelay", methodDelayDescribe(), map[string][]string{}, nil, []string{"validate", "delayJitter"}},
		{"javaMethodException", methodExceptionDescribe(),
			map[string][]string{groupTrafficFilter: {"erroneousCallRate"}}, []string{groupTrafficFilter}, []string{"validate"}},
		{"jdbcTemplateDelay", jdbcTemplateDelayDescribe(),
			map[string][]string{groupTrafficFilter: {"operations", "jdbcUrl"}}, []string{groupTrafficFilter}, []string{"delayJitter"}},
		{"jdbcTemplateException", jdbcTemplateExceptionDescribe(),
			map[string][]string{groupTrafficFilter: {"operations", "jdbcUrl", "erroneousCallRate"}}, []string{groupTrafficFilter}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.groups, groupMembers(tt.desc))
			assert.Equal(t, tt.defined, definedGroups(tt.desc))

			// These actions set no order, so the list order decides and a group is shown where its first parameter
			// is: the groups come after the parameters shown directly, and "Additional Settings" comes last.
			seenGroup := false
			for i, p := range tt.desc.Parameters {
				assert.Nil(t, p.Order, "parameter %s: the list order decides", p.Name)
				if p.Group != nil {
					seenGroup = true
					if *p.Group == action_kit_api.ParameterGroupAdditionalSettings {
						assert.Equal(t, len(tt.desc.Parameters)-1, i, "Additional Settings is last")
					}
				} else {
					assert.False(t, seenGroup, "parameter %s is shown directly after a group", p.Name)
				}
				// No advanced parameter is left alone in "Additional Settings" on platforms with groups.
				if p.Advanced != nil && *p.Advanced {
					assert.NotNil(t, p.Group, "parameter %s", p.Name)
				}
			}
			for _, name := range tt.visible {
				for _, p := range tt.desc.Parameters {
					if p.Name == name {
						assert.Nil(t, p.Advanced, "parameter %s is shown on every platform", name)
						assert.Nil(t, p.Group, "parameter %s is shown directly", name)
					}
				}
			}
		})
	}
}
