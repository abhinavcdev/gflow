<p align="center">
  <h1 align="center">gflow</h1>
  <p align="center"><strong>Opinionated git workflow CLI — replaces git-flow, gh, and hub</strong></p>
  <p align="center">
    <a href="#installation">Install</a> •
    <a href="#quick-start">Quick Start</a> •
    <a href="#commands">Commands</a> •
    <a href="#configuration">Configuration</a> •
    <a href="#custom-strategies">Custom Strategies</a> •
    <a href="#providers">Providers</a>
  </p>
</p>

---

**gflow** encodes your team's branching + PR conventions into one CLI.  
`gflow pr` creates a branch, commits, pushes, opens a PR with a template, assigns reviewers — **in one command**.

Works with **GitHub**, **GitLab**, and **Bitbucket**.

```
$ gflow pr "Add user authentication"

  Create Pull Request

  ✓ Staged all changes
  ✓ Committed: feat: Add user authentication
  ✓ Pushed feature/user-auth to origin
  ✓ Pull request created!

  ✓ PR #42 created
    URL:    https://github.com/yourorg/yourrepo/pull/42
    Branch: feature/user-auth → main
```

## Why gflow?

| Feature | git-flow | gh | hub | **gflow** |
|---|---|---|---|---|
| Branch naming conventions | ✓ | ✗ | ✗ | **✓** |
| One-command PR creation | ✗ | Partial | Partial | **✓** |
| Auto-assign reviewers | ✗ | ✗ | ✗ | **✓** |
| PR templates | ✗ | ✗ | ✗ | **✓** |
| Commit conventions | ✗ | ✗ | ✗ | **✓** |
| Custom strategies | ✗ | ✗ | ✗ | **✓** |
| Multi-provider (GH/GL/BB) | ✗ | GitHub only | GitHub only | **✓** |
| Lifecycle hooks | ✗ | ✗ | ✗ | **✓** |
| Team config file | ✗ | ✗ | ✗ | **✓** |

## Installation

### Go Install

```bash
go install github.com/abhinavcdev/gflow@latest
```

### Homebrew (coming soon)

```bash
brew install gflow
```

### From Source

```bash
git clone https://github.com/abhinavcdev/gflow.git
cd gflow
make build
# Binary is at ./bin/gflow
```

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/abhinavcdev/gflow/releases).

## Quick Start

### 1. Initialize in your repo

```bash
cd your-project
gflow init
```

This launches an interactive wizard that creates a `.gflow.yml` config file:

```
   __ _  __| | _____      __
  / _` |/ _` |/ _ \ \ /\ / /
 | (_| | (_| | (_) \ V  V /
  \__, |\__,_|\___/ \_/\_/
  |___/
  opinionated git workflow CLI

  Initialize gflow

[1] Git Provider
    Detected: github (yourorg/yourrepo)
  ✓ Provider: github

[2] Branching Strategy
  ✓ Main branch: main
  ✓ Prefixes: conventional

[3] Pull Request Conventions
  ✓ Merge method: squash
  ✓ Auto-assign: true

[4] Commit Conventions
  ✓ Convention: conventional

  ✓ Created .gflow.yml
```

### 2. Set up your GitHub token

gflow needs a GitHub Personal Access Token to create PRs, manage releases, etc.

**Generate a token:**

