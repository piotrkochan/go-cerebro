package rbac

import (
	"net/http"
	"path"
	"strings"

	"github.com/lmenezes/cerebro/internal/config"
)

const AnonymousSubject = "anonymous"

type Request struct {
	Resource string
	Action   string
	Object   string
	System   bool
}

type Authorizer struct {
	enabled     bool
	defaultRole string
	policies    []config.RBACPolicy
	bindings    map[string][]string
}

func New(cfg config.RBAC) *Authorizer {
	bindings := map[string][]string{}
	for _, binding := range cfg.Bindings {
		bindings[binding.Subject] = append(bindings[binding.Subject], binding.Role)
	}
	return &Authorizer{
		enabled:     cfg.Enabled,
		defaultRole: cfg.DefaultRole,
		policies:    cfg.Policies,
		bindings:    bindings,
	}
}

func (a *Authorizer) Enabled() bool {
	return a != nil && a.enabled
}

func (a *Authorizer) Allow(subject string, groups []string, req Request) bool {
	if !a.Enabled() {
		return true
	}
	if req.System {
		return true
	}
	subjects := a.subjects(subject, groups)
	allowed := false
	for _, policy := range a.policies {
		if !matchesAny(policy.Subject, subjects) ||
			!wildcardMatch(policy.Resource, req.Resource) ||
			!wildcardMatch(policy.Action, req.Action) ||
			!wildcardMatch(policy.Object, req.Object) {
			continue
		}
		if policy.Effect == "deny" {
			return false
		}
		if policy.Effect == "allow" {
			allowed = true
		}
	}
	return allowed
}

func (a *Authorizer) subjects(subject string, groups []string) []string {
	if subject == "" {
		subject = AnonymousSubject
	}
	out := []string{subject, "user:" + subject}
	for _, group := range groups {
		if group == "" {
			continue
		}
		out = append(out, group, "group:"+group)
	}
	if a.defaultRole != "" {
		out = append(out, a.defaultRole)
	}
	out = append(out, a.bindings[subject]...)
	out = append(out, a.bindings["user:"+subject]...)
	for _, group := range groups {
		out = append(out, a.bindings[group]...)
		out = append(out, a.bindings["group:"+group]...)
	}
	return out
}

func matchesAny(pattern string, values []string) bool {
	for _, value := range values {
		if wildcardMatch(pattern, value) {
			return true
		}
	}
	return false
}

func wildcardMatch(pattern, value string) bool {
	if pattern == "*" || pattern == value {
		return true
	}
	ok, err := path.Match(pattern, value)
	return err == nil && ok
}

func Classify(method, requestPath string) Request {
	return classify(method, requestPath)
}

func classify(method, requestPath string) Request {
	action := actionForMethod(method)
	object := "*"
	cleaned := strings.Trim(requestPath, "/")
	segments := strings.Split(cleaned, "/")
	if len(segments) == 0 || segments[0] == "" {
		return Request{Resource: "app", Action: action, Object: object}
	}
	if segments[0] == "connect" {
		return Request{Resource: "ui", Action: action, Object: object, System: true}
	}
	if segments[0] != "clusters" || len(segments) < 3 {
		return Request{Resource: segments[0], Action: action, Object: object}
	}

	cluster := segments[1]
	rest := segments[2:]
	object = cluster
	switch rest[0] {
	case "cluster_changes", "navbar":
		return Request{Resource: "ui", Action: "read", Object: cluster, System: true}
	case "nodes", "overview":
		return classifyClusterResource(method, cluster, rest)
	case "commons":
		return classifyCommons(method, cluster, rest)
	case "rest":
		if method == http.MethodPost {
			return Request{Resource: "rest", Action: "execute", Object: cluster}
		}
		return Request{Resource: "rest", Action: action, Object: cluster}
	case "create_index":
		return Request{Resource: "indices", Action: "create", Object: targetObject(cluster, rest, 1)}
	case "index_settings":
		return Request{Resource: "indices", Action: actionForMethod(method), Object: targetObject(cluster, rest, 1)}
	case "data_explorer":
		if len(rest) > 2 && rest[2] == "documents" {
			return Request{Resource: "documents", Action: "write", Object: targetObject(cluster, rest, 1)}
		}
		return Request{Resource: "documents", Action: "read", Object: targetObject(cluster, rest, 1)}
	case "analysis":
		if method == http.MethodPost {
			return Request{Resource: "analysis", Action: "execute", Object: targetObject(cluster, rest, 2)}
		}
		return Request{Resource: "analysis", Action: "read", Object: targetObject(cluster, rest, 2)}
	case "cat":
		return Request{Resource: "cat", Action: "read", Object: targetObject(cluster, rest, 1)}
	case "aliases":
		return Request{Resource: "aliases", Action: action, Object: cluster}
	case "templates":
		if len(rest) >= 3 {
			return Request{Resource: "templates", Action: action, Object: cluster + "/" + rest[2]}
		}
		return Request{Resource: "templates", Action: action, Object: cluster}
	case "repositories":
		return Request{Resource: "repositories", Action: action, Object: targetObject(cluster, rest, 1)}
	case "snapshots":
		if len(rest) >= 4 && rest[3] == "restore" {
			return Request{Resource: "snapshots", Action: "restore", Object: cluster + "/" + rest[1] + "/" + rest[2]}
		}
		if len(rest) >= 3 {
			return Request{Resource: "snapshots", Action: action, Object: cluster + "/" + rest[1] + "/" + rest[2]}
		}
		return Request{Resource: "snapshots", Action: action, Object: targetObject(cluster, rest, 1)}
	case "data_streams":
		return classifyDataStreams(method, cluster, rest)
	case "ilm":
		return Request{Resource: "ilm", Action: action, Object: targetObject(cluster, rest, 2)}
	case "cluster_settings":
		return Request{Resource: "cluster_settings", Action: action, Object: cluster}
	default:
		return Request{Resource: rest[0], Action: action, Object: object}
	}
}

