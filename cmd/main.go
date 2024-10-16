package main

// boilerplate code of TUI / chat application is taken from
// https://github.com/charmbracelet/bubbletea/tree/master/examples/chat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var key string

func main() {

	key = os.Getenv("OPENAI_API_KEY")
	if key == "" {
		fmt.Printf("API key must be set via the 'OPENAI_API_KEY' environment variable\n\r")
		os.Exit(1)
	}

	program := tea.NewProgram(initialModel())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}

type model struct {
	viewport       viewport.Model
	messages       []string
	textarea       textarea.Model
	senderStyle    lipgloss.Style
	responderStyle lipgloss.Style
	err            error
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "What rhymes with..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to the chat room!
Type a message and press Enter to send.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:       ta,
		messages:       []string{},
		viewport:       vp,
		senderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		responderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		err:            nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			// Quit.
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "enter":
			userInput := m.textarea.Value()

			if userInput == "" {
				// Don't send empty messages.
				return m, nil
			}

			msgContents := fmt.Sprintf("What rhymes with the word '%s'?", userInput)

			client := openai.NewClient(option.WithAPIKey(key))
			chat, err := client.Chat.Completions.New(
				ctx,
				openai.ChatCompletionNewParams{
					Messages: openai.F(
						[]openai.ChatCompletionMessageParamUnion{openai.UserMessage(msgContents)},
					),
					Model: openai.F(openai.ChatModelGPT4o),
				},
			)
			if err != nil {
				fmt.Printf("error encountered while making request to OpenAI API: %s", err)
				return m, tea.Quit
			}

			// Simulate sending a message. In your application you'll want to
			// also return a custom command to send the message off to
			// a server.
			m.messages = append(m.messages, m.senderStyle.Render("You: ")+msgContents)
			m.messages = append(m.messages, m.responderStyle.Render("AI: "+chat.Choices[0].Message.Content))
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()
			return m, nil
		default:
			// Send all other keypresses to the textarea.
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}
