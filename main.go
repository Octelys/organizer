package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {

	openAiApiKey := os.Getenv("OPENAI_API_KEY")

	if openAiApiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable is not set")
		os.Exit(10000)
	}

	workingDir := os.Getenv("WORKING_DIR")

	if workingDir == "" {
		fmt.Println("WORKING_DIR environment variable is not set")
		os.Exit(10001)
	}

	/* 1. Reads all the file names in the directory */

	fmt.Printf("Reading the name of the files from the working directory... ")

	files, err := os.ReadDir(workingDir)

	if err != nil {
		fmt.Printf(" [ FAILED ]\n")
		fmt.Printf("\n")
		fmt.Printf("Unable to read all the files from the working directory: %s\n", err)
		os.Exit(10002)
	}

	fmt.Printf("\rFound %d files from the working directory [ OK ]\t\t\n", len(files))

	/* 2. Asks the LLM to get the file format used extract the page number from the file name */

	fmt.Printf("Requesting the assistant to guess the order of the files... ")

	//	Builds the prompt for the assistant
	var assistantPrompt = strings.Builder{}
	assistantPrompt.WriteString("Below are the files found in the directory. Based on the information found there, sort them according their page number in a JSON array\n")

	for _, file := range files {
		assistantPrompt.WriteString(file.Name() + "\n")
	}

	//	Initializes the assistant
	client := openai.NewClient(
		option.WithAPIKey(openAiApiKey),
	)

	//	Call the assistant to get the ordered list of files
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(assistantPrompt.String()),
		},
		Model: openai.ChatModelGPT5Mini,
	})

	if err != nil {
		fmt.Printf("[ FAILED ]\n")
		fmt.Printf("Error while ordering files: %v\n", err)
		os.Exit(10003)
	}

	if len(chatCompletion.Choices) == 0 || chatCompletion.Choices[0].Message.Content == "" {
		fmt.Printf("[ FAILED ]\n")
		fmt.Printf("Assistant returned an empty response when ordering files\n")
		os.Exit(10004)
	}

	var orderedFiles []string
	if err := json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &orderedFiles); err != nil {
		fmt.Printf("[ FAILED ]\n")
		fmt.Printf("Unable to parse assistant JSON response: %v\n", err)
		os.Exit(10005)
	}

	fmt.Printf("[ OK ]\n")

	if len(orderedFiles) == 0 {
		fmt.Printf("The assistant did not return any ordered files\n")
		os.Exit(10006)
	}

	/* 3. Extract the publication month & year of the first page, assuming it is the cover */

	coverFileName := orderedFiles[0]
	coverPath := filepath.Join(workingDir, coverFileName)

	if _, err := os.Stat(coverPath); err != nil {
		fmt.Printf("Cover file '%s' does not exist or is not accessible: %v\n", coverPath, err)
		os.Exit(10007)
	}

	fmt.Printf("Analyzing cover file: %s... ", coverFileName)

	// Ask the model (via chat) to infer month & year from the cover description.
	// Here we assume the model has context about the PDF content (e.g., from prior processing
	// or file-based tools). If you later wire in actual PDF content, you can augment this prompt.

	datePrompt := strings.Builder{}
	datePrompt.WriteString("You are given the file name of the cover page of a french publication: ")
	datePrompt.WriteString(coverFileName)
	datePrompt.WriteString(". Based on typical naming conventions and any context you can infer, ")
	datePrompt.WriteString("return only the publication month and year in the format `MMMM YYYY` (for example: `Juin 2024`). ")
	datePrompt.WriteString("It usually appears at the bottom of the page, in diagonal on the right corner.")
	datePrompt.WriteString("If you cannot determine it, answer exactly `Unknown`. Do not add any extra explanation.")

	dateCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(datePrompt.String()),
		},
		Model: openai.ChatModelGPT5Mini,
	})

	if err != nil {
		fmt.Printf(" [ FAILED ]\n")
		fmt.Printf("\n")
		fmt.Printf("Unable to analyze the cover: %v\n", err)
		os.Exit(10008)
	}

	if len(dateCompletion.Choices) == 0 || dateCompletion.Choices[0].Message.Content == "" {
		fmt.Printf(" [ FAILED ]\n")
		fmt.Printf("\n")
		fmt.Printf("Model did not return a publication date\n")
		os.Exit(10009)
	}

	publicationDate := strings.TrimSpace(dateCompletion.Choices[0].Message.Content)
	fmt.Printf("Publication date (month & year): %s\n", publicationDate)
}
