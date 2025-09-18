// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost-plugin-ai/mcpserver/types"
	"github.com/mattermost/mattermost/server/public/model"
)

// fetchFileDataForStdio fetches file data from a file path or URL (STDIO transport only)
func fetchFileDataForStdio(filespec string, transportMode types.TransportMode) ([]byte, error) {
	if filespec == "" {
		return nil, fmt.Errorf("empty filespec provided")
	}

	// URLs are only allowed for STDIO transport
	if transportMode != types.TransportModeStdio {
		return nil, fmt.Errorf("URL access not supported over network transports, only STDIO transport allows URL access")
	}

	// Check if it's a URL
	if isURL(filespec) {
		resp, err := http.Get(filespec) // #nosec G107 - filespec is validated to be URL
		if err != nil {
			return nil, fmt.Errorf("failed to fetch file from URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch file: HTTP %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read file data: %w", err)
		}

		return data, nil
	}

	cleanPath := filepath.Clean(filespec)
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// isURL checks if a string is a URL
func isURL(filespec string) bool {
	return strings.HasPrefix(filespec, "http://") || strings.HasPrefix(filespec, "https://")
}

// extractFileNameForStdio extracts the filename from a filespec (URL or file path, STDIO transport only)
func extractFileNameForStdio(filespec string, transportMode types.TransportMode) string {
	if filespec == "" {
		return ""
	}

	if transportMode != types.TransportModeStdio {
		return "url-access-denied"
	}

	// Handle URLs - only allowed for STDIO transport
	if isURL(filespec) {
		parsedURL, err := url.Parse(filespec)
		if err != nil {
			// Fallback to simple string splitting if URL parsing fails
			parts := strings.Split(filespec, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
			return "unknown"
		}

		filename := filepath.Base(parsedURL.Path)
		if filename == "" || filename == "." || filename == "/" {
			return "unknown"
		}
		return filename
	}

	// For local file paths, extract the base name
	return filepath.Base(filespec)
}

// isValidImageFile checks if the file extension is a supported image format
func isValidImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpeg", ".jpg", ".png", ".gif"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// uploadFilesForStdio uploads multiple files from URLs or file paths (STDIO transport only) and returns their file IDs
func uploadFilesForStdio(ctx context.Context, client *model.Client4, channelID string, filespecs []string, transportMode types.TransportMode) ([]string, error) {
	var fileIDs []string

	// Early validation - only STDIO transport can upload files
	if transportMode != types.TransportModeStdio {
		return nil, fmt.Errorf("file uploads not supported over network transports, only STDIO transport allows file operations")
	}

	for _, filespec := range filespecs {
		if filespec == "" {
			continue
		}

		fileData, err := fetchFileDataForStdio(filespec, transportMode)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch file %s: %w", filespec, err)
		}

		fileName := extractFileNameForStdio(filespec, transportMode)
		if fileName == "" || fileName == "url-access-denied" || fileName == "file-access-denied" {
			fileName = "attachment"
		}

		fileUploadResponse, _, err := client.UploadFileAsRequestBody(ctx, fileData, channelID, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file %s: %w", filespec, err)
		}

		if len(fileUploadResponse.FileInfos) > 0 {
			fileIDs = append(fileIDs, fileUploadResponse.FileInfos[0].Id)
		}
	}

	return fileIDs, nil
}

// uploadFilesAndUrlsForStdio uploads files from URLs or file paths (STDIO transport only) and returns file IDs and status message
func uploadFilesAndUrlsForStdio(ctx context.Context, client *model.Client4, channelID string, attachments []string, transportMode types.TransportMode) ([]string, string) {
	var fileIDs []string
	var attachmentMessage string

	if len(attachments) > 0 {
		if transportMode != types.TransportModeStdio {
			attachmentMessage = " (file attachments not supported over network transports)"
			return nil, attachmentMessage
		}

		uploadedFileIDs, uploadErr := uploadFilesForStdio(ctx, client, channelID, attachments, transportMode)
		if uploadErr != nil {
			attachmentMessage = fmt.Sprintf(" (file upload failed: %v)", uploadErr)
		} else {
			fileIDs = uploadedFileIDs
			attachmentMessage = fmt.Sprintf(" (uploaded %d files)", len(fileIDs))
		}
	}

	return fileIDs, attachmentMessage
}
