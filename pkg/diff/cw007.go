package diff

import (
	"encoding/json"
	"sort"

	"github.com/Controlwatch/control-watch/pkg/model"
)

const tokenRolesSubject = "token.roles"

// RoleMembersChanged detects changed member sets for RWA token roles.
type RoleMembersChanged struct{}

// Detect implements Detector.
func (RoleMembersChanged) Detect(before model.Observation, after model.Observation) Result {
	if reason := comparisonReason(before, after, tokenRolesSubject); reason != "" {
		return Result{NotCompared: []model.NotCompared{{Subject: tokenRolesSubject, Reason: reason}}}
	}
	beforeRoles := tokenRoles(before)
	afterRoles := tokenRoles(after)
	result := Result{}
	for _, role := range roleNames(beforeRoles, afterRoles) {
		beforeMembers := roleMembers(beforeRoles, role)
		afterMembers := roleMembers(afterRoles, role)
		path := tokenRolesSubject + "." + role
		if reason := measuredComparisonReason(before, after, path, beforeMembers, afterMembers); reason != "" {
			result.NotCompared = append(result.NotCompared, model.NotCompared{Subject: path, Reason: reason})
			continue
		}
		beforeSet := canonicalStringSet(beforeMembers.Value)
		afterSet := canonicalStringSet(afterMembers.Value)
		if beforeSet == afterSet {
			continue
		}
		result.Findings = append(result.Findings, model.Finding{
			Code:           "CW-007",
			Name:           "ROLE_MEMBERS_CHANGED",
			Subject:        path,
			Before:         beforeSet,
			After:          afterSet,
			ReviewPriority: model.PriorityMedium,
			Explanation:    "The member set for a token role changed.",
			NextStep:       "Review the role members and confirm the intended assignments.",
			Evidence:       model.Evidence{Items: []model.EvidenceItem{}},
		})
	}
	return result
}

func tokenRoles(observation model.Observation) map[string]model.Measured[[]string] {
	if observation.Token == nil || observation.Token.Roles == nil {
		return map[string]model.Measured[[]string]{}
	}
	return observation.Token.Roles
}

func roleNames(
	before map[string]model.Measured[[]string],
	after map[string]model.Measured[[]string],
) []string {
	names := make(map[string]struct{}, len(before)+len(after))
	for name := range before {
		names[name] = struct{}{}
	}
	for name := range after {
		names[name] = struct{}{}
	}
	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)
	return ordered
}

func roleMembers(roles map[string]model.Measured[[]string], role string) model.Measured[[]string] {
	members, ok := roles[role]
	if !ok {
		return model.Measure([]string{})
	}
	return members
}

func canonicalStringSet(values []string) string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	ordered := make([]string, 0, len(set))
	for value := range set {
		ordered = append(ordered, value)
	}
	sort.Strings(ordered)
	encoded, _ := json.Marshal(ordered)
	return string(encoded)
}
