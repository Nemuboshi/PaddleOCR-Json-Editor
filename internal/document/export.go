package document

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) ExportJSON(ctx context.Context, target string, progress func(int, int, string)) error {
	s.mu.RLock()
	if len(s.pages) == 0 {
		s.mu.RUnlock()
		return coded("not_loaded", "Import a Paddle JSON file first")
	}
	pages := append([]pageRecord(nil), s.pages...)
	assets := cloneAssets(s.assets)
	revision := s.revision
	s.mu.RUnlock()
	if filepath.Ext(target) == "" {
		target += ".json"
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".export-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	writer := bufio.NewWriter(temp)
	writer.WriteByte('[')
	for i, p := range pages {
		select {
		case <-ctx.Done():
			temp.Close()
			return ctx.Err()
		default:
		}
		if i > 0 {
			writer.WriteByte(',')
		}
		data, err := json.Marshal(p.Raw)
		if err != nil {
			temp.Close()
			return err
		}
		if _, err = writer.Write(data); err != nil {
			temp.Close()
			return err
		}
		if progress != nil {
			progress(i+1, len(pages), "Exporting JSON")
		}
	}
	writer.WriteString("]\n")
	if err = writer.Flush(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = copyAssets(assets, filepath.Dir(target)); err != nil {
		return err
	}
	if err = replaceFile(name, target); err != nil {
		return err
	}
	s.mu.Lock()
	if s.revision == revision {
		s.changed = false
	}
	s.mu.Unlock()
	return nil
}
func (s *Service) PageMarkdown(pageIndex int) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	page, err := s.requirePage(pageIndex)
	if err != nil {
		return "", err
	}
	return pageMarkdownContent(page, pageIndex)
}

func (s *Service) BlockContent(pageIndex, blockID int) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, block, err := s.requireBlock(pageIndex, blockID)
	if err != nil {
		return "", err
	}
	return stringValue(block["block_content"]), nil
}

