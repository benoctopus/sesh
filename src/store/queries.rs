use super::models::*;
use crate::error::{Error, Result};
use sqlx::sqlite::SqlitePool;
use std::path::PathBuf;

impl Project {
    pub fn from_row(row: &sqlx::sqlite::SqliteRow) -> Result<Self> {
        use sqlx::Row;
        Ok(Self {
            id: row.get::<i64, _>("id"),
            name: row.get::<String, _>("name"),
            display_name: row.get::<String, _>("display_name"),
            remote_url: row.get::<String, _>("remote_url"),
            clone_path: PathBuf::from(row.get::<String, _>("clone_path")),
            default_branch: row.get::<String, _>("default_branch"),
            created_at: row.get::<String, _>("created_at"),
            last_fetched_at: row.get::<Option<String>, _>("last_fetched_at"),
        })
    }
}

impl Worktree {
    pub fn from_row(row: &sqlx::sqlite::SqliteRow) -> Result<Self> {
        use sqlx::Row;
        Ok(Self {
            id: row.get::<i64, _>("id"),
            project_id: row.get::<i64, _>("project_id"),
            branch: row.get::<String, _>("branch"),
            path: PathBuf::from(row.get::<String, _>("path")),
            is_primary: row.get::<i64, _>("is_primary") != 0,
            created_at: row.get::<String, _>("created_at"),
            last_accessed_at: row.get::<String, _>("last_accessed_at"),
        })
    }
}

impl Session {
    pub fn from_row(row: &sqlx::sqlite::SqliteRow) -> Result<Self> {
        use sqlx::Row;
        Ok(Self {
            id: row.get::<i64, _>("id"),
            worktree_id: row.get::<i64, _>("worktree_id"),
            session_name: row.get::<String, _>("session_name"),
            backend: row.get::<String, _>("backend"),
            created_at: row.get::<String, _>("created_at"),
            last_attached_at: row.get::<String, _>("last_attached_at"),
        })
    }
}

pub async fn create_project(pool: &SqlitePool, project: CreateProject) -> Result<Project> {
    let id = sqlx::query(
        r#"
        INSERT INTO projects (name, display_name, remote_url, clone_path, default_branch)
        VALUES (?1, ?2, ?3, ?4, ?5)
        "#,
    )
    .bind(&project.name)
    .bind(&project.display_name)
    .bind(&project.remote_url)
    .bind(project.clone_path.to_string_lossy().to_string())
    .bind(&project.default_branch)
    .execute(pool)
    .await?
    .last_insert_rowid();

    get_project(pool, id).await
}

pub async fn get_project(pool: &SqlitePool, id: i64) -> Result<Project> {
    let row = sqlx::query("SELECT * FROM projects WHERE id = ?1")
        .bind(id)
        .fetch_one(pool)
        .await?;

    Project::from_row(&row)
}

pub async fn get_project_by_name(pool: &SqlitePool, name: &str) -> Result<Project> {
    let row = sqlx::query("SELECT * FROM projects WHERE name = ?1")
        .bind(name)
        .fetch_one(pool)
        .await?;

    Project::from_row(&row)
}

/// Find project by name with flexible matching (exact name, display_name, or partial match)
pub async fn find_project_by_name(pool: &SqlitePool, search: &str) -> Result<Project> {
    // Try exact match on name first
    if let Ok(project) = get_project_by_name(pool, search).await {
        return Ok(project);
    }
    
    // Try exact match on display_name or partial match on name (ends with search term)
    let rows = sqlx::query(
        "SELECT * FROM projects WHERE display_name = ?1 OR name LIKE ?2"
    )
    .bind(search)
    .bind(format!("%/{}", search))
    .fetch_all(pool)
    .await?;
    
    match rows.len() {
        0 => Err(Error::ProjectNotFound { name: search.to_string() }),
        1 => Project::from_row(&rows[0]),
        _ => {
            // Multiple matches - collect project names
            let matches: Vec<String> = rows.iter()
                .filter_map(|r| Project::from_row(r).ok())
                .map(|p| p.name)
                .collect();
            Err(Error::AmbiguousProjectName { 
                name: search.to_string(),
                matches,
            })
        }
    }
}

pub async fn list_projects(pool: &SqlitePool) -> Result<Vec<Project>> {
    let rows = sqlx::query("SELECT * FROM projects ORDER BY name")
        .fetch_all(pool)
        .await?;

    let mut projects = Vec::new();
    for row in rows {
        projects.push(Project::from_row(&row)?);
    }
    Ok(projects)
}

pub async fn delete_project(pool: &SqlitePool, id: i64) -> Result<()> {
    sqlx::query("DELETE FROM projects WHERE id = ?1")
        .bind(id)
        .execute(pool)
        .await?;
    Ok(())
}

