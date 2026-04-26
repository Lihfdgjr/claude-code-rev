package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"claudecode/internal/agents"
	"claudecode/internal/api"
	"claudecode/internal/chat"
	"claudecode/internal/commands"
	"claudecode/internal/config"
	"claudecode/internal/core"
	"claudecode/internal/hooks"
	"claudecode/internal/mcp"
	"claudecode/internal/memory"
	"claudecode/internal/oauth"
	"claudecode/internal/permissions"
	"claudecode/internal/plugins"
	"claudecode/internal/sessions"
	"claudecode/internal/skills"
	"claudecode/internal/telemetry"
	"claudecode/internal/tools"
	"claudecode/internal/ui"
)

const Version = "0.1.0-go"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println(Version)
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`claudecode - terminal coding agent

Usage:
  claudecode             start interactive session
  claudecode version     print version
  claudecode help        show this help

Environment:
  ANTHROPIC_API_KEY      required to talk to the API
  ANTHROPIC_BASE_URL     override default endpoint
  CLAUDECODE_MODEL       override default model`)
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, core.ErrConfigMissing) {
			return fmt.Errorf("ANTHROPIC_API_KEY is not set (or place a token in ~/.claude/settings.json)")
		}
		return fmt.Errorf("load config: %w", err)
	}

	transport := api.New(api.Options{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})

	gate := permissions.New(permissions.Config{
		Mode:         cfg.Permissions.Mode,
		AllowedTools: cfg.Permissions.AllowedTools,
		DeniedTools:  cfg.Permissions.DeniedTools,
	})

	// Memory & system prompt
	mem := memory.LoadProject(cfg.ProjectDir)
	sysprompt := chat.BuildSystemPrompt(mem)

	// Persistence stores
	sessionStore := sessions.New(filepath.Join(cfg.HomeDir, ".claude", "sessions"))
	oauthStore := oauth.NewStore(cfg.HomeDir)

	// Hooks
	hookCfg := loadHooks(cfg.HomeDir, cfg.ProjectDir)
	hookRunner := hooks.New(hookCfg)

	// MCP
	mcpMgr := mcp.NewManager()
	mcpServers := loadMCPServers(cfg.HomeDir, cfg.ProjectDir)
	if len(mcpServers) > 0 {
		_ = mcpMgr.Start(ctx, mcpServers)
		defer func() { _ = mcpMgr.Stop() }()
	}
	mcp.SetActive(mcpMgr)

	// Skills + sub-agent definitions
	skillLoader := skills.New(cfg.ProjectDir)
	agentDefs := agents.NewRegistry(cfg.HomeDir, cfg.ProjectDir)

	// Long-term memory store (autoDream target)
	memStore := memory.NewStore(cfg.HomeDir)
	dreamSched := &memory.DreamScheduler{
		Store:     memStore,
		Transport: transport,
		MinPeriod: 90 * time.Second,
	}

	// Plugins
	plugLoader := plugins.New(cfg.HomeDir, cfg.ProjectDir)
	loadedPlugins, _ := plugLoader.Load()

	// Tool registries
	baseTools := tools.Default()
	for _, t := range mcpMgr.Tools() {
		baseTools.Add(t)
	}
	baseTools.Add(tools.NewSkill(skillLoader))
	for _, p := range loadedPlugins {
		for _, t := range p.Tools {
			baseTools.Add(t)
		}
	}

	// Spawner uses base tools (no Agent — sub-agents can't recurse).
	spawner := chat.NewSpawner(chat.SpawnerConfig{
		Transport:   transport,
		Tools:       baseTools,
		Permissions: gate,
		Model:       cfg.Model,
	})

	// Parent tools = base + Agent (top-level can spawn sub-agents).
	parentTools := tools.New(append(baseTools.All(), tools.NewAgent(spawner, agentDefs)))
	parentTools.Add(tools.NewToolSearch(parentTools))

	// Telemetry (best-effort, append-only JSONL log).
	telLogger := telemetry.New(telemetry.DefaultPath())
	telemetry.SetGlobal(telLogger)
	defer func() { _ = telLogger.Close() }()

	// Session id for this run + transcript root
	sessionID := sessions.NewID()
	transcriptRoot := filepath.Join(cfg.HomeDir, ".claude", "sessions")
	transcript, _ := sessions.NewRecorder(transcriptRoot, sessionID)
	if transcript != nil {
		defer func() { _ = transcript.Close() }()
	}

	// Commands
	cmdReg := commands.Default(commands.Deps{
		SessionStore:   sessionStore,
		OAuthStore:     oauthStore,
		HooksCfg:       hookCfg,
		MemoryStore:    memStore,
		Transport:      transport,
		TranscriptRoot: filepath.Join(transcriptRoot, "transcripts"),
	})
	cmdReg.Add(commands.NewPlugins(plugLoader))
	for _, p := range loadedPlugins {
		for _, c := range p.Commands {
			cmdReg.Add(c)
		}
	}

	// Auto-compact when context exceeds threshold
	autoCompact := chat.NewAutoCompact(chat.AutoCompactConfig{Threshold: 0.75, MinPeriod: 2 * time.Minute})

	var driver core.Driver
	driver = chat.NewDriver(chat.Config{
		Transport:    transport,
		Tools:        parentTools,
		Commands:     cmdReg,
		Permissions:  gate,
		Hooks:        hookRunner,
		Transcript:   transcript,
		AutoCompact:  autoCompact,
		Model:        cfg.Model,
		SystemPrompt: sysprompt,
		SessionID:    sessionID,
		OnTurnDone: func(history []core.Message) {
			snap := sessions.SnapshotFromSession(sessionID, driver.Session())
			if err := sessionStore.Save(sessionID, snap); err != nil {
				fmt.Fprintf(os.Stderr, "session save: %v\n", err)
			}
		},
		OnPostTurn: func(history []core.Message) {
			dreamSched.ModelOf = driver.Session().Model
			dreamSched.HistoryOf = func() []core.Message { return history }
			dreamSched.MaybeRun()
		},
	})

	app := ui.New(driver)
	return app.Run(ctx)
}

func loadHooks(homeDir, projectDir string) hooks.Config {
	merged := hooks.Config{}
	for _, p := range settingsPaths(homeDir, projectDir) {
		c, err := hooks.LoadFromSettings(p)
		if err != nil || len(c) == 0 {
			continue
		}
		for k, specs := range c {
			merged[k] = append(merged[k], specs...)
		}
	}
	return merged
}

func loadMCPServers(homeDir, projectDir string) map[string]mcp.Config {
	servers := map[string]mcp.Config{}
	type fileShape struct {
		MCPServers map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
	for _, path := range settingsPaths(homeDir, projectDir) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var fs fileShape
		if err := json.Unmarshal(data, &fs); err != nil {
			continue
		}
		for name, c := range fs.MCPServers {
			servers[name] = mcp.Config{
				Name:    name,
				Command: c.Command,
				Args:    c.Args,
				Env:     c.Env,
			}
		}
	}
	return servers
}

func settingsPaths(homeDir, projectDir string) []string {
	var out []string
	if homeDir != "" {
		out = append(out, filepath.Join(homeDir, ".claude", "settings.json"))
	}
	if projectDir != "" {
		out = append(out, filepath.Join(projectDir, ".claude", "settings.json"))
	}
	return out
}
