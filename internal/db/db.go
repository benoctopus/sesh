package db

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"

	"github.com/benoctopus/sesh/internal/models"
	"github.com/rotisserie/eris"
)

// InitDB initializes a new database connection
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to open database: %s", dbPath)
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, eris.Wrap(err, "failed to enable foreign keys")
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, eris.Wrap(err, "failed to ping database")
	}

	return db, nil
}

// ==================== Project CRUD Operations ====================

// CreateProject creates a new project in the database
func CreateProject(db *sql.DB, project *models.Project) error {
	result, err := db.Exec(
		"INSERT INTO projects (name, remote_url, local_path, created_at) VALUES (?, ?, ?, ?)",
		project.Name, project.RemoteURL, project.LocalPath, time.Now(),
	)
	if err != nil {
		return eris.Wrap(err, "failed to insert project")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return eris.Wrap(err, "failed to get last insert id")
	}

	project.ID = int(id)
	return nil
}

// GetProject retrieves a project by name
func GetProject(db *sql.DB, name string) (*models.Project, error) {
	project := &models.Project{}
	var lastFetched sql.NullTime

	err := db.QueryRow(
		"SELECT id, name, remote_url, local_path, created_at, last_fetched FROM projects WHERE name = ?",
		name,
	).Scan(&project.ID, &project.Name, &project.RemoteURL, &project.LocalPath, &project.CreatedAt, &lastFetched)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "project not found: %s", name)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query project")
	}

	if lastFetched.Valid {
		project.LastFetched = &lastFetched.Time
	}

	return project, nil
}

// GetProjectByID retrieves a project by ID
func GetProjectByID(db *sql.DB, id int) (*models.Project, error) {
	project := &models.Project{}
	var lastFetched sql.NullTime

	err := db.QueryRow(
		"SELECT id, name, remote_url, local_path, created_at, last_fetched FROM projects WHERE id = ?",
		id,
	).Scan(&project.ID, &project.Name, &project.RemoteURL, &project.LocalPath, &project.CreatedAt, &lastFetched)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "project not found with id: %d", id)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query project by id")
	}

	if lastFetched.Valid {
		project.LastFetched = &lastFetched.Time
	}

	return project, nil
}

// GetProjectByRemote retrieves a project by remote URL
func GetProjectByRemote(db *sql.DB, remoteURL string) (*models.Project, error) {
	project := &models.Project{}
	var lastFetched sql.NullTime

	err := db.QueryRow(
		"SELECT id, name, remote_url, local_path, created_at, last_fetched FROM projects WHERE remote_url = ?",
		remoteURL,
	).Scan(&project.ID, &project.Name, &project.RemoteURL, &project.LocalPath, &project.CreatedAt, &lastFetched)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "project not found with remote: %s", remoteURL)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query project by remote")
	}

	if lastFetched.Valid {
		project.LastFetched = &lastFetched.Time
	}

	return project, nil
}

// GetAllProjects retrieves all projects
func GetAllProjects(db *sql.DB) ([]*models.Project, error) {
	rows, err := db.Query(
		"SELECT id, name, remote_url, local_path, created_at, last_fetched FROM projects ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, eris.Wrap(err, "failed to query all projects")
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		project := &models.Project{}
		var lastFetched sql.NullTime

		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.RemoteURL,
			&project.LocalPath,
			&project.CreatedAt,
			&lastFetched,
		)
		if err != nil {
			return nil, eris.Wrap(err, "failed to scan project row")
		}

		if lastFetched.Valid {
			project.LastFetched = &lastFetched.Time
		}

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, eris.Wrap(err, "error iterating project rows")
	}

	return projects, nil
}

// UpdateProjectFetchTime updates the last_fetched timestamp for a project
func UpdateProjectFetchTime(db *sql.DB, id int) error {
	_, err := db.Exec(
		"UPDATE projects SET last_fetched = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return eris.Wrapf(err, "failed to update project fetch time for id: %d", id)
	}
	return nil
}

// DeleteProject deletes a project and all associated worktrees and sessions
func DeleteProject(db *sql.DB, id int) error {
	// Foreign key constraints will cascade delete worktrees and sessions
	result, err := db.Exec("DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return eris.Wrapf(err, "failed to delete project with id: %d", id)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return eris.Errorf("project not found with id: %d", id)
	}

	return nil
}

// ==================== Worktree CRUD Operations ====================