func (s *Service) ExportMarkdown(ctx context.Context, target string, start, end int, progress func(int, int, string)) error {
	s.mu.RLock()
	if start < 1 || end < start || end > len(s.pages) {
		s.mu.RUnlock()
		return coded("invalid_range", "The page range is invalid")
	}
	pages := append([]pageRecord(nil), s.pages...)
	assets := markdownAssets(pages[start-1:end], s.assets)
	s.mu.RUnlock()
	if filepath.Ext(target) == "" {
		target += ".md"
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".markdown-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	writer := bufio.NewWriter(temp)
	for i := start - 1; i < end; i++ {
		select {
		case <-ctx.Done():
			temp.Close()
			return ctx.Err()
		default:
		}
		content, contentErr := pageMarkdownContent(&pages[i], i)
		if contentErr != nil {
			temp.Close()
			return contentErr
		}
		if i > start-1 {
			writer.WriteString("\n\n---\n\n")
		}
		writer.WriteString(content)
		if progress != nil {
			progress(i-start+2, end-start+1, "Exporting Markdown")
		}
	}
	writer.WriteByte('\n')
	if err = writer.Flush(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = copyAssets(assets, filepath.Dir(target)); err != nil {
		return err
	}
	return replaceFile(name, target)
}

func cloneAssets(source map[string]assetRecord) map[string]assetRecord { return maps.Clone(source) }

func markdownAssets(pages []pageRecord, source map[string]assetRecord) map[string]assetRecord {
	result := map[string]assetRecord{}
	for _, page := range pages {
		ignored := ignoredLabels(page.Raw)
		contents := ""
		for _, block := range page.Blocks {
			if !ignored[stringValue(block["block_label"])] {
				contents += markdownBlockContent(block)
			}
		}
		contents = strings.ReplaceAll(contents, "\\", "/")
		for id, asset := range source {
			if strings.Contains(contents, "assets/"+id) {
				result[id] = asset
			}
		}
		markdown, _ := page.Raw["markdown"].(map[string]any)
		images, _ := markdown["images"].(map[string]any)
		for key, value := range images {
			if !strings.Contains(contents, strings.ReplaceAll(key, "\\", "/")) {
				continue
			}
			id := strings.TrimPrefix(strings.ReplaceAll(stringValue(value), "\\", "/"), "assets/")
			if asset, ok := source[id]; ok {
				result[id] = asset
			}
		}
	}
	return result
}

func copyAssets(assets map[string]assetRecord, destination string) error {
	if len(assets) == 0 {
		return nil
	}
	dir := filepath.Join(destination, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for id, asset := range assets {
		target := filepath.Join(dir, id)
		temp, err := os.CreateTemp(dir, ".asset-")
		if err != nil {
			return err
		}
		source, err := os.Open(asset.Path)
		if err != nil {
			temp.Close()
			return err
		}
		_, err = io.Copy(temp, source)
		source.Close()
		if err == nil {
			err = temp.Sync()
		}
		if closeErr := temp.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(temp.Name())
			return err
		}
		if err = replaceFile(temp.Name(), target); err != nil {
			return err
		}
	}
	return nil
}
func formatBlockContent(raw map[string]any) bool {
	pr, _ := raw["prunedResult"].(map[string]any)
	settings, _ := pr["model_settings"].(map[string]any)
	value, _ := settings["format_block_content"].(bool)
	return value
}

func pageMarkdownContent(page *pageRecord, pageIndex int) (string, error) {
	if !formatBlockContent(page.Raw) {
		return "", coded("markdown_unavailable", fmt.Sprintf("Page %d does not contain formatted block content", pageIndex+1))
	}
	ignored := ignoredLabels(page.Raw)
	contents := []string{}
	for _, block := range page.Blocks {
		if !ignored[stringValue(block["block_label"])] {
			if content := markdownBlockContent(block); content != "" {
				contents = append(contents, content)
			}
		}
	}
	return strings.Join(contents, "\n\n"), nil
}

func markdownBlockContent(block map[string]any) string {
	content := stringValue(block["block_content"])
	if content == "" {
		return ""
	}
	switch stringValue(block["block_label"]) {
	case "doc_title":
		return markdownHeading(content, 1)
	case "paragraph_title":
		if level, ok := block["title_level"]; ok {
			return markdownHeading(content, intValue(level)+1)
		}
		return markdownTitle(content)
	case "abstract_title", "reference_title", "content_title":
		return markdownTitle(content)
	case "table_title", "figure_title", "chart_title":
		return centerMarkdown(content)
	case "formula", "display_formula", "inline_formula":
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "$") || strings.HasPrefix(trimmed, "\\(") || strings.Contains(trimmed, "<") {
			return content
		}
		return "$$" + content + "$$"
	case "table":
		content = strings.ReplaceAll(content, "<table>", "<table border=1 style='margin: auto; word-wrap: break-word;'>")
		content = strings.ReplaceAll(content, "<th>", "<th style='text-align: center; word-wrap: break-word;'>")
		return strings.ReplaceAll(content, "<td>", "<td style='text-align: center; word-wrap: break-word;'>")
	case "text", "ocr", "vertical_text", "reference_content", "vision_footnote":
		return strings.ReplaceAll(strings.ReplaceAll(content, "\n\n", "\n"), "\n", "\n\n")
	case "content":
		return strings.ReplaceAll(strings.ReplaceAll(content, "-\n", "  \n"), "\n", "  \n")
	default:
		return content
	}
}

func markdownHeading(content string, level int) string {
	if strings.HasPrefix(strings.TrimSpace(content), "#") {
		return content
	}
	return strings.Repeat("#", max(level, 1)) + " " + strings.ReplaceAll(strings.ReplaceAll(content, "-\n", ""), "\n", " ")
}

func markdownTitle(content string) string {
	trimmed := strings.TrimRight(content, ".")
	level := 2
	if strings.Contains(trimmed, ".") {
		level += strings.Count(trimmed, ".")
	}
	return markdownHeading(trimmed, level)
}

func centerMarkdown(content string) string {
	if strings.Contains(content, "text-align: center") {
		return content
	}
	content = strings.ReplaceAll(strings.ReplaceAll(content, "-\n", ""), "\n", " ")
	return `<div style="text-align: center;">` + content + "</div>\n"
}

func ignoredLabels(raw map[string]any) map[string]bool {
	result := map[string]bool{}
	pr, _ := raw["prunedResult"].(map[string]any)
	settings, _ := pr["model_settings"].(map[string]any)
	values, _ := settings["markdown_ignore_labels"].([]any)
	for _, v := range values {
		result[stringValue(v)] = true
	}
	return result
}
func (s *Service) Asset(id string) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return "", "", errors.New("invalid asset ID")
	}
	asset, ok := s.assets[id]
	if !ok {
		return "", "", os.ErrNotExist
	}
	return asset.Path, asset.MIME, nil
}