pub async fn create_worktree(pool: &SqlitePool, worktree: CreateWorktree) -> Result<Worktree> {
    let id = sqlx::query(
        r#"
        INSERT INTO worktrees (project_id, branch, path, is_primary)
        VALUES (?1, ?2, ?3, ?4)
        "#,
    )
    .bind(worktree.project_id)
    .bind(&worktree.branch)
    .bind(worktree.path.to_string_lossy().to_string())
    .bind(if worktree.is_primary { 1 } else { 0 })
    .execute(pool)
    .await?
    .last_insert_rowid();

    get_worktree(pool, id).await
}

pub async fn get_worktree(pool: &SqlitePool, id: i64) -> Result<Worktree> {
    let row = sqlx::query("SELECT * FROM worktrees WHERE id = ?1")
        .bind(id)
        .fetch_one(pool)
        .await?;

    Worktree::from_row(&row)
}

pub async fn get_worktree_by_project_branch(
    pool: &SqlitePool,
    project_id: i64,
    branch: &str,
) -> Result<Option<Worktree>> {
    let row = sqlx::query("SELECT * FROM worktrees WHERE project_id = ?1 AND branch = ?2")
        .bind(project_id)
        .bind(branch)
        .fetch_optional(pool)
        .await?;

    match row {
        Some(r) => Ok(Some(Worktree::from_row(&r)?)),
        None => Ok(None),
    }
}

pub async fn list_worktrees(pool: &SqlitePool, project_id: i64) -> Result<Vec<Worktree>> {
    let rows = sqlx::query("SELECT * FROM worktrees WHERE project_id = ?1 ORDER BY branch")
        .bind(project_id)
        .fetch_all(pool)
        .await?;

    let mut worktrees = Vec::new();
    for row in rows {
        worktrees.push(Worktree::from_row(&row)?);
    }
    Ok(worktrees)
}

pub async fn touch_worktree(pool: &SqlitePool, id: i64) -> Result<()> {
    sqlx::query("UPDATE worktrees SET last_accessed_at = datetime('now') WHERE id = ?1")
        .bind(id)
        .execute(pool)
        .await?;
    Ok(())
}

pub async fn create_session(pool: &SqlitePool, session: CreateSession) -> Result<Session> {
    let id = sqlx::query(
        r#"
        INSERT INTO sessions (worktree_id, session_name, backend)
        VALUES (?1, ?2, ?3)
        "#,
    )
    .bind(session.worktree_id)
    .bind(&session.session_name)
    .bind(&session.backend)
    .execute(pool)
    .await?
    .last_insert_rowid();

    get_session(pool, id).await
}

pub async fn get_session(pool: &SqlitePool, id: i64) -> Result<Session> {
    let row = sqlx::query("SELECT * FROM sessions WHERE id = ?1")
        .bind(id)
        .fetch_one(pool)
        .await?;

    Session::from_row(&row)
}

pub async fn get_session_for_worktree(pool: &SqlitePool, worktree_id: i64) -> Result<Session> {
    let row = sqlx::query("SELECT * FROM sessions WHERE worktree_id = ?1")
        .bind(worktree_id)
        .fetch_one(pool)
        .await?;

    Session::from_row(&row)
}

pub async fn get_session_by_name(pool: &SqlitePool, name: &str) -> Result<Option<Session>> {
    let row = sqlx::query("SELECT * FROM sessions WHERE session_name = ?1")
        .bind(name)
        .fetch_optional(pool)
        .await?;

    match row {
        Some(r) => Ok(Some(Session::from_row(&r)?)),
        None => Ok(None),
    }
}

pub async fn touch_session(pool: &SqlitePool, id: i64) -> Result<()> {
    sqlx::query("UPDATE sessions SET last_attached_at = datetime('now') WHERE id = ?1")
        .bind(id)
        .execute(pool)
        .await?;
    Ok(())
}

pub async fn add_to_history(pool: &SqlitePool, session_id: i64) -> Result<()> {
    sqlx::query("INSERT INTO session_history (session_id) VALUES (?1)")
        .bind(session_id)
        .execute(pool)
        .await?;
    Ok(())
}

pub async fn get_previous_session(
    pool: &SqlitePool,
    current_session: Option<&str>,
) -> Result<Session> {
    // Get the most recent session from history, excluding current if provided
    let row = if let Some(current) = current_session {
        sqlx::query(
            r#"
            SELECT s.* FROM sessions s
            INNER JOIN session_history h ON s.id = h.session_id
            WHERE s.session_name != ?1
            ORDER BY h.accessed_at DESC
            LIMIT 1
            "#,
        )
        .bind(current)
        .fetch_one(pool)
        .await?
    } else {
        sqlx::query(
            r#"
            SELECT s.* FROM sessions s
            INNER JOIN session_history h ON s.id = h.session_id
            ORDER BY h.accessed_at DESC
            LIMIT 1
            "#,
        )
        .fetch_one(pool)
        .await?
    };

    Session::from_row(&row)
}
