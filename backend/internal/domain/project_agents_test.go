package domain

import "testing"

func readinessSnapshot(id string, installation AgentInstallationState, authentication AgentAuthenticationState) AgentReadinessSnapshot {
	return AgentReadinessSnapshot{
		ID:             id,
		Label:          id,
		Installation:   AgentInstallationObservation{State: installation},
		Authentication: AgentAuthenticationObservation{State: authentication},
	}
}

func TestPreferredProjectAgent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		agents []AgentReadinessSnapshot
		want   AgentHarness
	}{
		{
			name: "confirmed agent beats a higher-priority unconfirmed one",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationUnknown, AgentAuthenticationUnknown),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
			},
			want: HarnessCodex,
		},
		{
			name: "an agent found installed beats an unchecked higher-priority one",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationUnknown, AgentAuthenticationUnknown),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationUnknown),
			},
			want: HarnessCodex,
		},
		{
			name: "authorized agent beats a higher-priority configured one",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
			},
			want: HarnessCodex,
		},
		{
			name: "authorized agent outside priority list beats higher-priority configured one",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("agy", AgentInstallationInstalled, AgentAuthenticationAuthorized),
			},
			want: HarnessAgy,
		},
		{
			name: "not_applicable auth shares top tier with authorized",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("aider", AgentInstallationInstalled, AgentAuthenticationNotApplicable),
			},
			want: HarnessAider,
		},
		{
			name: "configured installed agent beats installed agent with unknown auth",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationUnknown),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationConfigured),
			},
			want: HarnessCodex,
		},
		{
			name: "priority decides among configured agents",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("cursor", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
			},
			want: HarnessClaudeCode,
		},
		{
			name: "missing and signed-out agents are left out",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationNotInstalled, AgentAuthenticationUnknown),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationUnauthorized),
				readinessSnapshot("aider", AgentInstallationUnknown, AgentAuthenticationUnknown),
			},
			want: HarnessAider,
		},
		{
			name: "no launchable agent",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationNotInstalled, AgentAuthenticationUnknown),
			},
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := PreferredProjectAgent(tc.agents); got != tc.want {
				t.Fatalf("PreferredProjectAgent = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPreferredInstalledProjectAgent(t *testing.T) {
	unchecked := []AgentReadinessSnapshot{
		readinessSnapshot("claude-code", AgentInstallationUnknown, AgentAuthenticationUnknown),
		readinessSnapshot("codex", AgentInstallationUnknown, AgentAuthenticationUnknown),
	}
	if got := PreferredInstalledProjectAgent(unchecked); got != "" {
		t.Fatalf("with no agent found installed, PreferredInstalledProjectAgent = %q, want none", got)
	}
	found := append(unchecked, readinessSnapshot("cursor", AgentInstallationInstalled, AgentAuthenticationUnknown))
	if got := PreferredInstalledProjectAgent(found); got != HarnessCursor {
		t.Fatalf("PreferredInstalledProjectAgent = %q, want the installed agent %q", got, HarnessCursor)
	}
	signedOut := []AgentReadinessSnapshot{readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationUnauthorized)}
	if got := PreferredInstalledProjectAgent(signedOut); got != "" {
		t.Fatalf("a signed-out agent can't launch, PreferredInstalledProjectAgent = %q", got)
	}
}