// CreateWorktree creates a new worktree in the database
func CreateWorktree(db *sql.DB, worktree *models.Worktree) error {
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO worktrees (project_id, branch, path, is_main, created_at, last_used) VALUES (?, ?, ?, ?, ?, ?)",
		worktree.ProjectID, worktree.Branch, worktree.Path, worktree.IsMain, now, now,
	)
	if err != nil {
		return eris.Wrap(err, "failed to insert worktree")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return eris.Wrap(err, "failed to get last insert id")
	}

	worktree.ID = int(id)
	worktree.CreatedAt = now
	worktree.LastUsed = now
	return nil
}

// GetWorktree retrieves a worktree by project ID and branch
func GetWorktree(db *sql.DB, projectID int, branch string) (*models.Worktree, error) {
	worktree := &models.Worktree{}
	err := db.QueryRow(
		"SELECT id, project_id, branch, path, is_main, created_at, last_used FROM worktrees WHERE project_id = ? AND branch = ?",
		projectID,
		branch,
	).Scan(&worktree.ID, &worktree.ProjectID, &worktree.Branch, &worktree.Path, &worktree.IsMain, &worktree.CreatedAt, &worktree.LastUsed)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "worktree not found for project %d, branch %s", projectID, branch)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query worktree")
	}

	return worktree, nil
}

// GetWorktreeByID retrieves a worktree by ID
func GetWorktreeByID(db *sql.DB, id int) (*models.Worktree, error) {
	worktree := &models.Worktree{}
	err := db.QueryRow(
		"SELECT id, project_id, branch, path, is_main, created_at, last_used FROM worktrees WHERE id = ?",
		id,
	).Scan(&worktree.ID, &worktree.ProjectID, &worktree.Branch, &worktree.Path, &worktree.IsMain, &worktree.CreatedAt, &worktree.LastUsed)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "worktree not found with id: %d", id)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query worktree by id")
	}

	return worktree, nil
}

// GetWorktreeByPath retrieves a worktree by path
func GetWorktreeByPath(db *sql.DB, path string) (*models.Worktree, error) {
	worktree := &models.Worktree{}
	err := db.QueryRow(
		"SELECT id, project_id, branch, path, is_main, created_at, last_used FROM worktrees WHERE path = ?",
		path,
	).Scan(&worktree.ID, &worktree.ProjectID, &worktree.Branch, &worktree.Path, &worktree.IsMain, &worktree.CreatedAt, &worktree.LastUsed)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "worktree not found with path: %s", path)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query worktree by path")
	}

	return worktree, nil
}

// GetWorktreesByProject retrieves all worktrees for a project
func GetWorktreesByProject(db *sql.DB, projectID int) ([]*models.Worktree, error) {
	rows, err := db.Query(
		"SELECT id, project_id, branch, path, is_main, created_at, last_used FROM worktrees WHERE project_id = ? ORDER BY last_used DESC",
		projectID,
	)
	if err != nil {
		return nil, eris.Wrap(err, "failed to query worktrees by project")
	}
	defer rows.Close()

	var worktrees []*models.Worktree
	for rows.Next() {
		worktree := &models.Worktree{}
		err := rows.Scan(
			&worktree.ID,
			&worktree.ProjectID,
			&worktree.Branch,
			&worktree.Path,
			&worktree.IsMain,
			&worktree.CreatedAt,
			&worktree.LastUsed,
		)
		if err != nil {
			return nil, eris.Wrap(err, "failed to scan worktree row")
		}
		worktrees = append(worktrees, worktree)
	}

	if err := rows.Err(); err != nil {
		return nil, eris.Wrap(err, "error iterating worktree rows")
	}

	return worktrees, nil
}

// UpdateWorktreeLastUsed updates the last_used timestamp for a worktree
func UpdateWorktreeLastUsed(db *sql.DB, id int) error {
	_, err := db.Exec(
		"UPDATE worktrees SET last_used = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return eris.Wrapf(err, "failed to update worktree last_used for id: %d", id)
	}
	return nil
}

// DeleteWorktree deletes a worktree and its associated session
func DeleteWorktree(db *sql.DB, id int) error {
	// Foreign key constraints will cascade delete sessions
	result, err := db.Exec("DELETE FROM worktrees WHERE id = ?", id)
	if err != nil {
		return eris.Wrapf(err, "failed to delete worktree with id: %d", id)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return eris.Errorf("worktree not found with id: %d", id)
	}

	return nil
}

// ==================== Session CRUD Operations ====================

// CreateSession creates a new session in the database
func CreateSession(db *sql.DB, session *models.Session) error {
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO sessions (worktree_id, tmux_session_name, created_at, last_attached) VALUES (?, ?, ?, ?)",
		session.WorktreeID, session.TmuxSessionName, now, now,
	)
	if err != nil {
		return eris.Wrap(err, "failed to insert session")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return eris.Wrap(err, "failed to get last insert id")
	}

	session.ID = int(id)
	session.CreatedAt = now
	session.LastAttached = now
	return nil
}

