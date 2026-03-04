package main

import (
	_ "embed"
	"context"
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

	output := widget.NewMultiLineEntry()
	output.Wrapping = fyne.TextWrapWord
	output.SetPlaceHolder("AI response will appear here...")

	var btn *widget.Button

	btn = widget.NewButton("Generate Response", func() {
		userPrompt := input.Text
		if userPrompt == "" {
			output.SetText("Please enter a prompt.")
			return
		}

		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			if embeddedEnvParseErr != "" {
				output.SetText(fmt.Sprintf("Error: embedded .env parse failed (%s) and OPENAI_API_KEY is not set.", embeddedEnvParseErr))
			} else {
				output.SetText("Error: OPENAI_API_KEY not found in embedded .env or environment.")
			}
			return
		}

		btn.Disable()
		btn.SetText("Thinking...")

		go func(button *widget.Button) {
			client := openai.NewClient(option.WithAPIKey(apiKey))

			resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage(userPrompt),
				},
				Model: openai.ChatModelGPT4o,
			})

			if err != nil {
				output.SetText(fmt.Sprintf("Error: %v", err))
			} else {
				output.SetText(resp.Choices[0].Message.Content)
			}
			
			button.Enable()
			button.SetText("Generate Response")
		}(btn)
	})

	myWindow.SetContent(container.NewVBox(
		widget.NewLabel("Prompt:"),
		input,
		btn,
		widget.NewLabel("Response:"),
		container.NewGridWrap(fyne.NewSize(480, 250), output),
	))

	myWindow.ShowAndRun()
}