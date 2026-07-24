package auth

import "strings"

func mergeGroups(groupSets ...[]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, groups := range groupSets {
		for _, group := range groups {
			group = strings.TrimSpace(group)
			if group == "" || seen[group] {
				continue
			}
			seen[group] = true
			out = append(out, group)
		}
	}
	return out
}
