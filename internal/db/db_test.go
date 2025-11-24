package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/benoctopus/sesh/internal/models"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Create a temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return db
}

func TestInitDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB() failed: %v", err)
	}
	defer db.Close()

	// Verify database is initialized
	if err := db.Ping(); err != nil {
		t.Errorf("Database ping failed: %v", err)
	}

	// Verify foreign keys are enabled
	var foreignKeys int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Errorf("Failed to query foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("Foreign keys not enabled: got %d, want 1", foreignKeys)
	}

	// Verify migrations were run (check if tables exist)
	tables := []string{"projects", "worktrees", "sessions", "schema_migrations"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

func TestInitDB_InvalidPath(t *testing.T) {
	// Use an invalid path that should fail
	invalidPath := "/nonexistent/directory/test.db"

	db, err := InitDB(invalidPath)
	if err == nil {
		db.Close()
		t.Error("InitDB() should fail with invalid path")
	}
}

// ==================== Project CRUD Tests ====================

func TestCreateProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}

	err := CreateProject(db, project)
	if err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	if project.ID == 0 {
		t.Error("CreateProject() did not set project ID")
	}

	// Verify project was inserted
	retrieved, err := GetProject(db, project.Name)
	if err != nil {
		t.Fatalf("GetProject() failed: %v", err)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, project.Name)
	}
	if retrieved.RemoteURL != project.RemoteURL {
		t.Errorf("RemoteURL mismatch: got %q, want %q", retrieved.RemoteURL, project.RemoteURL)
	}
	if retrieved.LocalPath != project.LocalPath {
		t.Errorf("LocalPath mismatch: got %q, want %q", retrieved.LocalPath, project.LocalPath)
	}
}

func TestCreateProject_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}

	// First insert should succeed
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("First CreateProject() failed: %v", err)
	}

	// Second insert with same name should fail
	duplicate := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:different/repo.git",
		LocalPath: "/different/path",
	}

	err := CreateProject(db, duplicate)
	if err == nil {
		t.Error("CreateProject() should fail for duplicate name")
	}
}

func TestGetProject_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetProject(db, "nonexistent")
	if err == nil {
		t.Error("GetProject() should return error for nonexistent project")
	}
}

func TestGetProjectByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}

	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	retrieved, err := GetProjectByID(db, project.ID)
	if err != nil {
		t.Fatalf("GetProjectByID() failed: %v", err)
	}

	if retrieved.ID != project.ID {
		t.Errorf("ID mismatch: got %d, want %d", retrieved.ID, project.ID)
	}
}

func TestGetProjectByRemote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}

	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	retrieved, err := GetProjectByRemote(db, project.RemoteURL)
	if err != nil {
		t.Fatalf("GetProjectByRemote() failed: %v", err)
	}

	if retrieved.Name != project.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, project.Name)
	}
}

func TestGetAllProjects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create multiple projects
	projects := []*models.Project{
		{
			Name:      "github.com/test/repo1",
			RemoteURL: "git@github.com:test/repo1.git",
			LocalPath: "/home/user/.sesh/github.com/test/repo1/.git",
		},
		{
			Name:      "github.com/test/repo2",
			RemoteURL: "git@github.com:test/repo2.git",
			LocalPath: "/home/user/.sesh/github.com/test/repo2/.git",
		},
	}

	for _, p := range projects {
		if err := CreateProject(db, p); err != nil {
			t.Fatalf("CreateProject() failed: %v", err)
		}
	}

	retrieved, err := GetAllProjects(db)
	if err != nil {
		t.Fatalf("GetAllProjects() failed: %v", err)
	}

	if len(retrieved) != len(projects) {
		t.Errorf("GetAllProjects() returned %d projects, want %d", len(retrieved), len(projects))
	}
}

func TestUpdateProjectFetchTime(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}

	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	// Initially, LastFetched should be nil
	retrieved, err := GetProject(db, project.Name)
	if err != nil {
		t.Fatalf("GetProject() failed: %v", err)
	}
	if retrieved.LastFetched != nil {
		t.Error("LastFetched should initially be nil")
	}

	// Update fetch time
	time.Sleep(10 * time.Millisecond) // Small delay to ensure timestamp difference
	if err := UpdateProjectFetchTime(db, project.ID); err != nil {
		t.Fatalf("UpdateProjectFetchTime() failed: %v", err)
	}

	// Verify LastFetched was updated
	retrieved, err = GetProject(db, project.Name)
	if err != nil {
		t.Fatalf("GetProject() failed: %v", err)
	}
	if retrieved.LastFetched == nil {
		t.Error("LastFetched should be set after update")
	}
}

func TestDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}

	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	// Delete the project
	if err := DeleteProject(db, project.ID); err != nil {
		t.Fatalf("DeleteProject() failed: %v", err)
	}

	// Verify project was deleted
	_, err := GetProject(db, project.Name)
	if err == nil {
		t.Error("GetProject() should fail for deleted project")
	}
}

