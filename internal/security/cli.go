package security

import (
	"fmt"
	"regexp"
	"strings"
)

// k8sPodListJSONPath must stay identical to internal/source/k8s.go podListJSONPath.
const k8sPodListJSONPath = `{range .items[*]}{.metadata.name}{"\n"}{end}`

var (
	dns1123Label = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	digitsOnly   = regexp.MustCompile(`^[0-9]+$`)
	// dockerNamePattern matches allowlisted docker container tokens passed as argv.
	dockerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,200}$`)
	// cfAppPattern matches cf app argv tokens (aligned with internal/source/cf.go).
	cfAppPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
)

func argClean(args []string) error {
	for _, a := range args {
		if strings.ContainsAny(a, "\n\r\x00") {
			return fmt.Errorf("security: argument contains illegal control characters")
		}
	}
	return nil
}

// ValidateKubectlArgs enforces the allowlist for kubectl invocations used by log-mcp.
func ValidateKubectlArgs(args []string) error {
	if err := argClean(args); err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("security: empty kubectl argv")
	}
	switch args[0] {
	case "get":
		return validateKubectlGet(args)
	case "logs":
		return validateKubectlLogs(args)
	default:
		return fmt.Errorf("security: kubectl subcommand %q is not allowlisted", args[0])
	}
}

func validateKubectlGet(args []string) error {
	// get pods -n <ns> -o jsonpath=<fixed>
	if len(args) != 6 || args[1] != "pods" || args[2] != "-n" || args[4] != "-o" {
		return fmt.Errorf("security: kubectl get shape not allowlisted")
	}
	ns := args[3]
	if !dns1123Label.MatchString(ns) || len(ns) > 63 {
		return fmt.Errorf("security: invalid kubectl namespace argument")
	}
	want := "jsonpath=" + k8sPodListJSONPath
	if args[5] != want {
		return fmt.Errorf("security: kubectl get -o not allowlisted")
	}
	return nil
}

func validateKubectlLogs(args []string) error {
	// logs <pod> -n <ns> [-c <container>] [--tail <n>]
	if len(args) < 4 || args[0] != "logs" || args[2] != "-n" {
		return fmt.Errorf("security: kubectl logs shape not allowlisted")
	}
	pod := args[1]
	ns := args[3]
	if !dns1123Label.MatchString(pod) || len(pod) > 63 {
		return fmt.Errorf("security: invalid kubectl pod name")
	}
	if !dns1123Label.MatchString(ns) || len(ns) > 63 {
		return fmt.Errorf("security: invalid kubectl namespace")
	}
	i := 4
	for i < len(args) {
		switch {
		case args[i] == "-c" && i+1 < len(args):
			c := args[i+1]
			if !dns1123Label.MatchString(c) || len(c) > 63 {
				return fmt.Errorf("security: invalid kubectl container name")
			}
			i += 2
		case args[i] == "--tail" && i+1 < len(args):
			if !digitsOnly.MatchString(args[i+1]) {
				return fmt.Errorf("security: kubectl --tail must be numeric")
			}
			i += 2
		default:
			return fmt.Errorf("security: kubectl logs extra argument %q not allowlisted", args[i])
		}
	}
	return nil
}

// ValidateDockerArgs enforces the allowlist for docker invocations used by log-mcp.
func ValidateDockerArgs(args []string) error {
	if err := argClean(args); err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("security: empty docker argv")
	}
	switch args[0] {
	case "ps":
		if len(args) != 4 || args[1] != "-a" || args[2] != "--format" || args[3] != "{{.Names}}" {
			return fmt.Errorf("security: docker ps shape not allowlisted")
		}
		return nil
	case "logs":
		if len(args) < 2 {
			return fmt.Errorf("security: docker logs missing container")
		}
		if !dockerNamePattern.MatchString(args[1]) {
			return fmt.Errorf("security: invalid docker container name")
		}
		i := 2
		for i < len(args) {
			if args[i] == "--tail" && i+1 < len(args) {
				if !digitsOnly.MatchString(args[i+1]) {
					return fmt.Errorf("security: docker --tail must be numeric")
				}
				i += 2
				continue
			}
			return fmt.Errorf("security: docker logs argument %q not allowlisted", args[i])
		}
		return nil
	default:
		return fmt.Errorf("security: docker subcommand %q is not allowlisted", args[0])
	}
}

// ValidateCFArgs enforces the allowlist for cf invocations used by log-mcp.
func ValidateCFArgs(args []string) error {
	if err := argClean(args); err != nil {
		return err
	}
	switch len(args) {
	case 1:
		if args[0] != "apps" {
			return fmt.Errorf("security: cf one-arg invocation not allowlisted")
		}
		return nil
	case 3:
		if args[0] != "logs" || args[2] != "--recent" {
			return fmt.Errorf("security: cf logs shape not allowlisted")
		}
		app := args[1]
		if len(app) > 63 || !cfAppPattern.MatchString(app) || strings.HasSuffix(app, "-") {
			return fmt.Errorf("security: invalid cf app name")
		}
		return nil
	default:
		return fmt.Errorf("security: cf argv length %d not allowlisted", len(args))
	}
}
