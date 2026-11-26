package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	settingssvc "github.com/aoagents/agent-orchestrator/backend/internal/service/settings"
)

type harnessDefaultSettingsService struct {
	hibernationSettingsService
	defaults map[string]domain.HarnessDefault
}

func (s *harnessDefaultSettingsService) Get(context.Context) (settingssvc.Snapshot, error) {
	return settingssvc.Snapshot{HarnessDefaults: s.defaults}, nil
}

func (s *harnessDefaultSettingsService) PatchHarnessDefault(_ context.Context, agent domain.AgentHarness, patch settingssvc.HarnessDefaultPatch) (settingssvc.Snapshot, error) {
	if agent != domain.HarnessCodex {
		return settingssvc.Snapshot{}, settingssvc.ErrUnknownHarness
	}
	next := s.defaults[string(agent)]
	if patch.DisabledModels != nil {
		next.DisabledModels = *patch.DisabledModels
	}
	if patch.Hidden != nil {
		next.Hidden = *patch.Hidden
	}
	if next.IsZero() {
		delete(s.defaults, string(agent))
	} else {
		s.defaults[string(agent)] = next
	}
	return settingssvc.Snapshot{HarnessDefaults: s.defaults}, nil
}

func TestSettingsHarnessDefaultPatchSupportsPartialUpdates(t *testing.T) {
	svc := &harnessDefaultSettingsService{defaults: map[string]domain.HarnessDefault{}}
	c := &SettingsController{Svc: svc}
	patch := func(agent, body string) int {
		router := chi.NewRouter()
		router.Patch("/api/v1/settings/harness-defaults/{agent}", c.setHarnessDefault)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/api/v1/settings/harness-defaults/"+agent, strings.NewReader(body)))
		return w.Code
	}
	for _, body := range []string{`{}`, `{"model":"gpt-5","effort":"low"}`} {
		if code := patch("codex", body); code != http.StatusBadRequest {
			t.Fatalf("body=%s status=%d", body, code)
		}
	}
	if code := patch("not-an-agent", `{"disabledModels":[]}`); code != http.StatusNotFound {
		t.Fatalf("unknown harness status=%d", code)
	}
	// Only disabledModels.
	if code := patch("codex", `{"disabledModels":["gpt-5","gpt-5-mini"]}`); code != http.StatusOK {
		t.Fatalf("disabled models status=%d", code)
	}
	want := domain.HarnessDefault{DisabledModels: []string{"gpt-5", "gpt-5-mini"}}
	if got := svc.defaults["codex"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("after disabled models = %+v", got)
	}
	// Only hidden: preserves disabledModels.
	if code := patch("codex", `{"hidden":true}`); code != http.StatusOK {
		t.Fatalf("hidden status=%d", code)
	}
	want = domain.HarnessDefault{DisabledModels: []string{"gpt-5", "gpt-5-mini"}, Hidden: true}
	if got := svc.defaults["codex"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("after hidden = %+v", got)
	}
	// Both fields together.
	if code := patch("codex", `{"disabledModels":["gpt-5"],"hidden":false}`); code != http.StatusOK {
		t.Fatalf("both fields status=%d", code)
	}
	want = domain.HarnessDefault{DisabledModels: []string{"gpt-5"}, Hidden: false}
	if got := svc.defaults["codex"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("after both fields = %+v", got)
	}
}
