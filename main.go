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

//go:embed .env
var embeddedEnv string

var messages []openai.ChatCompletionMessageParamUnion

func main() {
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

	myApp := app.New()
	myWindow := myApp.NewWindow("Go AI Assistant")
	myWindow.Resize(fyne.NewSize(500, 450))

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter your prompt here...")
	loadingLabel := widget.NewLabel("AI is thinking...")
	loadingLabel.Hide()

	chatContainer := container.NewVBox()
	scroll := container.NewVScroll(chatContainer)
	scroll.SetMinSize(fyne.NewSize(480, 300))

	roleOptions := []string{"General Assistant", "Programming Expert", "Writing Assistant"}
	rolePrompts := map[string]string{
		"General Assistant":  "You are a helpful AI assistant.",
		"Programming Expert": "You are a senior software engineer helping developers.",
		"Writing Assistant":  "You are a professional writing assistant.",
	}
	selectedRole := "General Assistant"
	messages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(rolePrompts[selectedRole])}
	initializingRole := true
	roleSelect := widget.NewSelect(roleOptions, func(role string) {
		if role == "" {
			return
		}
		selectedRole = role
		messages = []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(rolePrompts[selectedRole])}
		if initializingRole {
			return
		}
		chatContainer.Objects = nil
		chatContainer.Add(widget.NewLabel("System: Role changed to " + selectedRole + ". Conversation reset."))
		chatContainer.Refresh()
		scroll.ScrollToBottom()
	})
	roleSelect.SetSelected(selectedRole)
	initializingRole = false

	var btn *widget.Button
	var saveBtn *widget.Button

	btn = widget.NewButton("Send", func() {
		userPrompt := input.Text
		if userPrompt == "" {
			chatContainer.Add(widget.NewLabel("System: Please enter a prompt."))
			scroll.ScrollToBottom()
			return
		}
		chatContainer.Add(widget.NewLabel("User: " + userPrompt))
		scroll.ScrollToBottom()
		input.SetText("")

		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			if embeddedEnvParseErr != "" {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Error: embedded .env parse failed (%s) and OPENAI_API_KEY is not set.", embeddedEnvParseErr)))
			} else {
				chatContainer.Add(widget.NewLabel("System: Error: OPENAI_API_KEY not found in embedded .env or environment."))
			}
			scroll.ScrollToBottom()
			return
		}

		btn.Disable()
		btn.SetText("Thinking...")
		loadingLabel.Show()
		if len(messages) == 0 {
			messages = append(messages, openai.SystemMessage(rolePrompts[selectedRole]))
		}
		messages = append(messages, openai.UserMessage(userPrompt))

		go func(button *widget.Button) {
			client := openai.NewClient(option.WithAPIKey(apiKey))
			stream := client.Chat.Completions.NewStreaming(context.Background(), openai.ChatCompletionNewParams{
				Messages: messages,
				Model: openai.ChatModelGPT4o,
			})
			defer stream.Close()

			response := ""
			aiLabel := widget.NewLabel("AI: ")
			fyne.Do(func() {
				chatContainer.Add(aiLabel)
				scroll.ScrollToBottom()
			})

			for stream.Next() {
				chunk := stream.Current()
				for _, choice := range chunk.Choices {
					if choice.Delta.Content == "" {
						continue
					}
					response += choice.Delta.Content
					currentText := response
					fyne.Do(func() {
						aiLabel.SetText("AI: " + currentText)
						scroll.ScrollToBottom()
					})
				}
			}

			if err := stream.Err(); err != nil {
				fyne.Do(func() {
					chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Stream error: %v", err)))
					scroll.ScrollToBottom()
				})
			} else {
				messages = append(messages, openai.AssistantMessage(response))
			}
			
			fyne.Do(func() {
				loadingLabel.Hide()
				button.Enable()
				button.SetText("Send")
			})
		}(btn)
	})

	saveBtn = widget.NewButton("Save Chat", func() {
		file, err := os.Create("chat_history.txt")
		if err != nil {
			chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed to save chat: %v", err)))
			scroll.ScrollToBottom()
			return
		}
		defer file.Close()

		snapshot := append([]openai.ChatCompletionMessageParamUnion(nil), messages...)
		for _, msg := range snapshot {
			raw, err := msg.MarshalJSON()
			if err != nil {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed to serialize a message: %v", err)))
				scroll.ScrollToBottom()
				continue
			}

			var item struct {
				Role    string      `json:"role"`
				Content interface{} `json:"content"`
			}
			if err := json.Unmarshal(raw, &item); err != nil {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed to parse a message: %v", err)))
				scroll.ScrollToBottom()
				continue
			}

			prefix := item.Role
			if item.Role == "user" {
				prefix = "User"
			} else if item.Role == "assistant" {
				prefix = "AI"
			}

			contentText := ""
			switch content := item.Content.(type) {
			case string:
				contentText = content
			default:
				contentText = fmt.Sprintf("%v", content)
			}

			if _, err := file.WriteString(fmt.Sprintf("%s: %s\n", prefix, contentText)); err != nil {
				chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed writing chat file: %v", err)))
				scroll.ScrollToBottom()
				return
			}

			if prefix == "AI" {
				if _, err := file.WriteString("\n"); err != nil {
					chatContainer.Add(widget.NewLabel(fmt.Sprintf("System: Failed writing chat file: %v", err)))
					scroll.ScrollToBottom()
					return
				}
			}
		}

		chatContainer.Add(widget.NewLabel("System: Chat saved to chat_history.txt"))
		scroll.ScrollToBottom()
	})

	myWindow.SetContent(container.NewVBox(
		widget.NewLabel("AI Assistant"),
		container.NewHBox(widget.NewLabel("Role:"), roleSelect),
		scroll,
		loadingLabel,
		input,
		container.NewHBox(btn, saveBtn),
	))

	myWindow.ShowAndRun()
}