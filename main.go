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
)

type Page struct {
	File   string `json:"file"`
	Number uint8  `json:"number"`
}

type Publication struct {
	Title  string  `json:"title"`
	Number uint8   `json:"number"`
	Month  []uint8 `json:"months"`
	Year   uint16  `json:"year"`
}

func monthNumbersToNames(nums []uint8) []string {
	months := []string{
		"Janvier", "Février", "Mars", "Avril", "Mai", "Juin",
		"Juillet", "Août", "Septembre", "Octobre", "Novembre", "Décembre",
	}

	names := make([]string, 0, len(nums))
	for _, n := range nums {
		if n >= 1 && n <= 12 {
			names = append(names, months[n-1])
		}
	}
	return names
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
		assistantPrompt.WriteString("Below are the files found in the directory. Based on the information found there, sort them according to their page number in a JSON array (for example: [{\"file\": \"page_01.pdf\", \"number\": 1 }, {\"file\": \"page_02.pdf\", \"number\": 2 }]). If the 1st file starts at the number 0, make sure you start counting at 1. Return only valid JSON and no extra text.\n")

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

		var orderedPages []Page
		if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &orderedPages); err != nil {
			fmt.Printf("[ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tUnable to parse assistant JSON response: %v\n", err)
			os.Exit(10005)
		}

		if len(orderedPages) == 0 {
			fmt.Printf("[ FAILED ]\n")
			fmt.Printf("\n")
			fmt.Printf("\tThe assistant did not return any ordered files\n")
			os.Exit(10006)
		}

		fmt.Printf("[ OK ]\n")

		//	----------------------------------------------------------------------------------------------------------------
		// 	3. Extract the publication month(s) & year of the first page, assuming it is the cover
		//	----------------------------------------------------------------------------------------------------------------
		coverFileName := orderedPages[0].File

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

		newPublicationFolder := filepath.Join(workingDir, fmt.Sprintf("__%s", publication.Title))

		if _, err := os.Stat(newPublicationFolder); os.IsNotExist(err) {
			err := os.Mkdir(newPublicationFolder, os.ModePerm)
			if err != nil {
				fmt.Printf("Unable to create folder %s: %v\n", newPublicationFolder, err)
				os.Exit(10010)
			}
		}

		knownMonths := monthNumbersToNames(publication.Month)
		publicationMonths := strings.Join(knownMonths, " - ")
		publicationDate := fmt.Sprintf("%s %d", publicationMonths, publication.Year)

		newPublicationFolderNumber := filepath.Join(newPublicationFolder, fmt.Sprintf("Numéro %02d | %s", publication.Number, publicationDate))

		if _, err := os.Stat(newPublicationFolderNumber); os.IsNotExist(err) {
			err := os.Mkdir(newPublicationFolderNumber, os.ModePerm)
			if err != nil {
				fmt.Printf("Unable to create folder %s: %v\n", newPublicationFolderNumber, err)
				os.Exit(10010)
			}
		}

		for _, orderedPage := range orderedPages {
			srcPath := filepath.Join(publicationFolder, orderedPage.File)

			pageFileName := fmt.Sprintf("%03d%s", orderedPage.Number, strings.ToLower(filepath.Ext(orderedPage.File)))
			dstPath := filepath.Join(newPublicationFolderNumber, pageFileName)

			src, err := os.Open(srcPath)
			if err != nil {
				fmt.Printf("Unable to open source file %s: %v\n", srcPath, err)
				os.Exit(10010)
			}
			defer src.Close()

			dst, err := os.Create(dstPath)
			if err != nil {
				fmt.Printf("Unable to create destination file %s: %v\n", dstPath, err)
				os.Exit(10010)
			}
			defer dst.Close()

			if _, err := io.Copy(dst, src); err != nil {
				fmt.Printf("Unable to copy the file from %s to %s: %v\n", srcPath, dstPath, err)
				os.Exit(10010)
			}

			fmt.Printf("\tFile %s copied\n", dstPath)
		}
	}
}
