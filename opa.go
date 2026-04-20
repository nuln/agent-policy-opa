// Package opa implements a PolicyEngine using Open Policy Agent (OPA).
package opa

import (
	"context"
	"fmt"
	"os"

	agent "github.com/nuln/agent-core"
	"github.com/open-policy-agent/opa/rego"
)

func init() {
	agent.RegisterPluginConfigSpec(agent.PluginConfigSpec{
		PluginName:  "opa",
		PluginType:  "policy",
		Description: "Open Policy Agent (OPA) Rego policy engine",
		Fields: []agent.ConfigField{
			{Key: "policy_dir", EnvVar: "OPA_POLICY_DIR", Description: "Directory containing .rego policy files", Type: agent.ConfigFieldString},
		},
	})

	agent.RegisterPolicyEngine("opa", func(opts map[string]any) (agent.PolicyEngine, error) {
		policyDir, _ := opts["policy_dir"].(string)
		if policyDir == "" {
			policyDir = os.Getenv("OPA_POLICY_DIR")
		}
		return New(policyDir), nil
	})
}

// OPAPolicyEngine evaluates OPA Rego policies.
type OPAPolicyEngine struct {
	policyDir string
}

// New creates an OPAPolicyEngine. policyDir is optional.
func New(policyDir string) *OPAPolicyEngine {
	return &OPAPolicyEngine{policyDir: policyDir}
}

// Evaluate evaluates a Rego query (policy) against input.
// policy is either a Rego query string (e.g., "data.authz.allow") or a module text.
func (e *OPAPolicyEngine) Evaluate(ctx context.Context, policy string, input map[string]any) (agent.PolicyDecision, error) {
	opts := []func(*rego.Rego){
		rego.Query(policy),
		rego.Input(input),
	}
	if e.policyDir != "" {
		opts = append(opts, rego.Load([]string{e.policyDir}, nil))
	}

	r := rego.New(opts...)
	rs, err := r.Eval(ctx)
	if err != nil {
		return agent.PolicyDecision{}, fmt.Errorf("opa: evaluate %q: %w", policy, err)
	}
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return agent.PolicyDecision{Allow: false, Reason: "opa: no result"}, nil
	}

	allowed, _ := rs[0].Expressions[0].Value.(bool)
	return agent.PolicyDecision{Allow: allowed, Reason: fmt.Sprintf("opa: %v", allowed)}, nil
}
