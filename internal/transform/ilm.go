package transform

import "encoding/json"

type ILMPolicy struct {
	Name         string           `json:"name"`
	Version      any              `json:"version,omitempty"`
	ModifiedDate any              `json:"modified_date,omitempty"`
	Phases       []string         `json:"phases"`
	Policy       json.RawMessage  `json:"policy"`
	InUseBy      ILMPolicyInUseBy `json:"in_use_by"`
}

type ILMPolicyInUseBy struct {
	Indices             []string `json:"indices"`
	DataStreams         []string `json:"data_streams"`
	ComposableTemplates []string `json:"composable_templates"`
}

func ILMPolicies(raw json.RawMessage) []ILMPolicy {
	var root map[string]struct {
		Version      any              `json:"version"`
		ModifiedDate any              `json:"modified_date"`
		Policy       json.RawMessage  `json:"policy"`
		InUseBy      ILMPolicyInUseBy `json:"in_use_by"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return []ILMPolicy{}
	}
	out := make([]ILMPolicy, 0, len(root))
	for name, item := range root {
		out = append(out, ILMPolicy{
			Name:         name,
			Version:      item.Version,
			ModifiedDate: item.ModifiedDate,
			Phases:       ilmPolicyPhases(item.Policy),
			Policy:       item.Policy,
			InUseBy:      normalizeILMPolicyInUseBy(item.InUseBy),
		})
	}
	return out
}

func normalizeILMPolicyInUseBy(value ILMPolicyInUseBy) ILMPolicyInUseBy {
	if value.Indices == nil {
		value.Indices = []string{}
	}
	if value.DataStreams == nil {
		value.DataStreams = []string{}
	}
	if value.ComposableTemplates == nil {
		value.ComposableTemplates = []string{}
	}
	return value
}

func ilmPolicyPhases(raw json.RawMessage) []string {
	var policy struct {
		Phases map[string]any `json:"phases"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return []string{}
	}
	order := []string{"hot", "warm", "cold", "frozen", "delete"}
	out := []string{}
	for _, phase := range order {
		if _, ok := policy.Phases[phase]; ok {
			out = append(out, phase)
		}
	}
	for phase := range policy.Phases {
		if !containsString(out, phase) {
			out = append(out, phase)
		}
	}
	return out
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
