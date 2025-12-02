package session

import (
	"testing"
)

func TestNewEditorManager(t *testing.T) {
	tests := []struct {
		name    string
		command string
		mode    EditorMode
	}{
		{
			name:    "code open",
			command: "code",
			mode:    EditorModeOpen,
		},
		{
			name:    "code workspace",
			command: "code",
			mode:    EditorModeWorkspace,
		},
		{
			name:    "code replace",
			command: "code",
			mode:    EditorModeReplace,
		},
		{
			name:    "cursor open",
			command: "cursor",
			mode:    EditorModeOpen,
		},
		{
			name:    "zed open",
			command: "zed",
			mode:    EditorModeOpen,
		},
		{
			name:    "zed reuse",
			command: "zed",
			mode:    EditorModeReuse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewEditorManager(tt.command, tt.mode)
			if mgr.command != tt.command {
				t.Errorf("command = %q, want %q", mgr.command, tt.command)
			}
			if mgr.mode != tt.mode {
				t.Errorf("mode = %q, want %q", mgr.mode, tt.mode)
			}
		})
	}
}

func TestEditorManager_Name(t *testing.T) {
	tests := []struct {
		command string
		mode    EditorMode
		want    string
	}{
		{"code", EditorModeOpen, "code:open"},
		{"code", EditorModeWorkspace, "code:workspace"},
		{"code", EditorModeReplace, "code:replace"},
		{"cursor", EditorModeOpen, "cursor:open"},
		{"cursor", EditorModeWorkspace, "cursor:workspace"},
		{"cursor", EditorModeReplace, "cursor:replace"},
		{"zed", EditorModeOpen, "zed:open"},
		{"zed", EditorModeReuse, "zed:reuse"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			mgr := NewEditorManager(tt.command, tt.mode)
			if got := mgr.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEditorManager_IsInsideSession(t *testing.T) {
	mgr := NewEditorManager("code", EditorModeOpen)
	if mgr.IsInsideSession() {
		t.Error("IsInsideSession() = true, want false")
	}
}

func TestEditorManager_Exists(t *testing.T) {
	mgr := NewEditorManager("code", EditorModeOpen)
	exists, err := mgr.Exists("test")
	if err != nil {
		t.Errorf("Exists() returned error: %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}
}

func TestEditorManager_List(t *testing.T) {
	mgr := NewEditorManager("code", EditorModeOpen)
	_, err := mgr.List()
	if err == nil {
		t.Error("List() returned nil, want error")
	}
}

func TestEditorManager_Delete(t *testing.T) {
	mgr := NewEditorManager("code", EditorModeOpen)
	err := mgr.Delete("test")
	if err == nil {
		t.Error("Delete() returned nil, want error")
	}
}

func TestEditorManager_GetCurrentSessionName(t *testing.T) {
	mgr := NewEditorManager("code", EditorModeOpen)
	_, err := mgr.GetCurrentSessionName()
	if err == nil {
		t.Error("GetCurrentSessionName() returned nil, want error")
	}
}

func TestEditorManager_buildArgs(t *testing.T) {
	tests := []struct {
		name    string
		command string
		mode    EditorMode
		path    string
		want    []string
	}{
		// VS Code / Cursor
		{"code open", "code", EditorModeOpen, "/path/to/project", []string{"/path/to/project"}},
		{"code workspace", "code", EditorModeWorkspace, "/path/to/project", []string{"--add", "/path/to/project"}},
		{"code replace", "code", EditorModeReplace, "/path/to/project", []string{"-r", "/path/to/project"}},
		{"cursor open", "cursor", EditorModeOpen, "/path/to/project", []string{"/path/to/project"}},
		{"cursor workspace", "cursor", EditorModeWorkspace, "/path/to/project", []string{"--add", "/path/to/project"}},
		{"cursor replace", "cursor", EditorModeReplace, "/path/to/project", []string{"-r", "/path/to/project"}},
		// Zed (uses different flags)
		{"zed open", "zed", EditorModeOpen, "/path/to/project", []string{"-n", "/path/to/project"}},
		{"zed reuse", "zed", EditorModeReuse, "/path/to/project", []string{"/path/to/project"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewEditorManager(tt.command, tt.mode)
			got := mgr.buildArgs(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("buildArgs() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseEditorBackend(t *testing.T) {
	tests := []struct {
		backend     string
		wantCommand string
		wantMode    EditorMode
		wantErr     bool
	}{
		{"code:open", "code", EditorModeOpen, false},
		{"code:workspace", "code", EditorModeWorkspace, false},
		{"code:replace", "code", EditorModeReplace, false},
		{"cursor:open", "cursor", EditorModeOpen, false},
		{"cursor:workspace", "cursor", EditorModeWorkspace, false},
		{"cursor:replace", "cursor", EditorModeReplace, false},
		{"zed:open", "zed", EditorModeOpen, false},
		{"zed:reuse", "zed", EditorModeReuse, false},
		{"invalid", "", "", true},
		{"code:invalid", "", "", true},
		{"zed:invalid", "", "", true},
		{"tmux", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.backend, func(t *testing.T) {
			command, mode, err := ParseEditorBackend(tt.backend)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEditorBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if command != tt.wantCommand {
					t.Errorf("command = %q, want %q", command, tt.wantCommand)
				}
				if mode != tt.wantMode {
					t.Errorf("mode = %q, want %q", mode, tt.wantMode)
				}
			}
		})
	}
}

func TestIsEditorBackend(t *testing.T) {
	tests := []struct {
		backend string
		want    bool
	}{
		{"code:open", true},
		{"code:workspace", true},
		{"code:replace", true},
		{"cursor:open", true},
		{"cursor:workspace", true},
		{"cursor:replace", true},
		{"zed:open", true},
		{"zed:reuse", true},
		{"tmux", false},
		{"zellij", false},
		{"auto", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.backend, func(t *testing.T) {
			if got := IsEditorBackend(tt.backend); got != tt.want {
				t.Errorf("IsEditorBackend(%q) = %v, want %v", tt.backend, got, tt.want)
			}
		})
	}
}

func TestGetBackendName_EditorBackends(t *testing.T) {
	tests := []struct {
		backend BackendType
		want    string
	}{
		{BackendCodeOpen, "VS Code (new window)"},
		{BackendCodeWorkspace, "VS Code (add to workspace)"},
		{BackendCodeReplace, "VS Code (replace window)"},
		{BackendCursorOpen, "Cursor (new window)"},
		{BackendCursorWorkspace, "Cursor (add to workspace)"},
		{BackendCursorReplace, "Cursor (replace window)"},
		{BackendZedOpen, "Zed (new window)"},
		{BackendZedReuse, "Zed (reuse window)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.backend), func(t *testing.T) {
			if got := GetBackendName(tt.backend); got != tt.want {
				t.Errorf("GetBackendName(%q) = %q, want %q", tt.backend, got, tt.want)
			}
		})
	}
}
