package main

type TargetGroup struct {
	TargetId *string           `json:"target_id,omitempty"`
	Targets  []string          `json:"targets"`
	Labels   map[string]string `json:"labels"`
}

func (t *TargetGroup) Eq(o *TargetGroup) bool {
	if len(t.Targets) != len(o.Targets) {
		return false
	}

	if len(t.Labels) != len(o.Labels) {
		return false
	}

	// Compare targets
	targets := make(map[string]bool)
	for _, target := range t.Targets {
		targets[target] = true
	}

	for _, target := range o.Targets {
		if _, ok := targets[target]; ok {
			continue
		}
		return false
	}

	// Compare labels
	for k, v := range t.Labels {
		if v2, ok := o.Labels[k]; ok && v2 == v {
			continue
		}
		return false
	}

	return true
}
