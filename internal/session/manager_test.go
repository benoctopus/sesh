package session

import (
	"os"
	"testing"
)

func TestGetBackendName(t *testing.T) {
	tests := []struct {
		name    string
		backend BackendType
		want    string
	}{
		{
			name:    "tmux backend",
			backend: BackendTmux,
			want:    "Tmux",
		},
		{
			name:    "zellij backend",
			backend: BackendZellij,
			want:    "Zellij",
		},
		{
			name:    "screen backend",
			backend: BackendScreen,
			want:    "GNU Screen",
		},
		{
			name:    "none backend",
			backend: BackendNone,
			want:    "None (no session manager)",
		},
		{
			name:    "unknown backend",
			backend: BackendType("unknown"),
			want:    "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBackendName(tt.backend)
			if got != tt.want {
				t.Errorf("GetBackendName(%q) = %q, want %q", tt.backend, got, tt.want)
			}
		})
	}
}

func TestNoneManager(t *testing.T) {
	mgr := NewNoneManager()

	t.Run("Name", func(t *testing.T) {
		if mgr.Name() != string(BackendNone) {
			t.Errorf("Name() = %q, want %q", mgr.Name(), BackendNone)
		}
	})

	t.Run("IsInsideSession", func(t *testing.T) {
		if mgr.IsInsideSession() {
			t.Error("IsInsideSession() = true, want false")
		}
	})

	t.Run("Create returns error", func(t *testing.T) {
		err := mgr.Create("test", "/tmp")
		if err == nil {
			t.Error("Create() returned nil, want error")
		}
	})

	t.Run("Attach returns error", func(t *testing.T) {
		err := mgr.Attach("test")
		if err == nil {
			t.Error("Attach() returned nil, want error")
		}
	})

	t.Run("Switch returns error", func(t *testing.T) {
		err := mgr.Switch("test")
		if err == nil {
			t.Error("Switch() returned nil, want error")
		}
	})

	t.Run("Delete returns error", func(t *testing.T) {
		err := mgr.Delete("test")
		if err == nil {
			t.Error("Delete() returned nil, want error")
		}
	})

	t.Run("List returns empty", func(t *testing.T) {
		sessions, err := mgr.List()
		if err != nil {
			t.Errorf("List() returned error: %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("List() returned %d sessions, want 0", len(sessions))
		}
	})

	t.Run("Exists returns false", func(t *testing.T) {
		exists, err := mgr.Exists("test")
		if err != nil {
			t.Errorf("Exists() returned error: %v", err)
		}
		if exists {
			t.Error("Exists() = true, want false")
		}
	})

	t.Run("IsRunning returns false", func(t *testing.T) {
		running, err := mgr.IsRunning()
		if err != nil {
			t.Errorf("IsRunning() returned error: %v", err)
		}
		if running {
			t.Error("IsRunning() = true, want false")
		}
	})
}

func TestIsInsideTmux(t *testing.T) {
	// Save original environment
	originalTmux := os.Getenv("TMUX")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		} else {
			os.Unsetenv("TMUX")
		}
	}()

	t.Run("not inside tmux", func(t *testing.T) {
		os.Unsetenv("TMUX")
		if IsInsideTmux() {
			t.Error("IsInsideTmux() = true, want false")
		}
	})

	t.Run("inside tmux", func(t *testing.T) {
		os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
		if !IsInsideTmux() {
			t.Error("IsInsideTmux() = false, want true")
		}
	})
}

func TestIsInsideZellij(t *testing.T) {
	// Save original environment
	originalZellij := os.Getenv("ZELLIJ")
	defer func() {
		if originalZellij != "" {
			os.Setenv("ZELLIJ", originalZellij)
		} else {
			os.Unsetenv("ZELLIJ")
		}
	}()

	t.Run("not inside zellij", func(t *testing.T) {
		os.Unsetenv("ZELLIJ")
		if IsInsideZellij() {
			t.Error("IsInsideZellij() = true, want false")
		}
	})

	t.Run("inside zellij", func(t *testing.T) {
		os.Setenv("ZELLIJ", "1")
		if !IsInsideZellij() {
			t.Error("IsInsideZellij() = false, want true")
		}
	})
}

func TestIsInsideScreen(t *testing.T) {
	// Save original environment
	originalSTY := os.Getenv("STY")
	defer func() {
		if originalSTY != "" {
			os.Setenv("STY", originalSTY)
		} else {
			os.Unsetenv("STY")
		}
	}()

	t.Run("not inside screen", func(t *testing.T) {
		os.Unsetenv("STY")
		if IsInsideScreen() {
			t.Error("IsInsideScreen() = true, want false")
		}
	})

	t.Run("inside screen", func(t *testing.T) {
		os.Setenv("STY", "12345.pts-1.hostname")
		if !IsInsideScreen() {
			t.Error("IsInsideScreen() = false, want true")
		}
	})
}

func TestIsInsideAnySession(t *testing.T) {
	// Save original environment
	originalTmux := os.Getenv("TMUX")
	originalZellij := os.Getenv("ZELLIJ")
	originalSTY := os.Getenv("STY")
	defer func() {
		if originalTmux != "" {
			os.Setenv("TMUX", originalTmux)
		} else {
			os.Unsetenv("TMUX")
		}
		if originalZellij != "" {
			os.Setenv("ZELLIJ", originalZellij)
		} else {
			os.Unsetenv("ZELLIJ")
		}
		if originalSTY != "" {
			os.Setenv("STY", originalSTY)
		} else {
			os.Unsetenv("STY")
		}
	}()

	t.Run("not inside any session", func(t *testing.T) {
		os.Unsetenv("TMUX")
		os.Unsetenv("ZELLIJ")
		os.Unsetenv("STY")
		if IsInsideAnySession() {
			t.Error("IsInsideAnySession() = true, want false")
		}
	})

	t.Run("inside tmux", func(t *testing.T) {
		os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
		os.Unsetenv("ZELLIJ")
		os.Unsetenv("STY")
		if !IsInsideAnySession() {
			t.Error("IsInsideAnySession() = false, want true")
		}
	})

	t.Run("inside zellij", func(t *testing.T) {
		os.Unsetenv("TMUX")
		os.Setenv("ZELLIJ", "1")
		os.Unsetenv("STY")
		if !IsInsideAnySession() {
			t.Error("IsInsideAnySession() = false, want true")
		}
	})

	t.Run("inside screen", func(t *testing.T) {
		os.Unsetenv("TMUX")
		os.Unsetenv("ZELLIJ")
		os.Setenv("STY", "12345.pts-1.hostname")
		if !IsInsideAnySession() {
			t.Error("IsInsideAnySession() = false, want true")
		}
	})
}
