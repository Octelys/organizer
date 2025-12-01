package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"organizer/internal/abstractions/entities"
	"organizer/internal/abstractions/interfaces"
	"organizer/internal/ai"
	"organizer/internal/audit"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

const (
	CoverPageAssistantPrompt      = "You are given a JPG file containing an image of a cover scanner of a French publication. Based on typical naming conventions and any context you can infer, return only the title, publication number and publication month and year in the JSON format `{ \"title\": string, \"months\": [number,], \"year\": number, \"number\": number }`. If you cannot determine it, answer exactly `Unknown`. Do not add any extra explanation."
	TableOfContentAssistantPrompt = "This page should be a Summary page of a french magazine. Give me each section name with the page numbers. Returns the structure in the following Json format: {\"error\": string, \"entries\": [{\"title\": string, \"pageNumbers\": [number]}]. Order the result by the Numbers from the lower number to the highest. Only keep the entries that have the words 'Test(s)', 'Sélection(s)' (case insensitive)"
	GameTestedAssistantPrompt     = "This page a test of a game. Found the name of the game and the console is on. If it is on the page, return the score given to the game. The result should be return in the following Json format: {\"title\": string, \"console\": string, \"score\": number, \"outOf\": number}."
)

type AnalyzerService struct {
	aiProxy              *ai.AiProxy
	magazinePagesChannel interfaces.MagazinePagesChannel
	magazinesChannel     chan entities.Magazine
	auditService         *audit.AuditService
	context              context.Context
	waitGroup            *sync.WaitGroup
}

func New(
	aiProxy *ai.AiProxy,
	magazinePagesChannel interfaces.MagazinePagesChannel,
	auditService *audit.AuditService,
	context context.Context,
	waitGroup *sync.WaitGroup) *AnalyzerService {

	service := AnalyzerService{
		aiProxy:              aiProxy,
		auditService:         auditService,
		magazinePagesChannel: magazinePagesChannel,
		magazinesChannel:     make(chan entities.Magazine),
		context:              context,
		waitGroup:            waitGroup,
	}

	return &service
}

func (a *AnalyzerService) Run() {

	a.waitGroup.Add(1)

	go func() {

		a.auditService.Log(entities.Audit{Severity: entities.Information, Timestamp: time.Now(), Text: fmt.Sprintf("Analyzer service started.")})

		defer a.waitGroup.Done()

		err := a.monitor()

		if err != nil {
			a.auditService.Log(entities.Audit{Severity: entities.Information, Timestamp: time.Now(), Text: fmt.Sprintf("An error occurred in the analyzer service: %v", err)})
		}
	}()
}

func (a *AnalyzerService) monitor() error {

	for magazinePages := range a.magazinePagesChannel.Pages() {
		a.analyzePages(magazinePages)
		a.analyzeTableOfContent(magazinePages)
	}

	close(a.magazinesChannel)

	a.auditService.Log(entities.Audit{Severity: entities.Information, Timestamp: time.Now(), Text: fmt.Sprintf("Analyzer service stopped.")})

	return nil
}

func (a *AnalyzerService) analyzePages(magazinePages entities.MagazinePages) {

	if magazinePages.Pages == nil || len(magazinePages.Pages) == 0 {
		a.auditService.Log(entities.Audit{
			Severity:  entities.Information,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("No pages to analyze.")})
		return
	}

	coverFileName := magazinePages.Pages[0].File

	a.auditService.Log(entities.Audit{
		Severity:  entities.Information,
		Timestamp: time.Now(),
		Text:      fmt.Sprintf("Analyzing cover file '%s'\n", coverFileName)})

	coverPath := filepath.Join(magazinePages.Folder, coverFileName)

	if _, err := os.Stat(coverPath); err != nil {
		a.auditService.Log(entities.Audit{
			Severity:  entities.Error,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("Cover file '%s' does not exist or is not accessible: %v", coverPath, err)})
		return
	}

	reader, err := os.Open(coverPath)

	if err != nil {
		a.auditService.Log(entities.Audit{
			Severity:  entities.Error,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("Cover file '%s' does not exist or is not accessible: %v", coverPath, err)})
		return
	}

	defer reader.Close()

	response, err := a.aiProxy.SendRequestWithImage(CoverPageAssistantPrompt, reader)

	if err != nil {
		a.auditService.Log(entities.Audit{
			Severity:  entities.Error,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("An error occurred trying to analyze the cover file '%s': %v", coverPath, err)})
		return
	}

	if response == "" || response == "Unknown" {
		a.auditService.Log(entities.Audit{
			Severity:  entities.Error,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("Unable to retieve the metadata of the cover file '%s'", coverPath)})
		return
	}

	var metadata entities.MagazineMetadata
	if err := json.Unmarshal([]byte(response), &metadata); err != nil {
		a.auditService.Log(entities.Audit{
			Severity:  entities.Error,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("Unable to decode the magazine metadata of cover file '%s': %v", coverPath, err)})
		a.auditService.Log(entities.Audit{
			Severity:  entities.Debug,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("Received: %s\n", response)})
		return
	}

	a.auditService.Log(entities.Audit{
		Severity:  entities.Information,
		Timestamp: time.Now(),
		Text:      fmt.Sprintf("Analysis done: found publication title is '%s' and its number is '%d'", metadata.Title, metadata.Number)})

	a.magazinesChannel <- entities.Magazine{
		Metadata: metadata,
		Pages:    magazinePages.Pages,
		Folder:   magazinePages.Folder,
	}
}

