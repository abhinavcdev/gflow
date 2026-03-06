package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/manifoldco/promptui"
)

var (
	// Color palette
	Primary   = lipgloss.Color("#7C3AED") // violet
	Secondary = lipgloss.Color("#06B6D4") // cyan
	Success   = lipgloss.Color("#10B981") // green
	Warning   = lipgloss.Color("#F59E0B") // amber
	Danger    = lipgloss.Color("#EF4444") // red
	Muted     = lipgloss.Color("#6B7280") // gray
	White     = lipgloss.Color("#FFFFFF")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Italic(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	BoldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1)

	StepStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)
)

// Icons
const (
	IconCheck   = "✓"
	IconCross   = "✗"
	IconArrow   = "→"
	IconDot     = "●"
	IconBranch  = ""
	IconPR      = ""
	IconRocket  = "🚀"
	IconGear    = "⚙"
	IconLock    = "🔒"
	IconTag     = "🏷"
	IconMerge   = ""
	IconWarning = "⚠"
	IconInfo    = "ℹ"
	IconStar    = "★"
)

// Banner prints the gflow banner
func Banner() {
	banner := `
         __ _                
   __ _ / _| | _____      __
  / _` + "`" + ` | |_| |/ _ \ \ /\ / /
 | (_| |  _| | (_) \ V  V / 
  \__, |_| |_|\___/ \_/\_/  
  |___/`
	fmt.Println(lipgloss.NewStyle().Foreground(Primary).Bold(true).Render(banner))
	fmt.Println(MutedStyle.Render("  opinionated git workflow CLI"))
	fmt.Println()
}

// Title prints a styled title
func Title(text string) {
	fmt.Println(TitleStyle.Render(text))
}

// Step prints a step indicator
func Step(number int, text string) {
	prefix := StepStyle.Render(fmt.Sprintf("[%d]", number))
	fmt.Printf("%s %s\n", prefix, text)
}

// StepDone prints a completed step
func StepDone(text string) {
	fmt.Printf("  %s %s\n", SuccessStyle.Render(IconCheck), text)
}

// StepFail prints a failed step
func StepFail(text string) {
	fmt.Printf("  %s %s\n", ErrorStyle.Render(IconCross), text)
}

// StepSkip prints a skipped step
func StepSkip(text string) {
	fmt.Printf("  %s %s\n", MutedStyle.Render("-"), MutedStyle.Render(text))
}

// Info prints an info message
func Info(text string) {
	fmt.Printf("  %s %s\n", lipgloss.NewStyle().Foreground(Secondary).Render(IconInfo), text)
}

// Warn prints a warning message
func Warn(text string) {
	fmt.Printf("  %s %s\n", WarningStyle.Render(IconWarning), text)
}

// Error prints an error message
func Error(text string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", ErrorStyle.Render(IconCross), text)
}

// Errorf prints a formatted error message
func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

// SuccessMsg prints a success message
func SuccessMsg(text string) {
	fmt.Printf("\n  %s %s\n", SuccessStyle.Render(IconCheck), SuccessStyle.Render(text))
}

// Detail prints a detail line (indented, muted)
func Detail(label, value string) {
	fmt.Printf("    %s %s\n", MutedStyle.Render(label+":"), value)
}

// Box prints text in a box
func Box(title, content string) {
	titleRendered := BoldStyle.Render(title)
	contentRendered := content
	boxContent := fmt.Sprintf("%s\n%s", titleRendered, contentRendered)
	fmt.Println(BoxStyle.Render(boxContent))
}

// Summary prints a summary box
func Summary(title string, items map[string]string) {
	var lines []string
	for k, v := range items {
		lines = append(lines, fmt.Sprintf("  %s %s %s",
			MutedStyle.Render(IconDot),
			BoldStyle.Render(k+":"),
			v,
		))
	}
	fmt.Println()
	fmt.Println(TitleStyle.Render("  " + title))
	fmt.Println(strings.Join(lines, "\n"))
	fmt.Println()
}

// OrderedSummary prints a summary with ordered items
func OrderedSummary(title string, items []SummaryItem) {
	fmt.Println()
	fmt.Println(TitleStyle.Render("  " + title))
	for _, item := range items {
		fmt.Printf("  %s %s %s\n",
			MutedStyle.Render(IconDot),
			BoldStyle.Render(item.Label+":"),
			item.Value,
		)
	}
	fmt.Println()
}

// SummaryItem is a key-value pair for ordered summaries
type SummaryItem struct {
	Label string
	Value string
}

// NewSpinner creates a new spinner with a message
func NewSpinner(msg string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = "  " + msg
	_ = s.Color("magenta")
	return s
}

// StartSpinner starts a spinner and returns it for stopping later
func StartSpinner(msg string) *spinner.Spinner {
	s := NewSpinner(msg)
	s.Start()
	return s
}

// StopSpinner stops the spinner with a success message
func StopSpinner(s *spinner.Spinner, msg string) {
	s.Stop()
	StepDone(msg)
}

// StopSpinnerFail stops the spinner with a failure message
func StopSpinnerFail(s *spinner.Spinner, msg string) {
	s.Stop()
	StepFail(msg)
}

// PromptInput prompts the user for text input
func PromptInput(label, defaultVal string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultVal,
		Stdin:   os.Stdin,
	}
	return prompt.Run()
}

// PromptInputRequired prompts the user for required text input
func PromptInputRequired(label, defaultVal string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultVal,
		Stdin:   os.Stdin,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("value is required")
			}
			return nil
		},
	}
	return prompt.Run()
}

// PromptSelect prompts the user to select from a list
func PromptSelect(label string, items []string) (int, string, error) {
	prompt := promptui.Select{
		Label: label,
		Items: items,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   fmt.Sprintf("%s {{ . | cyan }}", IconArrow),
			Inactive: "  {{ . }}",
			Selected: fmt.Sprintf("%s {{ . | green }}", IconCheck),
		},
		Stdin: os.Stdin,
	}
	return prompt.Run()
}

// PromptConfirm prompts the user for a yes/no confirmation
func PromptConfirm(label string, defaultYes bool) (bool, error) {
	defaultStr := "n"
	if defaultYes {
		defaultStr = "y"
	}
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
		Default:   defaultStr,
		Stdin:     os.Stdin,
	}
	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			return false, nil
		}
		return false, err
	}
	return strings.ToLower(result) == "y" || result == "", nil
}

// PromptMultiInput prompts for multiple space-separated values
func PromptMultiInput(label, defaultVal, hint string) ([]string, error) {
	displayLabel := label
	if hint != "" {
		displayLabel = fmt.Sprintf("%s %s", label, MutedStyle.Render("("+hint+")"))
	}
	result, err := PromptInput(displayLabel, defaultVal)
	if err != nil {
		return nil, err
	}
	if result == "" {
		return nil, nil
	}
	parts := strings.Fields(result)
	return parts, nil
}

// Divider prints a divider line
func Divider() {
	fmt.Println(MutedStyle.Render("  " + strings.Repeat("─", 50)))
}

// Newline prints a blank line
func Newline() {
	fmt.Println()
}
