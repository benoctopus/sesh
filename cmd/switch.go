package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/fuzzy"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/pr"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/tty"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	switchProjectName    string
	switchStartupCommand string
	switchPR             bool
	switchDetach         bool
	switchSelectProject  bool
)

var switchCmd = &cobra.Command{
	Use:     "switch [-p project] [branch]",
	Aliases: []string{"sw"},
	Short:   "Switch to a branch or pull request (create worktree if needed)",
	Long: `Switch to a branch or pull request, creating a worktree and session if they don't exist.
If no branch is specified, an interactive fuzzy finder will show all available branches.
Use --pr to select from open pull requests instead.

The project is automatically detected from the current working directory,
or can be specified explicitly with the --project flag.

If the branch doesn't exist locally or remotely, a new branch will be created automatically.

If a git URL is provided for the --project flag and the repository has not been cloned yet,
it will be automatically cloned before switching to the branch.

Use --select-project to interactively select a project and then a session from that project.

Examples:
  sesh switch feature-foo                                    # Switch to existing branch
  sesh sw new-feature                                        # Create new branch automatically
  sesh switch                                                # Interactive fuzzy branch selection
  sesh switch --pr                                           # Interactive PR selection
  sesh switch --select-project                               # Interactive project and session selection
  sesh switch --project myproject feature-bar                # Explicit project
  sesh switch -p git@github.com:user/repo.git main           # Auto-clone and switch
  sesh switch -p https://github.com/user/repo.git feature    # Auto-clone HTTPS URL
  sesh switch -c "direnv allow" feature-baz                  # Run startup command
  sesh switch -d feature-test                                # Create session without attaching`,
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
	switchCmd.Flags().
		StringVarP(&switchProjectName, "project", "p", "", "Specify project explicitly")
	switchCmd.Flags().
		StringVarP(&switchStartupCommand, "command", "c", "", "Command to run after switching to session")
	switchCmd.Flags().
		BoolVar(&switchPR, "pr", false, "Select from open pull requests")
	switchCmd.Flags().
		BoolVarP(&switchDetach, "detach", "d", false, "Create session without attaching to it")
	switchCmd.Flags().
		BoolVar(&switchSelectProject, "select-project", false, "Interactively select project then session")
}