func (a *AnalyzerService) analyzeTableOfContent(magazinePages entities.MagazinePages) {

	if magazinePages.Pages == nil || len(magazinePages.Pages) == 0 {
		a.auditService.Log(entities.Audit{
			Severity:  entities.Information,
			Timestamp: time.Now(),
			Text:      fmt.Sprintf("No pages to analyze.")})
		return
	}

	var tableContent entities.TableContent

	for _, page := range magazinePages.Pages[1:] {

		pageFile := filepath.Join(magazinePages.Folder, page.File)

		if _, err := os.Stat(pageFile); err != nil {
			a.auditService.Log(entities.Audit{
				Severity:  entities.Error,
				Timestamp: time.Now(),
				Text:      fmt.Sprintf("File '%s' does not exist or is not accessible: %v", pageFile, err)})
			return
		}

		reader, err := os.Open(pageFile)

		if err != nil {
			a.auditService.Log(entities.Audit{
				Severity:  entities.Error,
				Timestamp: time.Now(),
				Text:      fmt.Sprintf("File '%s' does not exist or is not accessible: %v", pageFile, err)})
			return
		}

		defer reader.Close()

		response, err := a.aiProxy.SendRequestWithImage(TableOfContentAssistantPrompt, reader)

		if err != nil {
			a.auditService.Log(entities.Audit{
				Severity:  entities.Error,
				Timestamp: time.Now(),
				Text:      fmt.Sprintf("An error occurred trying to analyze the file '%s': %v", pageFile, err)})
			return
		}

		if response == "" {
			a.auditService.Log(entities.Audit{
				Severity:  entities.Error,
				Timestamp: time.Now(),
				Text:      fmt.Sprintf("Unable to retieve the metadata of the file '%s'", pageFile)})
			return
		}

		if err := json.Unmarshal([]byte(response), &tableContent); err != nil {
			a.auditService.Log(entities.Audit{
				Severity:  entities.Error,
				Timestamp: time.Now(),
				Text:      fmt.Sprintf("Unable to decode the table of content of file '%s': %v", pageFile, err)})
			a.auditService.Log(entities.Audit{
				Severity:  entities.Debug,
				Timestamp: time.Now(),
				Text:      fmt.Sprintf("Received: %s\n", response)})
			continue
		}

		if tableContent.Error != "" || len(tableContent.Entries) == 0 {
			continue
		} else {
			break
		}
	}

	var gamesTested []entities.Game

	for _, entry := range tableContent.Entries {
		for _, pageNumber := range entry.PageNumbers {

			idx := slices.IndexFunc(magazinePages.Pages, func(p entities.MagazinePage) bool {
				return p.Number == pageNumber
			})

			pageFile := filepath.Join(magazinePages.Folder, magazinePages.Pages[idx].File)

			if _, err := os.Stat(pageFile); err != nil {
				a.auditService.Log(entities.Audit{
					Severity:  entities.Error,
					Timestamp: time.Now(),
					Text:      fmt.Sprintf("File '%s' does not exist or is not accessible: %v", pageFile, err)})
				return
			}

			reader, err := os.Open(pageFile)

			if err != nil {
				a.auditService.Log(entities.Audit{
					Severity:  entities.Error,
					Timestamp: time.Now(),
					Text:      fmt.Sprintf("File '%s' does not exist or is not accessible: %v", pageFile, err)})
				return
			}

			defer reader.Close()

			response, err := a.aiProxy.SendRequestWithImage(GameTestedAssistantPrompt, reader)

			if err != nil {
				a.auditService.Log(entities.Audit{
					Severity:  entities.Error,
					Timestamp: time.Now(),
					Text:      fmt.Sprintf("An error occurred trying to analyze the file '%s': %v", pageFile, err)})
				return
			}

			if response == "" {
				a.auditService.Log(entities.Audit{
					Severity:  entities.Error,
					Timestamp: time.Now(),
					Text:      fmt.Sprintf("Unable to retrieve the game tested from the file '%s'", pageFile)})
				return
			}

			var gameTested entities.Game
			if err := json.Unmarshal([]byte(response), &gameTested); err != nil {
				a.auditService.Log(entities.Audit{
					Severity:  entities.Error,
					Timestamp: time.Now(),
					Text:      fmt.Sprintf("Unable to decode the table of content of file '%s': %v", pageFile, err)})
				a.auditService.Log(entities.Audit{
					Severity:  entities.Debug,
					Timestamp: time.Now(),
					Text:      fmt.Sprintf("Received: %s\n", response)})
				continue
			}

			gamesTested = append(gamesTested, gameTested)
		}
	}
}

func (a *AnalyzerService) Magazines() <-chan entities.Magazine {
	return a.magazinesChannel
}
