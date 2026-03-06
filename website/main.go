package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed templates/*.html static/css/*.css static/js/*.js
var content embed.FS

var templates *template.Template

func init() {
	funcMap := template.FuncMap{
		"join": strings.Join,
	}
	templates = template.Must(template.New("").Funcs(funcMap).ParseFS(content, "templates/*.html"))
}

func main() {
	// Static files
	http.Handle("/static/", http.FileServer(http.FS(content)))

	// Pages
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/docs", handleDocs)
	http.HandleFunc("/docs/", handleDocs)
	http.HandleFunc("/docs/commands", handleDocsCommands)
	http.HandleFunc("/docs/config", handleDocsConfig)
	http.HandleFunc("/docs/getting-started", handleDocsGettingStarted)

	log.Println("🚀 gflow website running at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

type PageData struct {
	Title       string
	Description string
	Page        string
	Commands    []Command
	Features    []Feature
}

type Command struct {
	Name        string
	Alias       string
	Description string
	Usage       string
	Flags       []string
	Icon        string
}

type Feature struct {
	Title       string
	Description string
	Icon        string
	Tag         string
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := PageData{
		Title:       "gflow — opinionated git workflow CLI",
		Description: "One CLI to replace git-flow, gh, and hub. Branch, commit, PR, merge — all in one command.",
		Page:        "home",
		Features: []Feature{
			{Title: "One Command PRs", Description: "Stage, commit, push, and open a PR — all in one shot. gflow pr does everything.", Icon: "⚡", Tag: "workflow"},
			{Title: "Real-Time Dashboard", Description: "gflow dash fetches branch, PRs, CI status, and commits in parallel using Go's goroutines.", Icon: "📊", Tag: "visibility"},
			{Title: "Conventional Commits", Description: "Interactive commit helper enforces your team's commit conventions. No more messy history.", Icon: "✨", Tag: "consistency"},
			{Title: "CI-Aware Merging", Description: "gflow finish checks GitHub CI status before merging. Failing checks block the merge.", Icon: "🛡", Tag: "safety"},
			{Title: "Branch Protection", Description: "gflow protect sets up GitHub branch rules from your terminal. No UI clicking needed.", Icon: "🔒", Tag: "security"},
			{Title: "Team Conventions", Description: ".gflow.yml encodes your branching, PR, and commit conventions. Commit it. Share it.", Icon: "📋", Tag: "teams"},
		},
	}

	renderPage(w, "home", data)
}

func handleDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/getting-started", http.StatusFound)
}

func handleDocsGettingStarted(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "Getting Started — gflow docs",
		Page:  "docs-getting-started",
	}
	renderPage(w, "docs-getting-started", data)
}

func handleDocsCommands(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "Commands — gflow docs",
		Page:  "docs-commands",
		Commands: []Command{
			{Name: "init", Description: "Interactive setup wizard. Creates .gflow.yml in your repo root.", Usage: "gflow init", Icon: "⚙️"},
			{Name: "start", Description: "Create a new branch from your base branch with the right prefix.", Usage: "gflow start <type> <name>", Icon: "🌱"},
			{Name: "pr", Description: "Stage, commit, push, and open a PR — all in one command.", Usage: "gflow pr [title]", Icon: "⚡"},
			{Name: "finish", Description: "Check CI, merge PR, delete branch, switch to main. Full cleanup.", Usage: "gflow finish", Icon: "✅"},
			{Name: "dash", Description: "Real-time dashboard showing branch, PRs, CI, commits — fetched in parallel.", Usage: "gflow dash", Icon: "📊"},
			{Name: "commit", Description: "Interactive conventional commit with type/scope selection.", Usage: "gflow commit", Icon: "✨"},
			{Name: "sync", Description: "Rebase your branch on the latest base branch.", Usage: "gflow sync", Icon: "🔄"},
			{Name: "release", Description: "Version bump, changelog generation, tag, and publish.", Usage: "gflow release [major|minor|patch]", Icon: "🏷"},
			{Name: "protect", Description: "Set up GitHub branch protection rules via the API.", Usage: "gflow protect [branch]", Icon: "🔒"},
			{Name: "checkout", Description: "Fuzzy-switch between branches with an interactive picker.", Usage: "gflow checkout", Icon: "🔀"},
			{Name: "log", Description: "Pretty commit log with type icons and branch context.", Usage: "gflow log", Icon: "📜"},
			{Name: "diff", Description: "Show changes between your branch and base with file icons.", Usage: "gflow diff", Icon: "📝"},
			{Name: "clean", Description: "Prune merged and stale branches, local and remote.", Usage: "gflow clean", Icon: "🧹"},
			{Name: "status", Description: "Show branch status, PR info, and sync state.", Usage: "gflow status", Icon: "📍"},
			{Name: "config", Description: "View or edit your .gflow.yml configuration.", Usage: "gflow config", Icon: "🔧"},
			{Name: "reopen", Description: "Mark a draft PR ready or reopen a closed PR.", Usage: "gflow reopen [number]", Icon: "♻️"},
			{Name: "pr list", Alias: "pr ls", Description: "List all open PRs with branch and status info.", Usage: "gflow pr list", Icon: "📋"},
			{Name: "pr view", Description: "View detailed info about a PR.", Usage: "gflow pr view [number]", Icon: "👁"},
		},
	}
	renderPage(w, "docs-commands", data)
}

func handleDocsConfig(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "Configuration — gflow docs",
		Page:  "docs-config",
	}
	renderPage(w, "docs-config", data)
}

func renderPage(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name+".html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", 500)
	}
}