func runSwitch(cmd *cobra.Command, args []string) error {
	disp := display.NewStderr()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Handle special case: --select-project flag triggers interactive project and session selection
	if switchSelectProject {
		return runInteractiveProjectSessionSelection(cfg)
	}

	// Handle auto-clone if a git URL is provided
	if switchProjectName != "" && git.IsGitURL(switchProjectName) {
		remoteURL := switchProjectName

		// Generate project name from the URL
		projectName, err := git.GenerateProjectName(remoteURL)
		if err != nil {
			return eris.Wrap(err, "failed to generate project name from remote URL")
		}

		// Check if project already exists
		existingProject, err := state.GetProject(cfg.WorkspaceDir, projectName)
		if err != nil || existingProject == nil {
			// Project doesn't exist, clone it
			if err := cloneRepository(cfg, remoteURL, projectName); err != nil {
				return eris.Wrap(err, "failed to clone repository")
			}
		}

		// Update switchProjectName to use the generated project name
		switchProjectName = projectName
	}

	// Resolve project from filesystem state
	proj, err := project.ResolveProject(cfg.WorkspaceDir, switchProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	var branch string

	// Handle PR selection if --pr flag is set
	if switchPR {
		disp := display.NewStderr()

		if len(args) > 0 {
			return eris.New("cannot specify branch name with --pr flag")
		}

		// Get remote URL
		remoteURL, err := git.GetRemoteURL(proj.LocalPath)
		if err != nil {
			return eris.Wrap(err, "failed to get remote URL")
		}

		// Create PR provider
		provider, err := pr.NewProvider(remoteURL)
		if err != nil {
			return eris.Wrap(err, "failed to create PR provider")
		}

		// Check if gh CLI is installed and authenticated (for GitHub)
		if provider.Name() == "github" {
			if err := pr.CheckGHCLI(); err != nil {
				return err
			}
		}

		// List open PRs
		prs, err := provider.ListOpenPRs(cmd.Context(), proj.LocalPath)
		if err != nil {
			return eris.Wrap(err, "failed to list pull requests")
		}

		if len(prs) == 0 {
			return eris.New("no open pull requests found")
		}

		// Format PRs for fuzzy finder
		prChoices := make([]string, len(prs))
		for i, pullRequest := range prs {
			prChoices[i] = pr.FormatPRForFuzzyFinder(pullRequest)
		}

		// Create reader from PR choices for fuzzy finder
		prReader := io.NopCloser(strings.NewReader(strings.Join(prChoices, "\n")))

		// Get binary path for preview command
		var selectedPR string
		bin, err := os.Executable()
		if err != nil {
			// Fallback to simple selection without preview if we can't get binary path
			selectedPR, err = fuzzy.SelectBranchFromReader(prReader)
			if err != nil {
				return eris.Wrap(err, "failed to select pull request")
			}
		} else {
			// Use preview command with absolute binary path
			previewCmd := fmt.Sprintf("%s info --pr {}", bin)
			selectedPR, err = fuzzy.SelectBranchFromReaderWithPreview(prReader, previewCmd)
			if err != nil {
				return eris.Wrap(err, "failed to select pull request")
			}
		}

		// Parse PR number from selection
		prNum, err := pr.ParsePRNumber(selectedPR)
		if err != nil {
			return eris.Wrap(err, "failed to parse PR number")
		}

		// Get the branch for this PR
		branch, err = provider.GetPRBranch(cmd.Context(), proj.LocalPath, prNum)
		if err != nil {
			return eris.Wrap(err, "failed to get PR branch")
		}

		disp.Printf(
			"%s Switching to PR #%d branch: %s\n",
			disp.InfoText("→"),
			prNum,
			disp.Bold(branch),
		)
	} else if len(args) > 0 {
		branch = args[0]
	} else {
		// No branch specified
		if !tty.IsInteractive() {
			return eris.New("branch argument required in noninteractive mode (usage: sesh switch <branch>)")
		}

		// Use streaming fuzzy finder in interactive mode
		// Start git fetch in background - don't wait for it
		go func() {
			if err := git.Fetch(proj.LocalPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: git fetch failed: %s\n", eris.ToString(err, true))
			}
		}()

		// Stream branches directly from git to fzf for instant UI
		branchReader, err := git.StreamRemoteBranches(cmd.Context(), proj.LocalPath)
		if err != nil {
			return eris.Wrap(err, "failed to start branch listing")
		}

		// Get binary path for preview command
		bin, err := os.Executable()
		if err != nil {
			// Fallback to simple selection without preview if we can't get binary path
			selectedBranch, err := fuzzy.SelectBranchFromReader(branchReader)
			if err != nil {
				return eris.Wrap(err, "failed to select branch")
			}
			branch = selectedBranch
		} else {
			// Use preview command with absolute binary path
			// Pass the project name and branch to the info command
			// The info command will generate the proper session name internally
			previewCmd := fmt.Sprintf("%s info --project %s {}", bin, proj.Name)
			selectedBranch, err := fuzzy.SelectBranchFromReaderWithPreview(branchReader, previewCmd)
			if err != nil {
				return eris.Wrap(err, "failed to select branch")
			}
			branch = selectedBranch
		}
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	_ = cleanOrphanedSessions(proj, sessionMgr, disp)

	// Check if worktree already exists in filesystem
	existingWorktree, err := state.GetWorktree(proj, branch)
	if err == nil && existingWorktree != nil {
		// Worktree exists, attach to existing or create new session
		disp.Printf(
			"%s %s\n",
			disp.InfoText("→"),
			disp.Bold(fmt.Sprintf("Switching to existing worktree: %s", existingWorktree.Path)),
		)

		// Generate session name
		sessionName := workspace.GenerateSessionName(proj.Name, branch)

		// Check if session is running
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return eris.Wrap(err, "failed to check session existence")
		}

		if exists {
			// Record session history before attaching
			recordSessionHistory(sessionName, proj.Name, branch)

			// In noninteractive mode or detached mode, don't attach
			if !tty.IsInteractive() || switchDetach {
				disp.Printf(
					"%s Session %s already exists\n",
					disp.SuccessText("✓"),
					disp.Bold(sessionName),
				)
				return nil
			}

			// Attach to existing session
			return sessionMgr.Attach(sessionName)
		}

		// Session doesn't exist, create it
		disp.Printf(
			"%s Creating %s session %s\n",
			disp.InfoText("✨"),
			sessionMgr.Name(),
			disp.Bold(sessionName),
		)
		if err := sessionMgr.Create(sessionName, existingWorktree.Path); err != nil {
			return eris.Wrap(err, "failed to create session")
		}

		// Execute startup command if configured
		startupCmd := getStartupCommand(cfg, existingWorktree.Path)
		if startupCmd != "" && sessionMgr.Name() == "tmux" {
			disp.Printf(
				"%s Running startup command: %s\n",
				disp.InfoText("⚙"),
				disp.Faint(startupCmd),
			)
			if tmuxMgr, ok := sessionMgr.(*session.TmuxManager); ok {
				if err := tmuxMgr.SendKeys(sessionName, startupCmd); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to run startup command: %v\n", err)
				}
			}
		}

		// Record session history before attaching
		recordSessionHistory(sessionName, proj.Name, branch)

		// In noninteractive mode or detached mode, don't attach
		if !tty.IsInteractive() || switchDetach {
			disp.Printf(
				"%s Session %s created successfully\n",
				disp.SuccessText("✓"),
				disp.Bold(sessionName),
			)
			return nil
		}

		return sessionMgr.Attach(sessionName)
	}

	// Worktree doesn't exist, check branch existence
	exists, _, err := git.DoesBranchExist(proj.LocalPath, branch)
	if err != nil {
		return eris.Wrap(err, "failed to check branch existence")
	}

	// Get worktree path
	projectPath := workspace.GetProjectPath(cfg.WorkspaceDir, proj.Name)
	worktreePath := workspace.GetWorktreePath(projectPath, branch)

	// Create worktree based on branch state
	if exists {
		// Branch exists, create worktree from it
		// In bare repos (which sesh uses), this automatically sets up tracking to origin
		disp.Printf("%s Creating worktree for branch: %s\n", disp.InfoText("✨"), disp.Bold(branch))
		if err := git.CreateWorktree(proj.LocalPath, branch, worktreePath); err != nil {
			return eris.Wrap(err, "failed to create worktree from branch")
		}
	} else {
		// Branch doesn't exist, create new branch and worktree
		disp.Printf("%s Creating new branch and worktree: %s\n", disp.SuccessText("✨"), disp.Bold(branch))
		if err := git.CreateWorktreeNewBranch(proj.LocalPath, branch, worktreePath, "HEAD"); err != nil {
			return eris.Wrap(err, "failed to create worktree with new branch")
		}
	}

	// Create session
	sessionName := workspace.GenerateSessionName(proj.Name, branch)
	disp.Printf(
		"%s Creating %s session %s\n",
		disp.InfoText("✨"),
		sessionMgr.Name(),
		disp.Bold(sessionName),
	)
	if err := sessionMgr.Create(sessionName, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create session")
	}

	disp.Printf("\n%s Successfully switched to %s\n", disp.SuccessText("✓"), disp.Bold(branch))
	disp.Printf("  %s %s\n", disp.Faint("Worktree:"), worktreePath)
	disp.Printf("  %s %s\n", disp.Faint("Session:"), sessionName)

	// Execute startup command if configured
	startupCmd := getStartupCommand(cfg, worktreePath)
	if startupCmd != "" && sessionMgr.Name() == "tmux" {
		disp.Printf("%s Running startup command: %s\n", disp.InfoText("⚙"), disp.Faint(startupCmd))
		if tmuxMgr, ok := sessionMgr.(*session.TmuxManager); ok {
			if err := tmuxMgr.SendKeys(sessionName, startupCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to run startup command: %v\n", err)
			}
		}
	}

	// Record session history before attaching
	recordSessionHistory(sessionName, proj.Name, branch)

	// In noninteractive mode or detached mode, don't attach
	if !tty.IsInteractive() || switchDetach {
		return nil
	}

	// Attach to session
	disp.Printf("\n%s Attaching to session...\n", disp.InfoText("→"))
	return sessionMgr.Attach(sessionName)
}

