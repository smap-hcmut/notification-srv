package websocket

import (
	"encoding/json"
	"testing"
)

func TestMessageTypeProjectProgress(t *testing.T) {
	if MessageTypeProjectProgress != "project_progress" {
		t.Errorf("expected 'project_progress', got '%s'", MessageTypeProjectProgress)
	}
}

func TestMessageTypeProjectCompleted(t *testing.T) {
	if MessageTypeProjectCompleted != "project_completed" {
		t.Errorf("expected 'project_completed', got '%s'", MessageTypeProjectCompleted)
	}
}

func TestProgressPayload_JSONRoundtrip(t *testing.T) {
	original := ProgressPayload{
		ProjectID:       "proj-123",
		Status:          "processing",
		Total:           100,
		Done:            50,
		Errors:          2,
		ProgressPercent: 50.0,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded ProgressPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify roundtrip
	if decoded.ProjectID != original.ProjectID {
		t.Errorf("ProjectID mismatch: got %s, want %s", decoded.ProjectID, original.ProjectID)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status mismatch: got %s, want %s", decoded.Status, original.Status)
	}
	if decoded.Total != original.Total {
		t.Errorf("Total mismatch: got %d, want %d", decoded.Total, original.Total)
	}
	if decoded.Done != original.Done {
		t.Errorf("Done mismatch: got %d, want %d", decoded.Done, original.Done)
	}
	if decoded.Errors != original.Errors {
		t.Errorf("Errors mismatch: got %d, want %d", decoded.Errors, original.Errors)
	}
	if decoded.ProgressPercent != original.ProgressPercent {
		t.Errorf("ProgressPercent mismatch: got %f, want %f", decoded.ProgressPercent, original.ProgressPercent)
	}
}

func TestProgressPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payload ProgressPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: ProgressPayload{
				ProjectID:       "proj-123",
				Status:          "processing",
				Total:           100,
				Done:            50,
				Errors:          0,
				ProgressPercent: 50.0,
			},
			wantErr: false,
		},
		{
			name: "empty project ID",
			payload: ProgressPayload{
				ProjectID:       "",
				Status:          "processing",
				Total:           100,
				Done:            50,
				Errors:          0,
				ProgressPercent: 50.0,
			},
			wantErr: true,
		},
		{
			name: "negative total",
			payload: ProgressPayload{
				ProjectID:       "proj-123",
				Status:          "processing",
				Total:           -1,
				Done:            50,
				Errors:          0,
				ProgressPercent: 50.0,
			},
			wantErr: true,
		},
		{
			name: "negative done",
			payload: ProgressPayload{
				ProjectID:       "proj-123",
				Status:          "processing",
				Total:           100,
				Done:            -1,
				Errors:          0,
				ProgressPercent: 50.0,
			},
			wantErr: true,
		},
		{
			name: "negative errors",
			payload: ProgressPayload{
				ProjectID:       "proj-123",
				Status:          "processing",
				Total:           100,
				Done:            50,
				Errors:          -1,
				ProgressPercent: 50.0,
			},
			wantErr: true,
		},
		{
			name: "progress percent over 100",
			payload: ProgressPayload{
				ProjectID:       "proj-123",
				Status:          "processing",
				Total:           100,
				Done:            50,
				Errors:          0,
				ProgressPercent: 101.0,
			},
			wantErr: true,
		},
		{
			name: "negative progress percent",
			payload: ProgressPayload{
				ProjectID:       "proj-123",
				Status:          "processing",
				Total:           100,
				Done:            50,
				Errors:          0,
				ProgressPercent: -1.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
