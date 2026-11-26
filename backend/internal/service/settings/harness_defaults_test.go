package settings

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

type harnessSettingsStore struct {
	hibernationSettingsStore
	defaults map[string]domain.HarnessDefault
}

func (s *harnessSettingsStore) GetAppSettings(context.Context) (Snapshot, error) {
	return Snapshot{HarnessDefaults: s.defaults}, s.err
}
func (s *harnessSettingsStore) SetHarnessDefault(_ context.Context, agent domain.AgentHarness, value domain.HarnessDefault, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	if value.IsZero() {
		delete(s.defaults, string(agent))
	} else {
		s.defaults[string(agent)] = value
	}
	return nil
}

func harnessDefault(t *testing.T, svc *Service, agent domain.AgentHarness) domain.HarnessDefault {
	t.Helper()
	snapshot, err := svc.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.HarnessDefaults[string(agent)]
}

func TestHarnessDefaultsValidateAndSurviveServiceRestart(t *testing.T) {
	ctx := context.Background()
	store := &harnessSettingsStore{defaults: make(map[string]domain.HarnessDefault)}
	svc := New(store, nil, Offering{}, nil)
	if _, err := svc.SetHarnessDefault(ctx, "not-an-agent", domain.HarnessDefault{DisabledModels: []string{"m"}}); !errors.Is(err, ErrUnknownHarness) {
		t.Fatalf("unknown harness: %v", err)
	}
	if _, err := svc.SetHarnessDefault(ctx, domain.HarnessCodex, domain.HarnessDefault{DisabledModels: []string{"m"}}); err != nil {
		t.Fatal(err)
	}
	restarted := New(store, nil, Offering{}, nil)
	if got := harnessDefault(t, restarted, domain.HarnessCodex); !reflect.DeepEqual(got, domain.HarnessDefault{DisabledModels: []string{"m"}}) {
		t.Fatalf("restart = %+v", got)
	}
	store.err = errors.New("write failed")
	if _, err := svc.SetHarnessDefault(ctx, domain.HarnessCodex, domain.HarnessDefault{}); err == nil {
		t.Fatal("write failure ignored")
	}
	store.err = nil
	if _, err := svc.SetHarnessDefault(ctx, domain.HarnessCodex, domain.HarnessDefault{}); err != nil {
		t.Fatal(err)
	}
	if got := harnessDefault(t, svc, domain.HarnessCodex); !got.IsZero() {
		t.Fatalf("reset = %+v", got)
	}
}

func TestHarnessDisabledModelsAreNormalizedAndBounded(t *testing.T) {
	ctx := context.Background()
	store := &harnessSettingsStore{defaults: make(map[string]domain.HarnessDefault)}
	svc := New(store, nil, Offering{}, nil)
	for _, ids := range [][]string{{" "}, {strings.Repeat("m", 257)}, {"a\nb"}, make([]string, maxDisabledModels+1)} {
		if _, err := svc.SetHarnessDefault(ctx, domain.HarnessAider, domain.HarnessDefault{DisabledModels: ids}); !errors.Is(err, ErrInvalidHarnessDefault) {
			t.Fatalf("disabled models %q: %v", ids, err)
		}
	}
	if _, err := svc.SetHarnessDefault(ctx, domain.HarnessAider, domain.HarnessDefault{DisabledModels: []string{" gpt-5 ", "claude", "gpt-5"}}); err != nil {
		t.Fatal(err)
	}
	if got := harnessDefault(t, svc, domain.HarnessAider); !reflect.DeepEqual(got.DisabledModels, []string{"gpt-5", "claude"}) {
		t.Fatalf("disabled models = %+v", got)
	}
	if _, err := svc.SetHarnessDefault(ctx, domain.HarnessAider, domain.HarnessDefault{DisabledModels: []string{}}); err != nil {
		t.Fatal(err)
	}
	if _, saved := store.defaults[string(domain.HarnessAider)]; saved {
		t.Fatal("empty disabled models should remove the entry")
	}
}

func TestPatchHarnessDefaultMergesAndResets(t *testing.T) {
	ctx := context.Background()
	store := &harnessSettingsStore{defaults: make(map[string]domain.HarnessDefault)}
	svc := New(store, nil, Offering{}, nil)

	if _, err := svc.PatchHarnessDefault(ctx, "not-an-agent", HarnessDefaultPatch{Hidden: new(true)}); !errors.Is(err, ErrUnknownHarness) {
		t.Fatalf("unknown harness: %v", err)
	}
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{}); !errors.Is(err, ErrInvalidHarnessDefault) {
		t.Fatalf("empty patch: %v", err)
	}

	// Set disabledModels first, then hide without clearing disabledModels.
	models := []string{" gpt-5 ", "gpt-5-mini", "gpt-5"}
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{DisabledModels: &models}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{Hidden: new(true)}); err != nil {
		t.Fatal(err)
	}
	want := domain.HarnessDefault{DisabledModels: []string{"gpt-5", "gpt-5-mini"}, Hidden: true}
	if got := harnessDefault(t, svc, domain.HarnessCodex); !reflect.DeepEqual(got, want) {
		t.Fatalf("after hide = %+v, want %+v", got, want)
	}

	// Clearing disabledModels while hidden keeps the entry because Hidden is still true.
	empty := []string{}
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{DisabledModels: &empty}); err != nil {
		t.Fatal(err)
	}
	wantHiddenOnly := domain.HarnessDefault{Hidden: true}
	if got := harnessDefault(t, svc, domain.HarnessCodex); !reflect.DeepEqual(got, wantHiddenOnly) {
		t.Fatalf("after clearing disabledModels while hidden = %+v, want %+v", got, wantHiddenOnly)
	}
	if _, saved := store.defaults[string(domain.HarnessCodex)]; !saved {
		t.Fatal("hidden-only entry should remain stored")
	}

	// Unhiding with no disabled models removes the entry completely.
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{Hidden: new(false)}); err != nil {
		t.Fatal(err)
	}
	if _, saved := store.defaults[string(domain.HarnessCodex)]; saved {
		t.Fatalf("fully cleared entry should be removed, got %+v", store.defaults)
	}

	// DefaultModel trims, validates, merges without clobbering, and clears on empty string.
	for _, bad := range []string{strings.Repeat("a", 257), "bad\nmodel"} {
		if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{DefaultModel: &bad}); !errors.Is(err, ErrInvalidHarnessDefault) {
			t.Fatalf("invalid defaultModel %q: %v", bad, err)
		}
	}
	dm := " gpt-6.2-astra "
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{DefaultModel: &dm}); err != nil {
		t.Fatal(err)
	}
	if got := svc.DefaultModelForHarness(ctx, domain.HarnessCodex); got != "gpt-6.2-astra" {
		t.Fatalf("DefaultModelForHarness = %q, want gpt-6.2-astra", got)
	}
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{Hidden: new(true)}); err != nil {
		t.Fatal(err)
	}
	if got := harnessDefault(t, svc, domain.HarnessCodex); !reflect.DeepEqual(got, domain.HarnessDefault{DefaultModel: "gpt-6.2-astra", Hidden: true}) {
		t.Fatalf("after hide with defaultModel = %+v", got)
	}
	clearModel := ""
	if _, err := svc.PatchHarnessDefault(ctx, domain.HarnessCodex, HarnessDefaultPatch{DefaultModel: &clearModel, Hidden: new(false)}); err != nil {
		t.Fatal(err)
	}
	if _, saved := store.defaults[string(domain.HarnessCodex)]; saved {
		t.Fatalf("clearing defaultModel and hidden should remove entry, got %+v", store.defaults)
	}
}