// recordSessionHistory records the session access in the database for session history (pop command)
// This is a best-effort operation - errors are logged but don't fail the command
func recordSessionHistory(sessionName, projectName, branch string) {
	// Get database path
	dbPath, err := config.GetDBPath()
	if err != nil {
		// Silently fail - session history is not critical
		return
	}

	// Ensure config directory exists (for database file)
	if err := config.EnsureConfigDir(); err != nil {
		return
	}

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		return
	}
	defer database.Close()

	// Add session to history
	_ = db.AddSessionHistory(database, sessionName, projectName, branch)
}

// getStartupCommand returns the startup command following the priority hierarchy:
// 1. Command-line flag (highest priority)
// 2. Per-project config (.sesh.yaml in worktree)
// 3. Global config
// 4. Empty string (no command)
func getStartupCommand(cfg *config.Config, worktreePath string) string {
	// 1. Check command-line flag
	if switchStartupCommand != "" {
		return switchStartupCommand
	}

	// 2. Check per-project config
	startupCmd, err := config.GetStartupCommand(worktreePath)
	if err == nil && startupCmd != "" {
		return startupCmd
	}

	// 3. Return global config (already loaded in cfg)
	return cfg.StartupCommand
}

// cloneRepository clones a repository into the workspace
// This is used when auto-cloning a repository specified by git URL
func cloneRepository(cfg *config.Config, remoteURL, projectName string) error {
	disp := display.NewStderr()

	// Ensure workspace directory exists
	if err := config.EnsureWorkspaceDir(); err != nil {
		return eris.Wrap(err, "failed to ensure workspace directory")
	}

	// Get project path in workspace
	projectPath := workspace.GetProjectPath(cfg.WorkspaceDir, projectName)

	// Clone repository as bare repo
	bareRepoPath := filepath.Join(projectPath, ".git")
	disp.Printf("%s Cloning %s\n", disp.InfoText("⬇"), disp.Bold(remoteURL))
	disp.Printf("  %s %s\n", disp.Faint("→"), projectPath)
	if err := git.Clone(remoteURL, bareRepoPath); err != nil {
		return eris.Wrap(err, "failed to clone repository")
	}

	// Get default branch
	defaultBranch, err := git.GetDefaultBranch(bareRepoPath)
	if err != nil {
		return eris.Wrap(err, "failed to get default branch")
	}

	// Create main worktree
	worktreePath := workspace.GetWorktreePath(projectPath, defaultBranch)
	disp.Printf(
		"%s Creating worktree for branch %s\n",
		disp.InfoText("✨"),
		disp.Bold(defaultBranch),
	)
	if err := git.CreateWorktree(bareRepoPath, defaultBranch, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create worktree")
	}

	disp.Printf("%s Successfully cloned %s\n", disp.SuccessText("✓"), disp.Bold(projectName))

	return nil
}

