use super::models::*;
use crate::error::{Error, Result};
use sqlx::sqlite::SqlitePool;
use std::path::PathBuf;

impl Project {
    pub async fn from_row(row: sqlx::sqlite::SqliteRow) -> Result<Self> {
        Ok(Self {
            id: row.get("id"),
            name: row.get("name"),
            display_name: row.get("display_name"),
            remote_url: row.get("remote_url"),
            clone_path: PathBuf::from(row.get::<String, _>("clone_path")),
            default_branch: row.get("default_branch"),
            created_at: row.get("created_at"),
            last_fetched_at: row.get("last_fetched_at"),
        })
    }
}

impl Worktree {
    pub async fn from_row(row: sqlx::sqlite::SqliteRow) -> Result<Self> {
        Ok(Self {
            id: row.get("id"),
            project_id: row.get("project_id"),
            branch: row.get("branch"),
            path: PathBuf::from(row.get::<String, _>("path")),
            is_primary: row.get::<i64, _>("is_primary") != 0,
            created_at: row.get("created_at"),
            last_accessed_at: row.get("last_accessed_at"),
        })
    }
}

impl Session {
    pub async fn from_row(row: sqlx::sqlite::SqliteRow) -> Result<Self> {
        Ok(Self {
            id: row.get("id"),
            worktree_id: row.get("worktree_id"),
            session_name: row.get("session_name"),
            backend: row.get("backend"),
            created_at: row.get("created_at"),
            last_attached_at: row.get("last_attached_at"),
        })
    }
}

pub async fn create_project(pool: &SqlitePool, project: CreateProject) -> Result<Project> {
    let id = sqlx::query!(
        r#"
        INSERT INTO projects (name, display_name, remote_url, clone_path, default_branch)
        VALUES (?1, ?2, ?3, ?4, ?5)
        "#,
        project.name,
        project.display_name,
        project.remote_url,
        project.clone_path.to_string_lossy(),
        project.default_branch
    )
    .execute(pool)
    .await?
    .last_insert_rowid();
    
    get_project(pool, id).await
}

pub async fn get_project(pool: &SqlitePool, id: i64) -> Result<Project> {
    let row = sqlx::query!(
        "SELECT * FROM projects WHERE id = ?1",
        id
    )
    .fetch_one(pool)
    .await?;
    
    Project::from_row(row.into()).await
}

pub async fn get_project_by_name(pool: &SqlitePool, name: &str) -> Result<Project> {
    let row = sqlx::query!(
        "SELECT * FROM projects WHERE name = ?1",
        name
    )
    .fetch_one(pool)
    .await?;
    
    Project::from_row(row.into()).await
}

pub async fn list_projects(pool: &SqlitePool) -> Result<Vec<Project>> {
    let rows = sqlx::query!("SELECT * FROM projects ORDER BY name")
        .fetch_all(pool)
        .await?;
    
    let mut projects = Vec::new();
    for row in rows {
        projects.push(Project::from_row(row.into()).await?);
    }
    Ok(projects)
}

pub async fn delete_project(pool: &SqlitePool, id: i64) -> Result<()> {
    sqlx::query!("DELETE FROM projects WHERE id = ?1", id)
        .execute(pool)
        .await?;
    Ok(())
}

pub async fn create_worktree(pool: &SqlitePool, worktree: CreateWorktree) -> Result<Worktree> {
    let id = sqlx::query!(
        r#"
        INSERT INTO worktrees (project_id, branch, path, is_primary)
        VALUES (?1, ?2, ?3, ?4)
        "#,
        worktree.project_id,
        worktree.branch,
        worktree.path.to_string_lossy(),
        if worktree.is_primary { 1 } else { 0 }
    )
    .execute(pool)
    .await?
    .last_insert_rowid();
    
    get_worktree(pool, id).await
}

pub async fn get_worktree(pool: &SqlitePool, id: i64) -> Result<Worktree> {
    let row = sqlx::query!(
        "SELECT * FROM worktrees WHERE id = ?1",
        id
    )
    .fetch_one(pool)
    .await?;
    
    Worktree::from_row(row.into()).await
}

