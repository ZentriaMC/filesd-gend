package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	uuid "github.com/satori/go.uuid"
)

func ConfigureEndpoint(registerCh chan<- *TargetRegisterMessage) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			var newTargetGroup TargetGroup
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&newTargetGroup); err != nil {
				http.Error(w, fmt.Sprintf("invalid json: %s", err), http.StatusBadRequest)
				return
			}

			if len(newTargetGroup.Targets) == 0 || len(newTargetGroup.Labels) == 0 {
				http.Error(w, "targets or labels are empty", http.StatusBadRequest)
				return
			}

			// Ensure that there are no duplicate targets
			seenTargets := make(map[string]bool)
			for _, target := range newTargetGroup.Targets {
				if _, ok := seenTargets[target]; ok {
					http.Error(w, fmt.Sprintf("duplicate target: '%s'", target), http.StatusBadRequest)
					return
				} else {
					seenTargets[target] = true
				}
			}

			// Ensure that there are no duplicate labels
			seenLabels := make(map[string]string)
			for label, v := range newTargetGroup.Labels {
				if v2, ok := seenLabels[label]; ok && v2 == v {
					http.Error(w, fmt.Sprintf("duplicate label: '%s'", label), http.StatusBadRequest)
					return
				} else {
					seenLabels[label] = v
				}
			}

			// Register the target
			targetId := uuid.NewV4()
			updated, err := updateTarget(r.Context(), registerCh, &TargetRegisterMessage{
				Register: true,
				TargetId: targetId,
				Target:   &newTargetGroup,
			})
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to register target: %s", err), http.StatusInternalServerError)
				return
			}

			if updated {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(struct {
					TargetId string `json:"target_id"`
				}{
					targetId.String(),
				})
			} else {
				http.Error(w, "attempted to register duplicate target", http.StatusConflict)
			}
		case "DELETE":
			var targetIdHolder struct {
				TargetId string `json:"target_id"`
			}
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&targetIdHolder); err != nil {
				http.Error(w, fmt.Sprintf("invalid json: %s", err), http.StatusBadRequest)
				return
			}

			targetId, err := uuid.FromString(targetIdHolder.TargetId)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid uuid: %s", err), http.StatusBadRequest)
				return
			}

			updated, err := updateTarget(r.Context(), registerCh, &TargetRegisterMessage{
				Register: false,
				TargetId: targetId,
			})
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to unregister target: %s", err), http.StatusInternalServerError)
				return
			}

			if updated {
				w.WriteHeader(http.StatusNoContent)
			} else {
				http.Error(w, "target did not exist", http.StatusConflict)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
