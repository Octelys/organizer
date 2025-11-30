package copier

import (
	"context"
	"fmt"
	"io"
	"organizer/internal/abstractions/entities"
	"organizer/internal/abstractions/interfaces"
	"organizer/internal/configuration"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	Prefix = "test-"
)

type CopierService struct {
	workingDirectory string
	magazinesChannel interfaces.MagazinesChannel
	context          context.Context
	waitGroup        *sync.WaitGroup
}

func New(
	configurationService *configuration.ConfigurationService,
	magazinesChannel interfaces.MagazinesChannel,
	context context.Context,
	waitGroup *sync.WaitGroup) *CopierService {

	service := CopierService{
		workingDirectory: configurationService.WorkingDirectory,
		magazinesChannel: magazinesChannel,
		context:          context,
		waitGroup:        waitGroup,
	}

	return &service
}

func (c *CopierService) Run() {

	c.waitGroup.Add(1)

	go func() {

		fmt.Println("Copier service started.")

		defer c.waitGroup.Done()

		err := c.monitor()

		if err != nil {

		}
	}()
}

func (c *CopierService) monitor() error {

	for magazine := range c.magazinesChannel.Magazines() {

		err := c.renameFiles(magazine)

		if err != nil {
			fmt.Printf("Unable to transfer %s %s: %v\n", magazine.Metadata.Title, magazine.Metadata.Number, err)
			return err
		}

		fmt.Printf("Magazine %s %d transferred\n", magazine.Metadata.Title, magazine.Metadata.Number)
	}

	fmt.Println("Copier service stopped.")

	return nil
}

func (c *CopierService) renameFiles(magazine entities.Magazine) error {

	fmt.Printf("Publication %s #%d\n", magazine.Metadata.Title, magazine.Metadata.Number)

	newPublicationFolder := filepath.Join(c.workingDirectory, fmt.Sprintf("%s%s", Prefix, magazine.Metadata.Title))

	if _, err := os.Stat(newPublicationFolder); os.IsNotExist(err) {
		err := os.Mkdir(newPublicationFolder, os.ModePerm)
		if err != nil {
			err := fmt.Errorf("unable to create folder %s: %v", newPublicationFolder, err)
			return err
		}
	}

	knownMonths := toNames(magazine.Metadata.Month)
	publicationMonths := strings.Join(knownMonths, " - ")
	publicationDate := fmt.Sprintf("%s %d", publicationMonths, magazine.Metadata.Year)

	newPublicationFolderNumber := filepath.Join(newPublicationFolder, fmt.Sprintf("Numéro %02d | %s", magazine.Metadata.Number, publicationDate))

	if _, err := os.Stat(newPublicationFolderNumber); os.IsNotExist(err) {
		err := os.Mkdir(newPublicationFolderNumber, os.ModePerm)
		if err != nil {
			err := fmt.Errorf("unable to create folder %s: %v", newPublicationFolderNumber, err)
			return err
		}
	}

	for _, magazinePage := range magazine.Pages {
		srcPath := filepath.Join(magazine.Folder, magazinePage.File)

		pageFileName := fmt.Sprintf("%03d%s", magazinePage.Number, strings.ToLower(filepath.Ext(magazinePage.File)))
		dstPath := filepath.Join(newPublicationFolderNumber, pageFileName)

		src, err := os.Open(srcPath)
		if err != nil {
			err := fmt.Errorf("unable to open source file %s: %v", srcPath, err)
			return err
		}
		defer src.Close()

		dst, err := os.Create(dstPath)
		if err != nil {
			err := fmt.Errorf("unable to create destination file %s: %v\n", dstPath, err)
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			err := fmt.Errorf("unable to copy the file from %s to %s: %v\n", srcPath, dstPath, err)
			return err
		}

		fmt.Printf("File %s copied\n", dstPath)
	}

	return nil
}

func toNames(nums []uint8) []string {
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
