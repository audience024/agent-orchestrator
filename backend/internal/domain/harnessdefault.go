package domain

// HarnessDefault is a daemon-owned picker preference for one harness.
// DisabledModels lists the models clients leave out of their pickers; empty
// means every catalog model, and models a catalog adds later are offered.
// DefaultModel names the model id preferred by default for this harness when
// neither project settings nor an explicit request names one; empty keeps the
// harness CLI's native default. Hidden leaves the harness itself out of agent
// and model pickers. Rows saved by earlier builds may carry model, effort, and
// enabledModels keys; they are ignored.
type HarnessDefault struct {
	DisabledModels []string `json:"disabledModels,omitempty" maxItems:"20000"`
	DefaultModel   string   `json:"defaultModel,omitempty"`
	Hidden         bool     `json:"hidden,omitempty"`
}

// IsZero reports whether the preference holds nothing worth storing.
func (d HarnessDefault) IsZero() bool {
	return len(d.DisabledModels) == 0 && d.DefaultModel == "" && !d.Hidden
}
