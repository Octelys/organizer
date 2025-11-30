package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"organizer/internal/configuration"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type AiProxy struct {
	model   shared.ResponsesModel
	client  *openai.Client
	context context.Context
}

func New(
	configurationService *configuration.ConfigurationService,
	context context.Context) (*AiProxy, error) {

	openaiClient := openai.NewClient(
		option.WithAPIKey(configurationService.OpenAiApiKey),
	)

	return &AiProxy{
		client:  &openaiClient,
		context: context,
		model:   openai.ChatModelGPT5Nano,
	}, nil
}

func (aiProxy *AiProxy) SendRequest(assistantPrompt string) (string, error) {

	response, err := aiProxy.client.Responses.New(aiProxy.context, responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: []responses.ResponseInputItemUnionParam{
				{
					OfInputMessage: &responses.ResponseInputItemMessageParam{
						Role: "user",
						Content: responses.ResponseInputMessageContentListParam{
							{
								OfInputText: &responses.ResponseInputTextParam{
									Text: assistantPrompt,
								},
							},
						},
					},
				},
			},
		},
		Model: aiProxy.model,
	})

	if err != nil {
		return "", fmt.Errorf("unable to process the prompt: %v", err)
	}

	outputText := response.OutputText()

	return outputText, nil
}

func (aiProxy *AiProxy) SendRequestWithImage(assistantPrompt string, reader io.Reader) (string, error) {

	var imageBase64StringBuilder strings.Builder
	imageBase64StringBuilder.WriteString("data:image/jpeg;base64,")

	encoder := base64.NewEncoder(base64.StdEncoding, &imageBase64StringBuilder)

	if _, err := io.Copy(encoder, reader); err != nil {
		return "", fmt.Errorf("unable to encode the image: %v", err)
	}

	response, err := aiProxy.client.Responses.New(aiProxy.context, responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: []responses.ResponseInputItemUnionParam{
				{
					OfInputMessage: &responses.ResponseInputItemMessageParam{
						Role: "user",
						Content: responses.ResponseInputMessageContentListParam{
							{
								OfInputText: &responses.ResponseInputTextParam{
									Text: assistantPrompt,
								},
							},
							{
								OfInputImage: &responses.ResponseInputImageParam{
									Type:     "input_image",
									ImageURL: param.NewOpt(imageBase64StringBuilder.String()),
								},
							},
						},
					},
				},
			},
		},
		Model: aiProxy.model,
	})

	if err != nil {
		return "", fmt.Errorf("unable to process the prompt: %v", err)
	}

	outputText := response.OutputText()

	return outputText, nil
}
