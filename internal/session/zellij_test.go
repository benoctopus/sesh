package session

import (
	"os"
	"testing"
)

func TestZellijManager_Name(t *testing.T) {
	mgr := NewZellijManager()
	if mgr.Name() != string(BackendZellij) {
		t.Errorf("Name() = %q, want %q", mgr.Name(), BackendZellij)
	}
}

func TestZellijManager_IsInsideSession(t *testing.T) {
	// Save original environment
	originalZellij := os.Getenv("ZELLIJ")
	defer func() {
		if originalZellij != "" {
			os.Setenv("ZELLIJ", originalZellij)
		} else {
			os.Unsetenv("ZELLIJ")
		}
	}()

	mgr := NewZellijManager()

	t.Run("not inside zellij", func(t *testing.T) {
		os.Unsetenv("ZELLIJ")
		if mgr.IsInsideSession() {
			t.Error("IsInsideSession() = true, want false")
		}
	})

	t.Run("inside zellij", func(t *testing.T) {
		os.Setenv("ZELLIJ", "1")
		if !mgr.IsInsideSession() {
			t.Error("IsInsideSession() = false, want true")
		}
	})
}

func TestZellijManager_GetCurrentSessionName(t *testing.T) {
	// Save original environment
	originalZellij := os.Getenv("ZELLIJ")
	originalSessionName := os.Getenv("ZELLIJ_SESSION_NAME")
	defer func() {
		if originalZellij != "" {
			os.Setenv("ZELLIJ", originalZellij)
		} else {
			os.Unsetenv("ZELLIJ")
		}
		if originalSessionName != "" {
			os.Setenv("ZELLIJ_SESSION_NAME", originalSessionName)
		} else {
			os.Unsetenv("ZELLIJ_SESSION_NAME")
		}
	}()

	mgr := NewZellijManager()

	t.Run("not inside zellij", func(t *testing.T) {
		os.Unsetenv("ZELLIJ")
		os.Unsetenv("ZELLIJ_SESSION_NAME")
		name, err := mgr.GetCurrentSessionName()
		if err != nil {
			t.Errorf("GetCurrentSessionName() returned error: %v", err)
		}
		if name != "" {
			t.Errorf("GetCurrentSessionName() = %q, want empty string", name)
		}
	})

	t.Run("inside zellij with session name", func(t *testing.T) {
		os.Setenv("ZELLIJ", "1")
		os.Setenv("ZELLIJ_SESSION_NAME", "test-session")
		name, err := mgr.GetCurrentSessionName()
		if err != nil {
			t.Errorf("GetCurrentSessionName() returned error: %v", err)
		}
		if name != "test-session" {
			t.Errorf("GetCurrentSessionName() = %q, want %q", name, "test-session")
		}
	})
}

func TestParseZellijList(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "empty output",
			output: "",
			want:   []string{},
		},
		{
			name:   "single session",
			output: "session1",
			want:   []string{"session1"},
		},
		{
			name:   "multiple sessions",
			output: "session1\nsession2\nsession3",
			want:   []string{"session1", "session2", "session3"},
		},
		{
			name:   "sessions with extra info",
			output: "session1 (created: 2024-01-01)\nsession2 (created: 2024-01-02)",
			want:   []string{"session1", "session2"},
		},
		{
			name:   "output with header",
			output: "ACTIVE SESSIONS:\nsession1\nsession2",
			want:   []string{"session1", "session2"},
		},
		{
			name:   "output with whitespace",
			output: "  session1  \n  session2  \n",
			want:   []string{"session1", "session2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseZellijList(tt.output)
			if len(got) != len(tt.want) {
				t.Errorf("parseZellijList() returned %d sessions, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseZellijList()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewSessionManager_Zellij(t *testing.T) {
	t.Run("creates zellij manager", func(t *testing.T) {
		mgr, err := NewSessionManager("zellij")
		if err != nil {
			t.Fatalf("NewSessionManager(\"zellij\") returned error: %v", err)
		}
		if mgr == nil {
			t.Fatal("NewSessionManager(\"zellij\") returned nil")
		}
		if mgr.Name() != string(BackendZellij) {
			t.Errorf("manager.Name() = %q, want %q", mgr.Name(), BackendZellij)
		}
	})
}
