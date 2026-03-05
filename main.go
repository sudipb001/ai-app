// package main provides a desktop AI assistant built with Fyne and OpenAI API.
//
// The application features:
// - Streaming chat responses (token-by-token display)
// - Conversation memory across multiple turns
// - Role-based system prompts (General Assistant, Programming Expert, Writing Assistant)
// - Chat history export to file
// - Thread-safe UI updates from async goroutines
// - Embedded .env configuration for standalone deployment
package main

import (
	_ "embed"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// embeddedEnv holds the embedded .env file content at build time.
// This allows the application to run without a separate .env file in deployment.
//go:embed .env
var embeddedEnv string

// messages maintains the conversation history for the current session.
// It includes system prompts, user messages, and AI responses.
// Each message is sent to the OpenAI API to provide context for streaming responses.
var messages []openai.ChatCompletionMessageParamUnion

func main() {
	// Initialize environment variables from embedded .env file.
	// Variables can be overridden by system environment variables.
	embeddedEnvParseErr := ""
	envMap, err := godotenv.Unmarshal(embeddedEnv)
	if err == nil {
		for key, value := range envMap {
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}
	} else {
		embeddedEnvParseErr = err.Error()
	}

	// Initialize Fyne application and window.
	myApp := app.New()
	myWindow := myApp.NewWindow("Go AI Assistant")
	myWindow.Resize(fyne.NewSize(500, 450))

	// User input field for typing prompts.
	input := widget.NewEntry()
	input.SetPlaceHolder("Enter your prompt here...")

	// Loading indicator shown during AI response generation.
	loadingLabel := widget.NewLabel("AI is thinking...")
	loadingLabel.Hide()

	// Chat history container with scrolling support.
	// Displays message exchanges between user and AI.
	chatContainer := container.NewVBox()
	scroll := container.NewVScroll(chatContainer)
	scroll.SetMinSize(fyne.NewSize(480, 300))

	// Role selection configuration.
	// Each role has a corresponding system prompt that guides AI behavior.
	roleOptions := []string{"General Assistant", "Programming Expert", "Writing Assistant"}
	rolePrompts := map[string]string{
		"General Assistant":  "You are a helpful AI assistant.",
		"Programming Expert": "You are a senior software engineer helping developers.",
		"Writing Assistant":  "You are a professional writing assistant.",
	}
	selectedRole := "General Assistant"
	// Initialize conversation with role-specific system message.
	messages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(rolePrompts[selectedRole])}
	
	// Flag to prevent ui update during initial role selection setup.
	initializingRole := true
	
	// Role selector dropdown with callback for role changes.
	roleSelect := widget.NewSelect(roleOptions, func(role string) {
		if role == "" {
			return
		}
		selectedRole = role
		// Reset conversation when role changes.
		messages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(rolePrompts[selectedRole])}
		// Skip UI update during initialization.
		if initializingRole {
			return
		}
		// Clear chat history and notify user of role change.
		chatContainer.Objects = nil
		chatContainer.Add(widget.NewLabel("System: Role changed to " + selectedRole + ". Conversation reset."))
		chatContainer.Refresh()
		scroll.ScrollToBottom()
	})
	roleSelect.SetSelected(selectedRole)
	initializingRole = false

	var btn *widget.Button
	var saveBtn *widget.Button

	// Send button callback: processes user input and requests streaming response from OpenAI.
	btn = widget.NewButton("Send", func() {
		// Validate input.
		userPrompt := input.Text
		if userPrompt == "" {
			chatContainer.Add(widget.NewLabel("System: Please enter a prompt."))
			scroll.ScrollToBottom()
			return
		}
		
		// Display user message in chat.
		chatContainer.Add(widget.NewLabel("User: " + userPrompt))
		scroll.ScrollToBottom()
		input.SetText("")

		// Retrieve API key from environment.
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			// Show error with diagnostics about .env parsing.
			if embeddedEnvParseErr != "" {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Error: embedded .env parse failed (%s) and OPENAI_API_KEY is not set.", embeddedEnvParseErr)))
			} else {
				chatContainer.Add(widget.NewLabel("System: Error: OPENAI_API_KEY not found in embedded .env or environment."))
			}
			scroll.ScrollToBottom()
			return
		}

		// Disable button and show loading indicator.
		btn.Disable()
		btn.SetText("Thinking...")
		loadingLabel.Show()
		
		// Ensure system prompt exists before adding user message.
		if len(messages) == 0 {
			messages = append(messages, openai.SystemMessage(rolePrompts[selectedRole]))
		}
		// Append user message to conversation history.
		messages = append(messages, openai.UserMessage(userPrompt))

		// Launch async goroutine to stream OpenAI response without blocking UI.
		go func(button *widget.Button) {
			// Create OpenAI client with API key.
			client := openai.NewClient(option.WithAPIKey(apiKey))
			
			// Request streaming chat completion using full conversation history.
			stream := client.Chat.Completions.NewStreaming(context.Background(), openai.ChatCompletionNewParams{
				Messages: messages,
				Model:    openai.ChatModelGPT4o,
			})
			defer stream.Close()

			// Prepare label for incremental response display.
			response := ""
			aiLabel := widget.NewLabel("AI: ")
			fyne.Do(func() {
				chatContainer.Add(aiLabel)
				scroll.ScrollToBottom()
			})

			// Stream tokens from API and update label in real-time.
			for stream.Next() {
				chunk := stream.Current()
				for _, choice := range chunk.Choices {
					// Skip empty content deltas.
					if choice.Delta.Content == "" {
						continue
					}
					// Accumulate token content.
					response += choice.Delta.Content
					currentText := response
					// Update UI safely from goroutine using fyne.Do.
					fyne.Do(func() {
						aiLabel.SetText("AI: " + currentText)
						scroll.ScrollToBottom()
					})
				}
			}

			// Handle stream errors or append successful response to history.
			if err := stream.Err(); err != nil {
				fyne.Do(func() {
					chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Stream error: %v", err)))
					scroll.ScrollToBottom()
				})
			} else {
				// Persist AI response to conversation memory for next turn.
				messages = append(messages, openai.AssistantMessage(response))
			}
			
			// Re-enable button and hide loading indicator on main thread.
			fyne.Do(func() {
				loadingLabel.Hide()
				button.Enable()
				button.SetText("Send")
			})
		}(btn)
	})

	// Save chat button callback: exports conversation to text file.
	saveBtn = widget.NewButton("Save Chat", func() {
		// Create or overwrite chat_history.txt.
		file, err := os.Create("chat_history.txt")
		if err != nil {
			chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed to save chat: %v", err)))
			scroll.ScrollToBottom()
			return
		}
		defer file.Close()

		// Snapshot current message history to avoid concurrent modification issues.
		snapshot := append([]openai.ChatCompletionMessageParamUnion(nil), messages...)
		
		// Iterate and serialize each message.
		for _, msg := range snapshot {
			// Marshal message to JSON for role/content extraction.
			raw, err := msg.MarshalJSON()
			if err != nil {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed to serialize a message: %v", err)))
				scroll.ScrollToBottom()
				continue
			}

			// Parse JSON to extract role and content fields.
			var item struct {
				Role    string      `json:"role"`
				Content interface{} `json:"content"`
			}
			if err := json.Unmarshal(raw, &item); err != nil {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed to parse a message: %v", err)))
				scroll.ScrollToBottom()
				continue
			}

			// Convert role to display-friendly prefix.
			prefix := item.Role
			if item.Role == "user" {
				prefix = "User"
			} else if item.Role == "assistant" {
				prefix = "AI"
			}

			// Extract content as string (handle different types).
			contentText := ""
			switch content := item.Content.(type) {
			case string:
				contentText = content
			default:
				contentText = fmt.Sprintf("%v", content)
			}

			// Write message line to file.
			if _, err := file.WriteString(fmt.Sprintf("%s: %s\n", prefix, contentText)); err != nil {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed writing chat file: %v", err)))
				scroll.ScrollToBottom()
				return
			}

			// Add blank line after AI messages for readability.
			if prefix == "AI" {
				if _, err := file.WriteString("\n"); err != nil {
					chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed writing chat file: %v", err)))
					scroll.ScrollToBottom()
					return
				}
			}
		}

		// Notify user of successful save.
		chatContainer.Add(widget.NewLabel("System: Chat saved to chat_history.txt"))
		scroll.ScrollToBottom()
	})

	// Build final UI layout with all components.
	myWindow.SetContent(container.NewVBox(
		widget.NewLabel("AI Assistant"),
		container.NewHBox(widget.NewLabel("Role:"), roleSelect),
		scroll,
		loadingLabel,
		input,
		container.NewHBox(btn, saveBtn),
	))

	// Start the Fyne event loop and render the UI.
	myWindow.ShowAndRun()
}