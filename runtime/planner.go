package runtime

import (
	"sort"

	commonprotocol "github.com/xtls/xray-core/common/protocol"
	xtcore "github.com/xtls/xray-core/core"
	"google.golang.org/protobuf/proto"
)

type OperationType string

const (
	OperationAddInbound    OperationType = "AddInbound"
	OperationRemoveInbound OperationType = "RemoveInbound"
	OperationReplaceInbound OperationType = "ReplaceInbound"
	OperationAddClient     OperationType = "AddClient"
	OperationRemoveClient  OperationType = "RemoveClient"
)

type Operation struct {
	Type    OperationType
	Tag     string
	Inbound *xtcore.InboundHandlerConfig
	User    *commonprotocol.User
	Email   string
}

type Plan struct {
	Operations   []Operation
	NeedFullSync bool
	Reason       string
}

type Planner struct {
}

func (p *Planner) Plan(current, desired *Snapshot) (*Plan, error) {
	plan := &Plan{}
	if current == nil || desired == nil {
		plan.NeedFullSync = true
		plan.Reason = "missing runtime snapshot"
		return plan, nil
	}

	currentTags := sortedKeys(current.Inbounds)
	desiredTags := sortedKeys(desired.Inbounds)

	currentSet := make(map[string]bool, len(currentTags))
	for _, tag := range currentTags {
		currentSet[tag] = true
	}
	desiredSet := make(map[string]bool, len(desiredTags))
	for _, tag := range desiredTags {
		desiredSet[tag] = true
	}

	for _, tag := range currentTags {
		if !desiredSet[tag] {
			plan.Operations = append(plan.Operations, Operation{
				Type:  OperationRemoveInbound,
				Tag:   tag,
				Email: tag,
			})
		}
	}

	for _, tag := range desiredTags {
		if !currentSet[tag] {
			plan.Operations = append(plan.Operations, Operation{
				Type:    OperationAddInbound,
				Tag:     tag,
				Inbound: desired.Inbounds[tag].Core,
			})
			continue
		}

		currentInbound := current.Inbounds[tag]
		desiredInbound := desired.Inbounds[tag]
		if !proto.Equal(currentInbound.Comparable, desiredInbound.Comparable) {
			plan.Operations = append(plan.Operations, Operation{
				Type:    OperationReplaceInbound,
				Tag:     tag,
				Inbound: desiredInbound.Core,
			})
			continue
		}

		appendUserDiffs(plan, tag, currentInbound.Users, desiredInbound.Users)
	}

	return plan, nil
}

func appendUserDiffs(plan *Plan, tag string, current, desired map[string]*commonprotocol.User) {
	currentKeys := sortedUserKeys(current)
	desiredKeys := sortedUserKeys(desired)

	for _, email := range currentKeys {
		desiredUser, ok := desired[email]
		if !ok {
			plan.Operations = append(plan.Operations, Operation{
				Type:  OperationRemoveClient,
				Tag:   tag,
				Email: email,
			})
			continue
		}
		if !proto.Equal(current[email], desiredUser) {
			plan.Operations = append(plan.Operations,
				Operation{
					Type:  OperationRemoveClient,
					Tag:   tag,
					Email: email,
				},
				Operation{
					Type: OperationAddClient,
					Tag:  tag,
					User: desiredUser,
				},
			)
		}
	}

	for _, email := range desiredKeys {
		if _, ok := current[email]; ok {
			continue
		}
		plan.Operations = append(plan.Operations, Operation{
			Type: OperationAddClient,
			Tag:  tag,
			User: desired[email],
		})
	}
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedUserKeys(m map[string]*commonprotocol.User) []string {
	return sortedKeys(m)
}
