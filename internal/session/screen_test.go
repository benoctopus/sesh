package session

import (
	"os"
	"reflect"
	"testing"
)

func TestScreenManager_Name(t *testing.T) {
	mgr := NewScreenManager()
	want := "screen"
	if got := mgr.Name(); got != want {
		t.Errorf("ScreenManager.Name() = %q, want %q", got, want)
	}
}

func TestScreenManager_IsInsideSession(t *testing.T) {
	// Save original environment
	originalSTY := os.Getenv("STY")
	defer func() {
		if originalSTY != "" {
			os.Setenv("STY", originalSTY)
		} else {
			os.Unsetenv("STY")
		}
	}()

	mgr := NewScreenManager()

	t.Run("not inside screen session", func(t *testing.T) {
		os.Unsetenv("STY")
		if mgr.IsInsideSession() {
			t.Error("IsInsideSession() = true, want false")
		}
	})

	t.Run("inside screen session", func(t *testing.T) {
		os.Setenv("STY", "12345.mysession")
		if !mgr.IsInsideSession() {
			t.Error("IsInsideSession() = false, want true")
		}
	})
}

func TestScreenManager_GetCurrentSessionName(t *testing.T) {
	// Save original environment
	originalSTY := os.Getenv("STY")
	defer func() {
		if originalSTY != "" {
			os.Setenv("STY", originalSTY)
		} else {
			os.Unsetenv("STY")
		}
	}()

	mgr := NewScreenManager()

	t.Run("not inside screen session", func(t *testing.T) {
		os.Unsetenv("STY")
		name, err := mgr.GetCurrentSessionName()
		if err != nil {
			t.Errorf("GetCurrentSessionName() returned error: %v", err)
		}
		if name != "" {
			t.Errorf("GetCurrentSessionName() = %q, want empty string", name)
		}
	})

	t.Run("inside screen session", func(t *testing.T) {
		os.Setenv("STY", "12345.mysession")
		name, err := mgr.GetCurrentSessionName()
		if err != nil {
			t.Errorf("GetCurrentSessionName() returned error: %v", err)
		}
		if name != "mysession" {
			t.Errorf("GetCurrentSessionName() = %q, want %q", name, "mysession")
		}
	})

	t.Run("complex session name", func(t *testing.T) {
		os.Setenv("STY", "67890.my-complex-session.hostname")
		name, err := mgr.GetCurrentSessionName()
		if err != nil {
			t.Errorf("GetCurrentSessionName() returned error: %v", err)
		}
		want := "my-complex-session.hostname"
		if name != want {
			t.Errorf("GetCurrentSessionName() = %q, want %q", name, want)
		}
	})

	t.Run("invalid STY format", func(t *testing.T) {
		os.Setenv("STY", "invalid")
		_, err := mgr.GetCurrentSessionName()
		if err == nil {
			t.Error("GetCurrentSessionName() with invalid STY format should return error")
		}
	})
}

func TestScreenManager_Switch(t *testing.T) {
	mgr := NewScreenManager()
	err := mgr.Switch("test")
	if err == nil {
		t.Error("Switch() should return error as screen doesn't support switching")
	}
}

func TestParseScreenList(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "no sessions",
			output: "No Sockets found in /var/run/screen/S-user.\n",
			want:   []string{},
		},
		{
			name: "single attached session",
			output: `There is a screen on:
	12345.mysession	(Attached)
1 Socket in /var/run/screen/S-user.
`,
			want: []string{"mysession"},
		},
		{
			name: "single detached session",
			output: `There is a screen on:
	12345.mysession	(Detached)
1 Socket in /var/run/screen/S-user.
`,
			want: []string{"mysession"},
		},
		{
			name: "multiple sessions mixed",
			output: `There are screens on:
	12345.session-one	(Attached)
	67890.session-two	(Detached)
	11111.my-project	(Attached)
3 Sockets in /var/run/screen/S-user.
`,
			want: []string{"session-one", "session-two", "my-project"},
		},
		{
			name: "sessions with complex names",
			output: `There are screens on:
	12345.my.session.name	(Attached)
	67890.test_session	(Detached)
2 Sockets in /var/run/screen/S-user.
`,
			want: []string{"my.session.name", "test_session"},
		},
		{
			name: "sessions with extra whitespace",
			output: `There are screens on:
		12345.session1	(Attached)
  67890.session2		(Detached)
	11111.session3	(Attached)
3 Sockets in /var/run/screen/S-user.
`,
			want: []string{"session1", "session2", "session3"},
		},
		{
			name: "real screen output example",
			output: `There are screens on:
	31462.pts-1.hostname	(Attached)
	31234.dev-session	(Detached)
	30999.project-foo	(Attached)
3 Sockets in /var/run/screen/S-user.
`,
			want: []string{"pts-1.hostname", "dev-session", "project-foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseScreenList(tt.output)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseScreenList() = %v, want %v", got, tt.want)
			}
		})
	}
}