pub async fn get_worktree_by_project_branch(
    pool: &SqlitePool,
    project_id: i64,
    branch: &str,
) -> Result<Option<Worktree>> {
    let row = sqlx::query!(
        "SELECT * FROM worktrees WHERE project_id = ?1 AND branch = ?2",
        project_id,
        branch
    )
    .fetch_optional(pool)
    .await?;
    
    match row {
        Some(r) => Ok(Some(Worktree::from_row(r.into()).await?)),
        None => Ok(None),
    }
}

pub async fn list_worktrees(pool: &SqlitePool, project_id: i64) -> Result<Vec<Worktree>> {
    let rows = sqlx::query!(
        "SELECT * FROM worktrees WHERE project_id = ?1 ORDER BY branch",
        project_id
    )
    .fetch_all(pool)
    .await?;
    
    let mut worktrees = Vec::new();
    for row in rows {
        worktrees.push(Worktree::from_row(row.into()).await?);
    }
    Ok(worktrees)
}

pub async fn touch_worktree(pool: &SqlitePool, id: i64) -> Result<()> {
    sqlx::query!(
        "UPDATE worktrees SET last_accessed_at = datetime('now') WHERE id = ?1",
        id
    )
    .execute(pool)
    .await?;
    Ok(())
}

pub async fn create_session(pool: &SqlitePool, session: CreateSession) -> Result<Session> {
    let id = sqlx::query!(
        r#"
        INSERT INTO sessions (worktree_id, session_name, backend)
        VALUES (?1, ?2, ?3)
        "#,
        session.worktree_id,
        session.session_name,
        session.backend
    )
    .execute(pool)
    .await?
    .last_insert_rowid();
    
    get_session(pool, id).await
}

pub async fn get_session(pool: &SqlitePool, id: i64) -> Result<Session> {
    let row = sqlx::query!(
        "SELECT * FROM sessions WHERE id = ?1",
        id
    )
    .fetch_one(pool)
    .await?;
    
    Session::from_row(row.into()).await
}

pub async fn get_session_for_worktree(
    pool: &SqlitePool,
    worktree_id: i64,
) -> Result<Session> {
    let row = sqlx::query!(
        "SELECT * FROM sessions WHERE worktree_id = ?1",
        worktree_id
    )
    .fetch_one(pool)
    .await?;
    
    Session::from_row(row.into()).await
}

pub async fn get_session_by_name(pool: &SqlitePool, name: &str) -> Result<Option<Session>> {
    let row = sqlx::query!(
        "SELECT * FROM sessions WHERE session_name = ?1",
        name
    )
    .fetch_optional(pool)
    .await?;
    
    match row {
        Some(r) => Ok(Some(Session::from_row(r.into()).await?)),
        None => Ok(None),
    }
}

pub async fn touch_session(pool: &SqlitePool, id: i64) -> Result<()> {
    sqlx::query!(
        "UPDATE sessions SET last_attached_at = datetime('now') WHERE id = ?1",
        id
    )
    .execute(pool)
    .await?;
    Ok(())
}

pub async fn add_to_history(pool: &SqlitePool, session_id: i64) -> Result<()> {
    sqlx::query!(
        "INSERT INTO session_history (session_id) VALUES (?1)",
        session_id
    )
    .execute(pool)
    .await?;
    Ok(())
}

pub async fn get_previous_session(
    pool: &SqlitePool,
    current_session: Option<&str>,
) -> Result<Session> {
    // Get the most recent session from history, excluding current if provided
    let query = if let Some(current) = current_session {
        sqlx::query!(
            r#"
            SELECT s.* FROM sessions s
            INNER JOIN session_history h ON s.id = h.session_id
            WHERE s.session_name != ?1
            ORDER BY h.accessed_at DESC
            LIMIT 1
            "#,
            current
        )
    } else {
        sqlx::query!(
            r#"
            SELECT s.* FROM sessions s
            INNER JOIN session_history h ON s.id = h.session_id
            ORDER BY h.accessed_at DESC
            LIMIT 1
            "#
        )
    };
    
    let row = query.fetch_one(pool).await?;
    Session::from_row(row.into()).await
}