// ==================== Worktree CRUD Tests ====================

func TestCreateWorktree(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a project first
	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}

	err := CreateWorktree(db, worktree)
	if err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	if worktree.ID == 0 {
		t.Error("CreateWorktree() did not set worktree ID")
	}

	// Verify worktree was inserted
	retrieved, err := GetWorktree(db, project.ID, "main")
	if err != nil {
		t.Fatalf("GetWorktree() failed: %v", err)
	}

	if retrieved.Branch != worktree.Branch {
		t.Errorf("Branch mismatch: got %q, want %q", retrieved.Branch, worktree.Branch)
	}
	if retrieved.Path != worktree.Path {
		t.Errorf("Path mismatch: got %q, want %q", retrieved.Path, worktree.Path)
	}
	if retrieved.IsMain != worktree.IsMain {
		t.Errorf("IsMain mismatch: got %v, want %v", retrieved.IsMain, worktree.IsMain)
	}
}

func TestGetWorktreeByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	retrieved, err := GetWorktreeByPath(db, worktree.Path)
	if err != nil {
		t.Fatalf("GetWorktreeByPath() failed: %v", err)
	}

	if retrieved.ID != worktree.ID {
		t.Errorf("ID mismatch: got %d, want %d", retrieved.ID, worktree.ID)
	}
}

func TestGetWorktreesByProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	// Create multiple worktrees
	worktrees := []*models.Worktree{
		{
			ProjectID: project.ID,
			Branch:    "main",
			Path:      "/home/user/.sesh/github.com/test/repo/main",
			IsMain:    true,
		},
		{
			ProjectID: project.ID,
			Branch:    "feature",
			Path:      "/home/user/.sesh/github.com/test/repo/feature",
			IsMain:    false,
		},
	}

	for _, w := range worktrees {
		if err := CreateWorktree(db, w); err != nil {
			t.Fatalf("CreateWorktree() failed: %v", err)
		}
	}

	retrieved, err := GetWorktreesByProject(db, project.ID)
	if err != nil {
		t.Fatalf("GetWorktreesByProject() failed: %v", err)
	}

	if len(retrieved) != len(worktrees) {
		t.Errorf("GetWorktreesByProject() returned %d worktrees, want %d", len(retrieved), len(worktrees))
	}
}

func TestUpdateWorktreeLastUsed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	originalLastUsed := worktree.LastUsed

	// Update last used time
	time.Sleep(10 * time.Millisecond)
	if err := UpdateWorktreeLastUsed(db, worktree.ID); err != nil {
		t.Fatalf("UpdateWorktreeLastUsed() failed: %v", err)
	}

	// Verify LastUsed was updated
	retrieved, err := GetWorktree(db, project.ID, "main")
	if err != nil {
		t.Fatalf("GetWorktree() failed: %v", err)
	}

	if !retrieved.LastUsed.After(originalLastUsed) {
		t.Error("LastUsed should be updated to a later time")
	}
}

func TestDeleteWorktree(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	// Delete the worktree
	if err := DeleteWorktree(db, worktree.ID); err != nil {
		t.Fatalf("DeleteWorktree() failed: %v", err)
	}

	// Verify worktree was deleted
	_, err := GetWorktree(db, project.ID, "main")
	if err == nil {
		t.Error("GetWorktree() should fail for deleted worktree")
	}
}

// ==================== Session CRUD Tests ====================

func TestCreateSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create project and worktree first
	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	session := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: "repo:main",
	}

	err := CreateSession(db, session)
	if err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	if session.ID == 0 {
		t.Error("CreateSession() did not set session ID")
	}

	// Verify session was inserted
	retrieved, err := GetSessionByWorktree(db, worktree.ID)
	if err != nil {
		t.Fatalf("GetSessionByWorktree() failed: %v", err)
	}

	if retrieved.TmuxSessionName != session.TmuxSessionName {
		t.Errorf("TmuxSessionName mismatch: got %q, want %q", retrieved.TmuxSessionName, session.TmuxSessionName)
	}
}

func TestGetSessionByTmuxName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	session := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: "repo:main",
	}
	if err := CreateSession(db, session); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	retrieved, err := GetSessionByTmuxName(db, "repo:main")
	if err != nil {
		t.Fatalf("GetSessionByTmuxName() failed: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("ID mismatch: got %d, want %d", retrieved.ID, session.ID)
	}
}

func TestGetAllSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	// Create multiple worktrees and sessions
	branches := []string{"main", "feature"}
	for _, branch := range branches {
		worktree := &models.Worktree{
			ProjectID: project.ID,
			Branch:    branch,
			Path:      "/home/user/.sesh/github.com/test/repo/" + branch,
			IsMain:    branch == "main",
		}
		if err := CreateWorktree(db, worktree); err != nil {
			t.Fatalf("CreateWorktree() failed: %v", err)
		}

		session := &models.Session{
			WorktreeID:      worktree.ID,
			TmuxSessionName: "repo:" + branch,
		}
		if err := CreateSession(db, session); err != nil {
			t.Fatalf("CreateSession() failed: %v", err)
		}
	}

	sessions, err := GetAllSessions(db)
	if err != nil {
		t.Fatalf("GetAllSessions() failed: %v", err)
	}

	if len(sessions) != len(branches) {
		t.Errorf("GetAllSessions() returned %d sessions, want %d", len(sessions), len(branches))
	}
}

func TestGetAllSessionDetails(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	session := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: "repo:main",
	}
	if err := CreateSession(db, session); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	details, err := GetAllSessionDetails(db)
	if err != nil {
		t.Fatalf("GetAllSessionDetails() failed: %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("GetAllSessionDetails() returned %d details, want 1", len(details))
	}

	detail := details[0]
	if detail.Session == nil || detail.Worktree == nil || detail.Project == nil {
		t.Error("GetAllSessionDetails() should populate all fields")
	}

	if detail.Project.Name != project.Name {
		t.Errorf("Project name mismatch: got %q, want %q", detail.Project.Name, project.Name)
	}
	if detail.Worktree.Branch != worktree.Branch {
		t.Errorf("Worktree branch mismatch: got %q, want %q", detail.Worktree.Branch, worktree.Branch)
	}
	if detail.Session.TmuxSessionName != session.TmuxSessionName {
		t.Errorf("Session name mismatch: got %q, want %q", detail.Session.TmuxSessionName, session.TmuxSessionName)
	}
}

func TestUpdateSessionLastAttached(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	session := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: "repo:main",
	}
	if err := CreateSession(db, session); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	originalLastAttached := session.LastAttached

	// Update last attached time
	time.Sleep(10 * time.Millisecond)
	if err := UpdateSessionLastAttached(db, session.ID); err != nil {
		t.Fatalf("UpdateSessionLastAttached() failed: %v", err)
	}

	// Verify LastAttached was updated
	retrieved, err := GetSessionByWorktree(db, worktree.ID)
	if err != nil {
		t.Fatalf("GetSessionByWorktree() failed: %v", err)
	}

	if !retrieved.LastAttached.After(originalLastAttached) {
		t.Error("LastAttached should be updated to a later time")
	}
}

func TestDeleteSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	session := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: "repo:main",
	}
	if err := CreateSession(db, session); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	// Delete the session
	if err := DeleteSession(db, session.ID); err != nil {
		t.Fatalf("DeleteSession() failed: %v", err)
	}

	// Verify session was deleted
	_, err := GetSessionByWorktree(db, worktree.ID)
	if err == nil {
		t.Error("GetSessionByWorktree() should fail for deleted session")
	}
}

// ==================== Foreign Key Cascade Tests ====================

func TestDeleteProject_CascadesWorktreesAndSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create project, worktree, and session
	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	session := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: "repo:main",
	}
	if err := CreateSession(db, session); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	// Delete the project
	if err := DeleteProject(db, project.ID); err != nil {
		t.Fatalf("DeleteProject() failed: %v", err)
	}

	// Verify worktree was cascade deleted
	_, err := GetWorktree(db, project.ID, "main")
	if err == nil {
		t.Error("Worktree should be cascade deleted when project is deleted")
	}

	// Verify session was cascade deleted
	_, err = GetSessionByWorktree(db, worktree.ID)
	if err == nil {
		t.Error("Session should be cascade deleted when project is deleted")
	}
}

func TestDeleteWorktree_CascadesSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create project, worktree, and session
	project := &models.Project{
		Name:      "github.com/test/repo",
		RemoteURL: "git@github.com:test/repo.git",
		LocalPath: "/home/user/.sesh/github.com/test/repo/.git",
	}
	if err := CreateProject(db, project); err != nil {
		t.Fatalf("CreateProject() failed: %v", err)
	}

	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    "main",
		Path:      "/home/user/.sesh/github.com/test/repo/main",
		IsMain:    true,
	}
	if err := CreateWorktree(db, worktree); err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}

	session := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: "repo:main",
	}
	if err := CreateSession(db, session); err != nil {
		t.Fatalf("CreateSession() failed: %v", err)
	}

	// Delete the worktree
	if err := DeleteWorktree(db, worktree.ID); err != nil {
		t.Fatalf("DeleteWorktree() failed: %v", err)
	}

	// Verify session was cascade deleted
	_, err := GetSessionByWorktree(db, worktree.ID)
	if err == nil {
		t.Error("Session should be cascade deleted when worktree is deleted")
	}

	// Verify project still exists
	_, err = GetProject(db, project.Name)
	if err != nil {
		t.Error("Project should still exist after worktree deletion")
	}
}
