package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestHarnessDefaultIgnoresRetiredKeys(t *testing.T) {
	var got HarnessDefault
	if err := json.Unmarshal([]byte(`{"model":"opus","effort":"high","enabledModels":["sonnet"],"disabledModels":["opus"]}`), &got); err != nil {
		t.Fatal(err)
	}
	if want := (HarnessDefault{DisabledModels: []string{"opus"}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if !(HarnessDefault{}).IsZero() || got.IsZero() {
		t.Fatalf("IsZero misreports %+v", got)
	}
}

func TestHarnessDefaultHiddenSerializationAndIsZero(t *testing.T) {
	hiddenOnly := HarnessDefault{Hidden: true}
	if hiddenOnly.IsZero() {
		t.Fatal("hidden-only HarnessDefault must not be zero")
	}
	raw, err := json.Marshal(hiddenOnly)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"hidden":true}` {
		t.Fatalf("marshal hidden-only = %s, want {\"hidden\":true}", raw)
	}
	var decoded HarnessDefault
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, hiddenOnly) {
		t.Fatalf("round-trip = %+v, want %+v", decoded, hiddenOnly)
	}

	both := HarnessDefault{DisabledModels: []string{"opus"}, Hidden: true}
	if both.IsZero() {
		t.Fatal("HarnessDefault with disabledModels and hidden must not be zero")
	}
	rawBoth, err := json.Marshal(both)
	if err != nil {
		t.Fatal(err)
	}
	var decodedBoth HarnessDefault
	if err := json.Unmarshal(rawBoth, &decodedBoth); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decodedBoth, both) {
		t.Fatalf("round-trip both = %+v, want %+v", decodedBoth, both)
	}

	zero := HarnessDefault{DisabledModels: []string{}, Hidden: false}
	if !zero.IsZero() {
		t.Fatalf("empty disabledModels + hidden=false must be zero: %+v", zero)
	}
}