// GetSessionByWorktree retrieves a session by worktree ID
func GetSessionByWorktree(db *sql.DB, worktreeID int) (*models.Session, error) {
	session := &models.Session{}
	err := db.QueryRow(
		"SELECT id, worktree_id, tmux_session_name, created_at, last_attached FROM sessions WHERE worktree_id = ?",
		worktreeID,
	).Scan(&session.ID, &session.WorktreeID, &session.TmuxSessionName, &session.CreatedAt, &session.LastAttached)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "session not found for worktree: %d", worktreeID)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query session by worktree")
	}

	return session, nil
}

// GetSessionByTmuxName retrieves a session by tmux session name
func GetSessionByTmuxName(db *sql.DB, tmuxName string) (*models.Session, error) {
	session := &models.Session{}
	err := db.QueryRow(
		"SELECT id, worktree_id, tmux_session_name, created_at, last_attached FROM sessions WHERE tmux_session_name = ?",
		tmuxName,
	).Scan(&session.ID, &session.WorktreeID, &session.TmuxSessionName, &session.CreatedAt, &session.LastAttached)

	if err == sql.ErrNoRows {
		return nil, eris.Wrapf(err, "session not found with name: %s", tmuxName)
	}
	if err != nil {
		return nil, eris.Wrap(err, "failed to query session by tmux name")
	}

	return session, nil
}

// GetAllSessions retrieves all sessions
func GetAllSessions(db *sql.DB) ([]*models.Session, error) {
	rows, err := db.Query(
		"SELECT id, worktree_id, tmux_session_name, created_at, last_attached FROM sessions ORDER BY last_attached DESC",
	)
	if err != nil {
		return nil, eris.Wrap(err, "failed to query all sessions")
	}
	defer rows.Close()

	var sessions []*models.Session
	for rows.Next() {
		session := &models.Session{}
		err := rows.Scan(
			&session.ID,
			&session.WorktreeID,
			&session.TmuxSessionName,
			&session.CreatedAt,
			&session.LastAttached,
		)
		if err != nil {
			return nil, eris.Wrap(err, "failed to scan session row")
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, eris.Wrap(err, "error iterating session rows")
	}

	return sessions, nil
}

// GetAllSessionDetails retrieves all sessions with their worktree and project information
func GetAllSessionDetails(db *sql.DB) ([]*models.SessionDetails, error) {
	query := `
		SELECT
			s.id, s.worktree_id, s.tmux_session_name, s.created_at, s.last_attached,
			w.id, w.project_id, w.branch, w.path, w.is_main, w.created_at, w.last_used,
			p.id, p.name, p.remote_url, p.local_path, p.created_at, p.last_fetched
		FROM sessions s
		INNER JOIN worktrees w ON s.worktree_id = w.id
		INNER JOIN projects p ON w.project_id = p.id
		ORDER BY s.last_attached DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, eris.Wrap(err, "failed to query session details")
	}
	defer rows.Close()

	var details []*models.SessionDetails
	for rows.Next() {
		session := &models.Session{}
		worktree := &models.Worktree{}
		project := &models.Project{}
		var lastFetched sql.NullTime

		err := rows.Scan(
			&session.ID,
			&session.WorktreeID,
			&session.TmuxSessionName,
			&session.CreatedAt,
			&session.LastAttached,
			&worktree.ID,
			&worktree.ProjectID,
			&worktree.Branch,
			&worktree.Path,
			&worktree.IsMain,
			&worktree.CreatedAt,
			&worktree.LastUsed,
			&project.ID,
			&project.Name,
			&project.RemoteURL,
			&project.LocalPath,
			&project.CreatedAt,
			&lastFetched,
		)
		if err != nil {
			return nil, eris.Wrap(err, "failed to scan session details row")
		}

		if lastFetched.Valid {
			project.LastFetched = &lastFetched.Time
		}

		details = append(details, &models.SessionDetails{
			Session:  session,
			Worktree: worktree,
			Project:  project,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, eris.Wrap(err, "error iterating session details rows")
	}

	return details, nil
}

// UpdateSessionLastAttached updates the last_attached timestamp for a session
func UpdateSessionLastAttached(db *sql.DB, id int) error {
	_, err := db.Exec(
		"UPDATE sessions SET last_attached = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return eris.Wrapf(err, "failed to update session last_attached for id: %d", id)
	}
	return nil
}

// DeleteSession deletes a session
func DeleteSession(db *sql.DB, id int) error {
	result, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return eris.Wrapf(err, "failed to delete session with id: %d", id)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return eris.Errorf("session not found with id: %d", id)
	}

	return nil
}