1. Go to [github.com/settings/tokens](https://github.com/settings/tokens)
2. Click **"Generate new token"** → **"Generate new token (classic)"**
3. Name it `gflow`
4. Select the **`repo`** scope (full control of private repositories)
5. Click **"Generate token"** and copy it

**Set it in your shell:**

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

**Make it permanent** (add to your shell profile):

```bash
# zsh (default on macOS)
echo 'export GITHUB_TOKEN=ghp_xxxxxxxxxxxx' >> ~/.zshrc
source ~/.zshrc

# bash
echo 'export GITHUB_TOKEN=ghp_xxxxxxxxxxxx' >> ~/.bashrc
source ~/.bashrc
```

> **Security:** Never commit your token to git. gflow reads it from the `GITHUB_TOKEN` environment variable — the token is never stored in `.gflow.yml`.

### 3. Start working

```bash
gflow start feature user-auth    # Create branch feature/user-auth
# ... make changes ...
gflow pr                          # Stage, commit, push, open PR
gflow finish                      # Merge PR, delete branch, cleanup
gflow dash                        # See your whole project at a glance
```

## Commands

### `gflow init`

Interactive setup wizard. Creates `.gflow.yml` in your repo root.

### `gflow start <type> <name>`

Create a new branch following your conventions.

```bash
gflow start feature user-auth       # → feature/user-auth
gflow start bugfix login-redirect   # → bugfix/login-redirect
gflow start hotfix critical-fix     # → hotfix/critical-fix
gflow start release 1.2.0           # → release/1.2.0
gflow start my-strategy task-name   # → custom strategy from config
```

**Flags:**
- `--base, -b` — override base branch
- `--from-current` — branch from current HEAD instead of base

### `gflow pr [title]`

Create a pull request from the current branch. The flagship command.

```bash
gflow pr                                # Interactive
gflow pr "Add user authentication"      # With title
gflow pr --draft                        # As draft
gflow pr -r alice -r bob                # With reviewers
gflow pr --label bug --label urgent     # With labels
gflow pr -m "feat: add auth"            # Auto-commit with message
```

**What it does:**
1. Detects uncommitted changes → stages & commits
2. Pushes branch to remote
3. Generates PR title from branch name + commit convention
4. Generates PR body from template or commit log
5. Creates PR via provider API
6. Assigns reviewers from config
7. Adds labels based on branch type

**Flags:**
- `--base, -b` — target branch
- `--draft, -d` — create as draft
- `--reviewer, -r` — add reviewer (repeatable)
- `--team-reviewer, -t` — add team reviewer (repeatable)
- `--label, -l` — add label (repeatable)
- `--body` — PR description
- `--message, -m` — commit message
- `--no-edit` — skip interactive prompts
- `--push` — push before creating (default: true)

### `gflow finish [branch]`

Merge the PR, delete the branch, and clean up.

```bash
gflow finish                    # Finish current branch
gflow finish feature/my-thing   # Finish specific branch
gflow finish --method squash    # Override merge method
gflow finish --no-delete        # Keep branch after merge
gflow finish --force            # Local merge without PR
```

### `gflow sync`

Keep your branch up-to-date with upstream.

```bash
gflow sync              # Rebase on default base
gflow sync --base main  # Rebase on specific branch
gflow sync --merge      # Merge instead of rebase
```

### `gflow release <version>`

Full release workflow — tag, changelog, publish.

```bash
gflow release 1.2.0          # Create release v1.2.0
gflow release patch           # Auto-bump patch (0.1.0 → 0.1.1)
gflow release minor           # Auto-bump minor (0.1.0 → 0.2.0)
gflow release major           # Auto-bump major (0.1.0 → 1.0.0)
gflow release 1.2.0 --draft   # Create as draft
```

**What it does:**
1. Determines version (or auto-bumps)
2. Generates changelog from commits
3. Creates annotated git tag
4. Pushes tag to remote
5. Creates GitHub/GitLab release with changelog

### `gflow status`

Show comprehensive branch status.

```bash
gflow status
```

Displays: current branch, type, base, working tree, commits ahead, associated PR info.

### `gflow config`

View or modify configuration.

```bash
gflow config                        # Show current config
gflow config --path                 # Show config file path
gflow config set pr.draft true      # Set a value
gflow config strategies             # List all strategies
```

### `gflow commit [message]`

Interactive conventional commit helper.

```bash
gflow commit                          # Full interactive flow
gflow commit "add user auth"          # Auto-detect type from branch
gflow commit -t feat -m "add auth"    # Specify type and message
gflow commit --all                    # Stage all before committing
gflow commit --amend                  # Amend previous commit
```

Interactively selects commit type (feat, fix, docs, etc.), optional scope, and description — then formats the message according to your team's convention.

### `gflow checkout [query]`

Fuzzy branch switcher with interactive picker. Aliases: `co`, `switch`.

```bash
gflow checkout                  # Interactive branch picker
gflow checkout auth             # Switch to branch matching "auth"
gflow checkout feature/user     # Direct checkout
gflow co auth                   # Short alias
gflow checkout -r               # Include remote branches
```

### `gflow log`

Pretty commit log with type icons and branch context.

```bash
gflow log              # Show commits since base branch
gflow log -n 20        # Show last 20 commits
gflow log --all        # Show full log, not just since base
```

### `gflow diff [base]`

Quick branch comparison summary before opening a PR.

```bash
gflow diff              # Diff against default base
gflow diff main         # Diff against specific branch
gflow diff --files      # Show only changed file names
gflow diff --full       # Show full diff output
```

### `gflow clean`

Prune merged and stale branches.

```bash
gflow clean                # Interactive cleanup
gflow clean --dry-run      # Preview what would be deleted
gflow clean --force        # Skip confirmation
gflow clean --remote       # Also delete remote branches
```

### `gflow reopen [pr-number]`

Mark a draft PR as ready for review, or reopen a closed PR.

```bash
gflow reopen             # Ready/reopen PR for current branch
gflow reopen 42          # Ready/reopen PR #42
gflow reopen --ready     # Explicitly mark draft as ready
```

### `gflow pr list`

List all open pull requests.

```bash
gflow pr list            # Show open PRs
gflow pr ls              # Short alias
```

### `gflow pr view [number]`

View detailed info about a pull request.

```bash
gflow pr view            # View PR for current branch
gflow pr view 42         # View PR #42
```

### `gflow dash`

**Real-time workflow dashboard** — your entire project at a glance. Uses Go's goroutines to fetch everything in parallel. Aliases: `dashboard`, `d`.

```bash
gflow dash
gflow d
```

```
         __ _
   __ _ / _| | _____      __
  / _` | |_| |/ _ \ \ /\ / /
 | (_| |  _| | (_) \ V  V /
  \__, |_| |_|\___/ \_/\_/
  |___/
  opinionated git workflow CLI
  fetched in 312ms using 5 parallel goroutines

  ─── BRANCH ──────────────────────────────────────
  ✨ feature/user-auth  (feature)
    base: main
    sync: ↑3 ahead  ✓ in sync

  ─── WORKING TREE ────────────────────────────────
  ●  2 staged  1 modified

  ─── PULL REQUEST ────────────────────────────────
  ●  #42 Add user authentication  Open
       https://github.com/you/repo/pull/42

  ─── OPEN PRs (2) ───────────────────────────────
  ●  #42  Add user authentication ←
  ◌  #38  Draft: refactor login flow

  ─── RECENT COMMITS ─────────────────────────────
  ✨ a1b2c3d feat: add auth middleware      2 min ago
  🐛 d4e5f6g fix: token validation          1 hour ago

  ─── QUICK ACTIONS ──────────────────────────────
  gflow commit                         conventional commit your changes
  gflow pr view                        view your open PR
```

Shows: branch state, sync status, working tree, current PR, all open PRs, recent commits with type icons, contextual quick actions — **all fetched in parallel**.

### `gflow version`

Print version and system info.

## Configuration

gflow is configured via `.gflow.yml` in your repo root. Commit this file to share conventions with your team.

### Full Example

```yaml
# gflow configuration
version: "0.1.0"

provider:
  name: github                    # github, gitlab, bitbucket
  host: ""                        # custom host for self-hosted
  token_env: GITHUB_TOKEN         # env var for auth token
  owner: your-org
  repo: your-repo

branching:
  main: main
  develop: develop
  use_develop: false              # true for git-flow style
  prefixes:
    feature: feature/
    bugfix: bugfix/
    hotfix: hotfix/
    release: release/
    support: support/

pr:
  default_base: main
  auto_assign: true               # assign PR creator
  draft: false                    # create as draft by default
  merge_method: squash            # merge, squash, rebase
  delete_branch_on_merge: true
  title_format: "{{.Type}}: {{.Description}}"
  reviewers:
    - alice
    - bob
  team_reviewers:
    - backend-team
  labels:
    - needs-review

commit:
  convention: conventional        # conventional, angular, none
  require_scope: false
  require_ticket: false
  ticket_pattern: '[A-Z]+-\d+'
  types:
    - feat
    - fix
    - docs
    - style
    - refactor
    - perf
    - test
    - build
    - ci
    - chore
    - revert

hooks:
  pre_commit: []
  pre_push: []
  pre_pr:
    - "make lint"
    - "make test"
  post_pr: []
  pre_merge: []
  post_merge:
    - "make deploy-staging"
```

## Custom Strategies

Define custom workflow strategies in `.gflow.yml` for specialized branch types:

```yaml
strategies:
  experiment:
    description: "Experimental feature branch"
    base_branch: develop
    prefix: "exp/"
    merge_method: squash
    labels:
      - experimental
    reviewers:
      - tech-lead
    hooks:
      pre_create:
        - "echo 'Starting experiment...'"
      post_create:
        - "echo 'Experiment branch ready'"
      pre_merge:
        - "make test"
        - "make benchmark"

  spike:
    description: "Technical spike / proof of concept"
    base_branch: main
    prefix: "spike/"
    merge_method: squash
    labels:
      - spike
      - tech-debt

  docs:
    description: "Documentation changes"
    base_branch: main
    prefix: "docs/"
    merge_method: squash
    labels:
      - documentation
    reviewers:
      - docs-team

  infra:
    description: "Infrastructure changes"
    base_branch: main
    prefix: "infra/"
    merge_method: merge
    labels:
      - infrastructure
      - ops
    hooks:
      pre_merge:
        - "terraform plan"
```

Use them just like built-in types:

```bash
gflow start experiment new-cache-layer
gflow start spike grpc-migration
gflow start docs api-reference
gflow start infra k8s-upgrade
```

## Providers

### GitHub
- **Auth:** `GITHUB_TOKEN` or `GH_TOKEN`
- **Self-hosted:** Set `provider.host` to your GitHub Enterprise host
- Full support: PRs, releases, reviewers, labels

### GitLab
- **Auth:** `GITLAB_TOKEN` or `GL_TOKEN`  
- **Self-hosted:** Set `provider.host` to your GitLab instance
- Full support: MRs (merge requests), releases, labels

### Bitbucket
- **Auth:** `BITBUCKET_TOKEN` or `BB_TOKEN`
- **Self-hosted:** Set `provider.host` to your Bitbucket Server host
- Support: PRs, merge methods

## Lifecycle Hooks

Run custom commands at key points in your workflow:

```yaml
hooks:
  pre_pr:
    - "make lint"      # Run linter before creating PR
    - "make test"      # Run tests before creating PR
  post_pr:
    - "slack notify"   # Notify Slack after PR creation
  post_merge:
    - "make deploy"    # Deploy after merge
```

Strategy-level hooks:

```yaml
strategies:
  hotfix:
    hooks:
      pre_merge:
        - "make test-critical"
      post_merge:
        - "make deploy-production"
```

## PR Templates

gflow automatically picks up PR templates from these locations:

1. `.github/pull_request_template.md`
2. `.github/PULL_REQUEST_TEMPLATE.md`
3. `.gitlab/merge_request_templates/Default.md`
4. `docs/pull_request_template.md`

Or specify a custom path:

```yaml
pr:
  template_path: ".github/custom_pr_template.md"
```

## Development

```bash
# Clone
git clone https://github.com/abhinavcdev/gflow.git
cd gflow

# Build
make build

# Run tests
make test

# Install locally
make install

# Release
make release
```

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<p align="center">
  Built with ❤️ for teams that care about developer experience.
</p>