func classifyClusterResource(method, cluster string, rest []string) Request {
	if rest[0] == "overview" && len(rest) > 1 {
		if rest[1] == "shard_allocation" {
			if method == http.MethodDelete {
				return Request{Resource: "shard_allocation", Action: "enable", Object: cluster}
			}
			return Request{Resource: "shard_allocation", Action: "disable", Object: cluster}
		}
		if rest[1] == "indices" {
			object := targetObject(cluster, rest, 2)
			if len(rest) >= 6 && rest[5] == "relocation" {
				return Request{Resource: "shards", Action: "relocate", Object: object}
			}
			if method == http.MethodDelete {
				return Request{Resource: "indices", Action: "delete", Object: object}
			}
			if len(rest) >= 4 {
				switch rest[3] {
				case "cache":
					return Request{Resource: "indices", Action: "clear_cache", Object: object}
				case "forcemerge":
					return Request{Resource: "indices", Action: "force_merge", Object: object}
				default:
					return Request{Resource: "indices", Action: rest[3], Object: object}
				}
			}
			return Request{Resource: "indices", Action: actionForMethod(method), Object: object}
		}
	}
	return Request{Resource: rest[0], Action: actionForMethod(method), Object: cluster}
}

func classifyCommons(method, cluster string, rest []string) Request {
	if len(rest) >= 2 && rest[1] == "indices" {
		return Request{Resource: "indices", Action: actionForMethod(method), Object: targetObject(cluster, rest, 2)}
	}
	if len(rest) >= 2 && rest[1] == "nodes" {
		return Request{Resource: "nodes", Action: actionForMethod(method), Object: targetObject(cluster, rest, 2)}
	}
	return Request{Resource: "ui", Action: actionForMethod(method), Object: cluster, System: true}
}

func classifyDataStreams(method, cluster string, rest []string) Request {
	object := targetObject(cluster, rest, 1)
	if len(rest) >= 3 {
		switch rest[2] {
		case "rollover":
			return Request{Resource: "data_streams", Action: "rollover", Object: object}
		case "lifecycle":
			return Request{Resource: "data_streams", Action: "update_lifecycle", Object: object}
		case "ilm":
			if method == http.MethodDelete {
				return Request{Resource: "data_streams", Action: "detach_ilm", Object: object}
			}
			return Request{Resource: "data_streams", Action: "attach_ilm", Object: object}
		}
	}
	return Request{Resource: "data_streams", Action: actionForMethod(method), Object: object}
}

func actionForMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead:
		return "read"
	case http.MethodDelete:
		return "delete"
	default:
		return "write"
	}
}

func targetObject(cluster string, parts []string, index int) string {
	if len(parts) <= index || parts[index] == "" {
		return cluster
	}
	return cluster + "/" + parts[index]
}