func TestPreferredProjectAgentNeedsChecks(t *testing.T) {
	cases := []struct {
		name   string
		agents []AgentReadinessSnapshot
		want   bool
	}{
		{
			name: "unconfirmed top-priority agent needs checks",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationUnknown, AgentAuthenticationUnknown),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
			},
			want: true,
		},
		{
			name: "installed agent with unknown auth above authorized pick needs checks",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationUnknown),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
			},
			want: true,
		},
		{
			name: "configured pick waits when another launchable agent is still unchecked",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("codex", AgentInstallationUnknown, AgentAuthenticationUnknown),
			},
			want: true,
		},
		{
			name: "configured pick waits when lower-priority agent has unknown auth",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("agy", AgentInstallationInstalled, AgentAuthenticationUnknown),
			},
			want: true,
		},
		{
			name: "configured pick needs no checks once no launchable agent is unchecked",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("codex", AgentInstallationNotInstalled, AgentAuthenticationUnknown),
			},
			want: false,
		},
		{
			name: "authorized pick ignores configured higher-priority and unchecked lower-priority agents",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationConfigured),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
				readinessSnapshot("cursor", AgentInstallationUnknown, AgentAuthenticationUnknown),
			},
			want: false,
		},
		{
			name: "confirmed top usable agent needs no checks",
			agents: []AgentReadinessSnapshot{
				readinessSnapshot("claude-code", AgentInstallationNotInstalled, AgentAuthenticationUnknown),
				readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
			},
			want: false,
		},
		{
			name:   "empty list needs no checks",
			agents: nil,
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PreferredProjectAgentNeedsChecks(tc.agents); got != tc.want {
				t.Fatalf("PreferredProjectAgentNeedsChecks = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilterVisibleProjectAgents(t *testing.T) {
	agents := []AgentReadinessSnapshot{
		readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationAuthorized),
		readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
		readinessSnapshot("cursor", AgentInstallationNotInstalled, AgentAuthenticationUnknown),
	}

	filtered := FilterVisibleProjectAgents(agents, map[string]HarnessDefault{
		"claude-code": {Hidden: true},
	})
	if got := PreferredProjectAgent(filtered); got != HarnessCodex {
		t.Fatalf("PreferredProjectAgent(filtered) = %q, want %q", got, HarnessCodex)
	}

	// When every launchable agent is hidden (leaving only not_installed), fall back
	// to the unfiltered snapshot so selection never returns empty.
	allLaunchableHidden := FilterVisibleProjectAgents(agents, map[string]HarnessDefault{
		"claude-code": {Hidden: true},
		"codex":       {Hidden: true},
	})
	if got := PreferredProjectAgent(allLaunchableHidden); got != HarnessClaudeCode {
		t.Fatalf("PreferredProjectAgent(allLaunchableHidden) = %q, want %q", got, HarnessClaudeCode)
	}

	// Hiding an unchecked higher-priority agent prevents unnecessary checks.
	hiddenUnchecked := FilterVisibleProjectAgents([]AgentReadinessSnapshot{
		readinessSnapshot("claude-code", AgentInstallationUnknown, AgentAuthenticationUnknown),
		readinessSnapshot("codex", AgentInstallationInstalled, AgentAuthenticationAuthorized),
	}, map[string]HarnessDefault{
		"claude-code": {Hidden: true},
	})
	if PreferredProjectAgentNeedsChecks(hiddenUnchecked) {
		t.Fatal("hidden unchecked higher-priority agent should not trigger checks")
	}

	// Hiding the only authorized agent exposes an unchecked visible candidate that needs checks.
	hiddenAuthorized := FilterVisibleProjectAgents([]AgentReadinessSnapshot{
		readinessSnapshot("claude-code", AgentInstallationInstalled, AgentAuthenticationAuthorized),
		readinessSnapshot("codex", AgentInstallationUnknown, AgentAuthenticationUnknown),
	}, map[string]HarnessDefault{
		"claude-code": {Hidden: true},
	})
	if !PreferredProjectAgentNeedsChecks(hiddenAuthorized) {
		t.Fatal("visible unchecked agent should trigger checks when higher-priority authorized agent is hidden")
	}
}

func TestFillRoleAgentsKeepsChosenRoles(t *testing.T) {
	cfg := ProjectConfig{Worker: RoleOverride{Harness: HarnessClaudeCode}}
	if !cfg.FillRoleAgents(HarnessCodex) {
		t.Fatal("expected the empty orchestrator role to be filled")
	}
	if cfg.Worker.Harness != HarnessClaudeCode || cfg.Orchestrator.Harness != HarnessCodex {
		t.Fatalf("FillRoleAgents = worker %q orchestrator %q", cfg.Worker.Harness, cfg.Orchestrator.Harness)
	}
	if cfg.FillRoleAgents(HarnessCursor) {
		t.Fatal("a fully configured project must not change")
	}
	if (&ProjectConfig{}).FillRoleAgents("") {
		t.Fatal("an empty agent must not fill anything")
	}
}