// runInteractiveProjectSessionSelection handles the case when --select-project flag is provided
// It fuzzy searches projects first, then sessions, and attaches to the selected session
func runInteractiveProjectSessionSelection(cfg *config.Config) error {
	disp := display.NewStderr()

	// Check if running in interactive mode
	if !tty.IsInteractive() {
		return eris.New("interactive mode not available in noninteractive environment")
	}

	// Initialize session manager first to get all sessions
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Get all sessions
	sessions, err := sessionMgr.List()
	if err != nil {
		return eris.Wrap(err, "failed to list sessions")
	}

	if len(sessions) == 0 {
		return eris.New("no active sessions found. Create a session first with 'sesh switch <branch>'")
	}

	// Step 1: Discover projects and filter to only those with active sessions
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return eris.Wrap(err, "failed to discover projects")
	}

	if len(projects) == 0 {
		return eris.New("no projects found in workspace")
	}

	// Build a map of projects that have active sessions
	projectSessionMap := make(map[string][]string)
	for _, sess := range sessions {
		// Session names follow the pattern: {project-name}_{branch}
		// Find which project this session belongs to
		for _, proj := range projects {
			projectPrefix := proj.Name + "_"
			if strings.HasPrefix(sess, projectPrefix) {
				projectSessionMap[proj.Name] = append(projectSessionMap[proj.Name], sess)
				break
			}
		}
	}

	if len(projectSessionMap) == 0 {
		return eris.New("no projects with active sessions found")
	}

	// Create list of project names that have sessions
	var projectNames []string
	projectMap := make(map[string]*models.Project)
	for _, proj := range projects {
		if _, hasSession := projectSessionMap[proj.Name]; hasSession {
			projectNames = append(projectNames, proj.Name)
			projectMap[proj.Name] = proj
		}
	}

	// Create reader from project names
	projectReader := io.NopCloser(strings.NewReader(strings.Join(projectNames, "\n")))

	// Use fuzzy finder to select project
	disp.Printf("%s Select a project:\n", disp.InfoText("→"))
	selectedProjectName, err := fuzzy.SelectBranchFromReader(projectReader)
	if err != nil {
		return eris.Wrap(err, "failed to select project")
	}

	selectedProject := projectMap[selectedProjectName]
	if selectedProject == nil {
		return eris.Errorf("selected project not found: %s", selectedProjectName)
	}

	// Step 2: Get sessions for the selected project
	projectSessions := projectSessionMap[selectedProject.Name]
	projectPrefix := selectedProject.Name + "_"

	// Create reader from session names
	sessionReader := io.NopCloser(strings.NewReader(strings.Join(projectSessions, "\n")))

	// Use fuzzy finder to select session
	disp.Printf("%s Select a session:\n", disp.InfoText("→"))
	selectedSession, err := fuzzy.SelectBranchFromReader(sessionReader)
	if err != nil {
		return eris.Wrap(err, "failed to select session")
	}

	// Extract branch name from session name
	// Session format: {project-name}_{branch}
	branch := strings.TrimPrefix(selectedSession, projectPrefix)

	// Record session history
	recordSessionHistory(selectedSession, selectedProject.Name, branch)

	// Step 3: Attach to the selected session
	disp.Printf("\n%s Attaching to session %s...\n", disp.InfoText("→"), disp.Bold(selectedSession))
	return sessionMgr.Attach(selectedSession)
}
