package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProjectJSONMarshaling(t *testing.T) {
	now := time.Now()
	project := &Project{
		ID:          1,
		Name:        "github.com/user/repo",
		RemoteURL:   "https://github.com/user/repo.git",
		LocalPath:   "/home/user/.sesh/github.com/user/repo.git",
		CreatedAt:   now,
		LastFetched: &now,
	}

	// Test marshaling
	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Failed to marshal project: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Project
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal project: %v", err)
	}

	if unmarshaled.ID != project.ID {
		t.Errorf("ID mismatch: got %d, want %d", unmarshaled.ID, project.ID)
	}
	if unmarshaled.Name != project.Name {
		t.Errorf("Name mismatch: got %q, want %q", unmarshaled.Name, project.Name)
	}
}

func TestWorktreeJSONMarshaling(t *testing.T) {
	now := time.Now()
	worktree := &Worktree{
		ID:        1,
		ProjectID: 1,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/user/repo/main",
		IsMain:    true,
		CreatedAt: now,
		LastUsed:  now,
	}

	// Test marshaling
	data, err := json.Marshal(worktree)
	if err != nil {
		t.Fatalf("Failed to marshal worktree: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Worktree
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal worktree: %v", err)
	}

	if unmarshaled.ID != worktree.ID {
		t.Errorf("ID mismatch: got %d, want %d", unmarshaled.ID, worktree.ID)
	}
	if unmarshaled.Branch != worktree.Branch {
		t.Errorf("Branch mismatch: got %q, want %q", unmarshaled.Branch, worktree.Branch)
	}
	if unmarshaled.IsMain != worktree.IsMain {
		t.Errorf("IsMain mismatch: got %v, want %v", unmarshaled.IsMain, worktree.IsMain)
	}
}

func TestSessionJSONMarshaling(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:              1,
		WorktreeID:      1,
		TmuxSessionName: "repo:main",
		CreatedAt:       now,
		LastAttached:    now,
	}

	// Test marshaling
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Session
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}

	if unmarshaled.ID != session.ID {
		t.Errorf("ID mismatch: got %d, want %d", unmarshaled.ID, session.ID)
	}
	if unmarshaled.TmuxSessionName != session.TmuxSessionName {
		t.Errorf("TmuxSessionName mismatch: got %q, want %q", unmarshaled.TmuxSessionName, session.TmuxSessionName)
	}
}

func TestSessionDetails(t *testing.T) {
	now := time.Now()
	details := &SessionDetails{
		Session: &Session{
			ID:              1,
			WorktreeID:      1,
			TmuxSessionName: "repo:main",
			CreatedAt:       now,
			LastAttached:    now,
		},
		Worktree: &Worktree{
			ID:        1,
			ProjectID: 1,
			Branch:    "main",
			Path:      "/home/user/.sesh/github.com/user/repo/main",
			IsMain:    true,
			CreatedAt: now,
			LastUsed:  now,
		},
		Project: &Project{
			ID:          1,
			Name:        "github.com/user/repo",
			RemoteURL:   "https://github.com/user/repo.git",
			LocalPath:   "/home/user/.sesh/github.com/user/repo.git",
			CreatedAt:   now,
			LastFetched: &now,
		},
	}

	if details.Session == nil {
		t.Error("Session is nil")
	}
	if details.Worktree == nil {
		t.Error("Worktree is nil")
	}
	if details.Project == nil {
		t.Error("Project is nil")
	}

	if details.Session.WorktreeID != details.Worktree.ID {
		t.Error("Session.WorktreeID doesn't match Worktree.ID")
	}
	if details.Worktree.ProjectID != details.Project.ID {
		t.Error("Worktree.ProjectID doesn't match Project.ID")
	}
}
