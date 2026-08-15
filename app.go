package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/Nemuboshi/PaddleOCR-Json-Editor/internal/document"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	doc           *document.Service
	mu            sync.Mutex
	cancel        context.CancelFunc
	forceClose    bool
	exportItems   []*menu.MenuItem
	pageToolsItem *menu.MenuItem
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	configureLog()
	root := filepath.Join(os.TempDir(), "paddle-json-editor-session")
	doc, err := document.New(root)
	if err != nil {
		panic(err)
	}
	a.doc = doc
	slog.Info("application started")
}
func (a *App) shutdown(context.Context) {
	if a.doc != nil {
		_ = a.doc.Close()
	}
}
func (a *App) beforeClose(context.Context) bool {
	if a.forceClose {
		return false
	}
	if a.doc != nil && a.doc.Status().Changed {
		runtime.EventsEmit(a.ctx, "app:close-requested")
		return true
	}
	return false
}
func (a *App) ConfirmClose()           { a.forceClose = true; runtime.Quit(a.ctx) }
func (a *App) Status() document.Status { return a.doc.Status() }
func (a *App) SelectImportFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Import Paddle JSON", Filters: []runtime.FileFilter{{DisplayName: "JSON files", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return "", err
	}
	return path, nil
}
func (a *App) Import(path string, download bool) (document.ImportResult, error) {
	if path == "" {
		return document.ImportResult{}, errors.New("select a JSON file first")
	}
	ctx, done, err := a.task()
	if err != nil {
		return document.ImportResult{}, err
	}
	defer done()
	result, err := a.doc.Import(ctx, path, download, a.progress("import"))
	if err == nil {
		for _, item := range a.exportItems {
			item.Enable()
		}
		runtime.MenuUpdateApplicationMenu(a.ctx)
	}
	return result, err
}
func (a *App) DownloadAssets() (document.ImportResult, error) {
	ctx, done, err := a.task()
	if err != nil {
		return document.ImportResult{}, err
	}
	defer done()
	return a.doc.DownloadAssets(ctx, a.progress("download"))
}
func (a *App) CancelTask() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.mu.Unlock()
}
func (a *App) Pages(from int) (document.PagesResponse, error)      { return a.doc.Pages(from) }
func (a *App) View(index int) (document.PageView, error)           { return a.doc.View(index) }
func (a *App) Block(page, block int) (document.BlockDetail, error) { return a.doc.Block(page, block) }
func (a *App) UpdateBlock(request document.UpdateBlockRequest) (document.Message, error) {
	if err := a.ensureIdle(); err != nil {
		return document.Message{}, err
	}
	return a.doc.UpdateBlock(request)
}
func (a *App) MoveBlock(page, block int, bbox string) (document.Message, error) {
	if err := a.ensureIdle(); err != nil {
		return document.Message{}, err
	}
	return a.doc.MoveBlock(page, block, bbox)
}
func (a *App) DeletePage(page int) (document.Message, error) {
	if err := a.ensureIdle(); err != nil {
		return document.Message{}, err
	}
	return a.doc.DeletePage(page)
}
func (a *App) DeleteBlock(page, block int) (document.Message, error) {
	if err := a.ensureIdle(); err != nil {
		return document.Message{}, err
	}
	return a.doc.DeleteBlock(page, block)
}
func (a *App) MergeBlocks(page int, blocks, separator string) (document.Message, error) {
	if err := a.ensureIdle(); err != nil {
		return document.Message{}, err
	}
	return a.doc.MergeBlocks(page, blocks, separator)
}
func (a *App) SplitBlock(page, block, at int) (document.Message, error) {
	if err := a.ensureIdle(); err != nil {
		return document.Message{}, err
	}
	return a.doc.SplitBlock(page, block, at)
}
func (a *App) ReorderBlocks(page int, blocks string) (document.Message, error) {
	if err := a.ensureIdle(); err != nil {
		return document.Message{}, err
	}
	return a.doc.ReorderBlocks(page, blocks)
}
func (a *App) Search(request document.SearchRequest) (document.SearchResponse, error) {
	return a.doc.Search(request)
}
func (a *App) PageMarkdown(page int) (string, error) { return a.doc.PageMarkdown(page) }
func (a *App) BlockContent(page, block int) (string, error) {
	return a.doc.BlockContent(page, block)
}
func (a *App) ExportJSON() (string, error) {
	status := a.doc.Status()
	if !status.Loaded {
		return "", errors.New("no document is loaded")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Export JSON", DefaultFilename: "document.json", Filters: []runtime.FileFilter{{DisplayName: "JSON files", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return "", err
	}
	ctx, done, err := a.task()
	if err != nil {
		return "", err
	}
	defer done()
	if err = a.doc.ExportJSON(ctx, path, a.progress("export-json")); err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}
func (a *App) ExportMarkdown(request document.MarkdownRequest) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Export Markdown", DefaultFilename: a.doc.MarkdownFilename(request.Start, request.End), Filters: []runtime.FileFilter{{DisplayName: "Markdown files", Pattern: "*.md"}}})
	if err != nil || path == "" {
		return "", err
	}
	ctx, done, err := a.task()
	if err != nil {
		return "", err
	}
	defer done()
	if err = a.doc.ExportMarkdown(ctx, path, request.Start, request.End, a.progress("export-markdown")); err != nil {
		return "", err
	}
	return filepath.Base(path), nil
}
func (a *App) SetPageToolsVisible(visible bool) {
	a.pageToolsItem.SetChecked(visible)
	runtime.MenuUpdateApplicationMenu(a.ctx)
}
func (a *App) ensureIdle() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return errors.New("another write task is active")
	}
	return nil
}

func (a *App) task() (context.Context, func(), error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return nil, nil, errors.New("another write task is active")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	return ctx, func() { a.mu.Lock(); a.cancel = nil; a.mu.Unlock() }, nil
}
func (a *App) progress(task string) func(int, int, string) {
	return func(done, total int, stage string) {
		runtime.EventsEmit(a.ctx, "task:progress", map[string]any{"task": task, "done": done, "total": total, "stage": stage})
	}
}
func configureLog() {
	config, err := os.UserConfigDir()
	if err != nil {
		return
	}
	dir := filepath.Join(config, "PaddleJsonEditor")
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	path := filepath.Join(dir, "paddle-json-editor.log")
	if info, err := os.Stat(path); err == nil && info.Size() > 2<<20 {
		_ = os.Remove(path)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err == nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(file, nil)))
	}
}

func formatError(err error) string {
	var coded *document.Error
	if errors.As(err, &coded) {
		data, _ := json.Marshal(coded)
		return string(data)
	}
	return fmt.Sprintf(`{"code":"internal","message":%q}`, err.Error())
}
