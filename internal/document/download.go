package document

import (
	"context"
	"os"
	"path/filepath"
)

func (s *Service) DownloadAssets(ctx context.Context, progress func(int, int, string)) (ImportResult, error) {
	s.mu.RLock()
	if len(s.pages) == 0 {
		s.mu.RUnlock()
		return ImportResult{}, coded("not_loaded", "Import a Paddle JSON file first")
	}
	pages := make([]pageRecord, len(s.pages))
	for i, page := range s.pages {
		copyPage, err := clonePage(page)
		if err != nil {
			s.mu.RUnlock()
			return ImportResult{}, err
		}
		pages[i] = copyPage
	}
	assets := cloneAssets(s.assets)
	revision := s.revision
	s.mu.RUnlock()

	hasRemote := false
	for _, page := range pages {
		for _, ref := range imageReferences(page.Raw) {
			if isRemote(ref) {
				hasRemote = true
				break
			}
		}
	}
	if !hasRemote {
		status := s.Status()
		return ImportResult{TotalPages: status.TotalPages, TotalBlocks: status.TotalBlocks}, nil
	}

	candidate, err := os.MkdirTemp(s.root, "download-")
	if err != nil {
		return ImportResult{}, err
	}
	defer os.RemoveAll(candidate)
	pageDir := filepath.Join(candidate, "pages")
	assetDir := filepath.Join(candidate, "assets")
	if err = os.MkdirAll(pageDir, 0o700); err != nil {
		return ImportResult{}, err
	}
	if err = os.MkdirAll(assetDir, 0o700); err != nil {
		return ImportResult{}, err
	}
	for id, asset := range assets {
		target := filepath.Join(assetDir, filepath.Base(asset.Path))
		if err = os.Link(asset.Path, target); err != nil {
			return ImportResult{}, err
		}
		asset.Path = target
		assets[id] = asset
	}
	downloaded, failed, err := downloadRemoteAssets(ctx, candidate, pages, assets, progress)
	if err != nil {
		return ImportResult{}, err
	}
	if downloaded == 0 {
		status := s.Status()
		return ImportResult{TotalPages: status.TotalPages, TotalBlocks: status.TotalBlocks, Failed: failed}, nil
	}
	for i := range pages {
		pages[i].Path = filepath.Join(pageDir, filepath.Base(pages[i].Path))
		if err = writeJSON(pages[i].Path, pages[i].Raw); err != nil {
			return ImportResult{}, err
		}
	}

	current := filepath.Join(s.root, "current")
	backup := filepath.Join(s.root, "old")
	_ = os.RemoveAll(backup)
	s.mu.Lock()
	if s.revision != revision {
		s.mu.Unlock()
		return ImportResult{}, coded("document_changed", "The document changed while resources were downloading")
	}
	if err = os.Rename(current, backup); err != nil {
		s.mu.Unlock()
		return ImportResult{}, err
	}
	if err = os.Rename(candidate, current); err != nil {
		_ = os.Rename(backup, current)
		s.mu.Unlock()
		return ImportResult{}, err
	}
	for i := range pages {
		pages[i].Path = filepath.Join(current, "pages", filepath.Base(pages[i].Path))
	}
	for id, asset := range assets {
		asset.Path = filepath.Join(current, "assets", filepath.Base(asset.Path))
		assets[id] = asset
	}
	s.pages, s.assets = pages, assets
	if downloaded > 0 {
		s.changed = true
		s.revision++
	}
	status := s.statusLocked()
	s.mu.Unlock()
	_ = os.RemoveAll(backup)
	return ImportResult{TotalPages: status.TotalPages, TotalBlocks: status.TotalBlocks, Downloaded: downloaded, Failed: failed}, nil
}
