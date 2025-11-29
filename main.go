package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	//"github.com/openai/openai-go/v3/responses"
)

type Publication struct {
	Title  string  `json:"title"`
	Number uint8   `json:"number"`
	Month  []uint8 `json:"months"`
	Year   uint16  `json:"year"`
}

func main() {
	ctx := context.Background()

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

	//	----------------------------------------------------------------------------------------------------------------
	// 	0. Read all the folders from the working directory
	//	----------------------------------------------------------------------------------------------------------------

	fmt.Printf("Reading the folders from the working directory %s... ", workingDir)

	folders, err := os.ReadDir(workingDir)

	if err != nil {
		fmt.Printf(" [ FAILED ]\n")
		fmt.Printf("\n")
		fmt.Printf("\tUnable to read all the folder from the working directory: %s\n", err)
		os.Exit(10002)
	}

	fmt.Println("[ OK ]")

	for _, folder := range folders {

		if !folder.IsDir() {
			continue
		}

		publicationFolder := filepath.Join(workingDir, folder.Name())

		//	----------------------------------------------------------------------------------------------------------------
		// 	1. Read all the file names in the directory
		//	----------------------------------------------------------------------------------------------------------------
		fmt.Printf("Reading the name of the files from the directory %s... ", publicationFolder)

		files, err := os.ReadDir(publicationFolder)

		if err != nil {
			fmt.Printf(" [ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tUnable to read all the files from the directory: %s\n", err)
			os.Exit(10002)
		}

		fmt.Println("[ OK ]")

		fmt.Printf("\r\tFound %d files from the directory\t\t\n", len(files))

		//	----------------------------------------------------------------------------------------------------------------
		// 	2. Ask the LLM to infer file order from file names
		//	----------------------------------------------------------------------------------------------------------------
		fmt.Printf("Requesting the assistant to guess the order of the files... ")

		var assistantPrompt strings.Builder
		assistantPrompt.WriteString("Below are the files found in the directory. Based on the information found there, sort them according to their page number in a JSON array (for example: [\"page_01.pdf\", \"page_02.pdf\"]). Return only valid JSON and no extra text.\n")

		for _, file := range files {
			assistantPrompt.WriteString(file.Name())
			assistantPrompt.WriteString("\n")
		}

		client := openai.NewClient(
			option.WithAPIKey(openAiApiKey),
		)

		chatResp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT5Mini,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(assistantPrompt.String()),
			},
		})

		if err != nil {
			fmt.Printf("[ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tError while ordering files: %v\n", err)
			os.Exit(10003)
		}

		if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
			fmt.Printf("[ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tAssistant returned an empty response when ordering files\n")
			os.Exit(10004)
		}

		var orderedFiles []string
		if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &orderedFiles); err != nil {
			fmt.Printf("[ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tUnable to parse assistant JSON response: %v\n", err)
			os.Exit(10005)
		}

		if len(orderedFiles) == 0 {
			fmt.Printf("[ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tThe assistant did not return any ordered files\n")
			os.Exit(10006)
		}

		fmt.Printf("[ OK ]\n")

		//	----------------------------------------------------------------------------------------------------------------
		// 	3. Extract the publication month(s) & year of the first page, assuming it is the cover
		//	----------------------------------------------------------------------------------------------------------------
		coverFileName := orderedFiles[0]

		fmt.Printf("Analyzing cover file '%s'... ", coverFileName)

		coverPath := filepath.Join(publicationFolder, coverFileName)

		if _, err := os.Stat(coverPath); err != nil {
			fmt.Printf("[ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tCover file '%s' does not exist or is not accessible: %v\n", coverPath, err)
			os.Exit(10007)
		}

		reader, err := os.Open(coverPath)

		if err != nil {
			fmt.Printf(" [ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tCover file '%s' does not exist or is not accessible: %v\n", coverPath, err)
			os.Exit(10007)
		}

		defer reader.Close()

		assistantPrompt.Reset()
		assistantPrompt.WriteString("You are given a JPG file containing an image of a cover page of a French publication: ")
		assistantPrompt.WriteString(coverFileName)
		assistantPrompt.WriteString(". Based on typical naming conventions and any context you can infer, ")
		assistantPrompt.WriteString("return only the title, publication number and publication month and year in the JSON format `{ \"title\": string, \"months\": [number,], \"year\": number, \"number\": number }`")
		assistantPrompt.WriteString("If you cannot determine it, answer exactly `Unknown`. Do not add any extra explanation.")

		fileContent, _ := io.ReadAll(reader)
		base64FileContent := base64.StdEncoding.EncodeToString(fileContent)

		publicationDateResponse, err := client.Responses.New(ctx, responses.ResponseNewParams{
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: []responses.ResponseInputItemUnionParam{
					{
						OfInputMessage: &responses.ResponseInputItemMessageParam{
							Role: "user",
							Content: responses.ResponseInputMessageContentListParam{
								{
									OfInputText: &responses.ResponseInputTextParam{
										Text: assistantPrompt.String(),
									},
								},
								{
									OfInputImage: &responses.ResponseInputImageParam{
										Type:     "input_image",
										ImageURL: param.NewOpt("data:image/jpeg;base64," + base64FileContent),
									},
								},
							},
						},
					},
				},
			},
			Model: openai.ChatModelGPT5Mini,
		})

		if err != nil {
			fmt.Printf(" [ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("Unable to analyze the cover: %v\n", err)
			os.Exit(10008)
		}

		if publicationDateResponse == nil || publicationDateResponse.OutputText() == "" {
			fmt.Printf(" [ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("Model did not return a publication date\n")
			os.Exit(10009)
		}

		fmt.Printf(" [ OK ]\n")
		fmt.Printf("\n")

		var publication Publication
		if err := json.Unmarshal([]byte(publicationDateResponse.OutputText()), &publication); err != nil {
			fmt.Println("decode error:", err)
			return
		}

		fmt.Printf("Publication %s #%d\n", publication.Title, publication.Number)
	}
}
