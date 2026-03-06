#!/usr/bin/env bash
# Install shell completions for gflow
# Usage: ./scripts/install-completions.sh [bash|zsh|fish|powershell]

set -euo pipefail

SHELL_TYPE="${1:-}"

if [ -z "$SHELL_TYPE" ]; then
    # Auto-detect shell
    case "$SHELL" in
        */bash) SHELL_TYPE="bash" ;;
        */zsh)  SHELL_TYPE="zsh" ;;
        */fish) SHELL_TYPE="fish" ;;
        *)      echo "Could not detect shell. Usage: $0 [bash|zsh|fish|powershell]"; exit 1 ;;
    esac
fi

echo "Installing gflow completions for $SHELL_TYPE..."

case "$SHELL_TYPE" in
    bash)
        COMP_DIR="${BASH_COMPLETION_USER_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions}"
        mkdir -p "$COMP_DIR"
        gflow completion bash > "$COMP_DIR/gflow"
        echo "Installed to $COMP_DIR/gflow"
        echo "Restart your shell or run: source $COMP_DIR/gflow"
        ;;
    zsh)
        COMP_DIR="${fpath[1]:-$HOME/.zsh/completions}"
        mkdir -p "$COMP_DIR"
        gflow completion zsh > "$COMP_DIR/_gflow"
        echo "Installed to $COMP_DIR/_gflow"
        echo "Restart your shell or run: source $COMP_DIR/_gflow"
        ;;
    fish)
        COMP_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions"
        mkdir -p "$COMP_DIR"
        gflow completion fish > "$COMP_DIR/gflow.fish"
        echo "Installed to $COMP_DIR/gflow.fish"
        ;;
    powershell)
        echo "Add this to your PowerShell profile:"
        echo '  gflow completion powershell | Out-String | Invoke-Expression'
        ;;
    *)
        echo "Unsupported shell: $SHELL_TYPE"
        echo "Supported: bash, zsh, fish, powershell"
        exit 1
        ;;
esac

echo "Done! gflow completions installed for $SHELL_TYPE."
