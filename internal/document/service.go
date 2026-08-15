package document

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

type Service struct {
	mu         sync.RWMutex
	root       string
	sourceName string
	pages      []pageRecord
	assets     map[string]assetRecord
	changed    bool
	revision   uint64
}

func New(root string) (*Service, error) {
	if err := os.RemoveAll(root); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(root, "current", "assets"), 0o700); err != nil {
		return nil, err
	}
	return &Service{root: root, assets: map[string]assetRecord{}}, nil
}

func (s *Service) Close() error { return os.RemoveAll(s.root) }

func (s *Service) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

func (s *Service) MarkdownFilename(start, end int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	base := strings.TrimSuffix(s.sourceName, filepath.Ext(s.sourceName))
	if base == "" {
		base = "document"
	}
	return fmt.Sprintf("%s-%d-%d.md", base, start, end)
}

func (s *Service) statusLocked() Status {
	blocks := 0
	remote, local := false, false
	for _, page := range s.pages {
		blocks += len(page.Blocks)
		for _, ref := range imageReferences(page.Raw) {
			if isRemote(ref) {
				remote = true
			} else if ref != "" {
				local = true
			}
		}
	}
	source := "No source"
	if len(s.pages) > 0 {
		switch {
		case remote && local:
			source = "Mixed"
		case remote:
			source = "Remote"
		case local:
			source = "Local"
		default:
			source = "No assets"
		}
	}
	return Status{Loaded: len(s.pages) > 0, Changed: s.changed, TotalPages: len(s.pages), TotalBlocks: blocks, Source: source}
}

func (s *Service) Import(ctx context.Context, sourcePath string, download bool, progress func(int, int, string)) (ImportResult, error) {
	info, err := os.Stat(sourcePath)
	if err != nil || !info.Mode().IsRegular() {
		return ImportResult{}, coded("file_not_found", "The selected JSON file does not exist")
	}
	if info.Size() > MaxDocumentBytes {
		return ImportResult{}, coded("file_too_large", "The JSON file is larger than 1 GiB")
	}
	candidate, err := os.MkdirTemp(s.root, "import-")
	if err != nil {
		return ImportResult{}, err
	}
	defer os.RemoveAll(candidate)
	if err = os.MkdirAll(filepath.Join(candidate, "pages"), 0o700); err != nil {
		return ImportResult{}, err
	}
	if err = os.MkdirAll(filepath.Join(candidate, "assets"), 0o700); err != nil {
		return ImportResult{}, err
	}

	file, err := os.Open(sourcePath)
	if err != nil {
		return ImportResult{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(bufio.NewReader(file))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return ImportResult{}, coded("json_error", err.Error())
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return ImportResult{}, coded("json_error", "This isn't a supported PaddleOCR JSON file")
	}
	pages := make([]pageRecord, 0, 128)
	for decoder.More() {
		select {
		case <-ctx.Done():
			return ImportResult{}, ctx.Err()
		default:
		}
		if len(pages) >= MaxPages {
			return ImportResult{}, coded("too_many_pages", "The document has more than 10,000 pages")
		}
		var raw map[string]any
		if err = decoder.Decode(&raw); err != nil {
			return ImportResult{}, coded("json_error", fmt.Sprintf("Page %d: %v", len(pages)+1, err))
		}
		blocks, err := validatePage(raw)
		if err != nil {
			return ImportResult{}, coded("json_error", fmt.Sprintf("Page %d: %v", len(pages)+1, err))
		}
		path := filepath.Join(candidate, "pages", fmt.Sprintf("%05d.json", len(pages)))
		if err = writeJSON(path, raw); err != nil {
			return ImportResult{}, err
		}
		pages = append(pages, pageRecord{Path: path, Raw: raw, Blocks: blocks})
		if progress != nil {
			progress(len(pages), 0, "Importing pages")
		}
	}
	if _, err = decoder.Token(); err != nil {
		return ImportResult{}, coded("json_error", err.Error())
	}
	if len(pages) == 0 {
		return ImportResult{}, coded("json_error", "The document contains no pages")
	}

	assets := map[string]assetRecord{}
	if err = importLocalAssets(sourcePath, candidate, pages, assets); err != nil {
		return ImportResult{}, err
	}
	downloaded, failed := 0, 0
	if download {
		downloaded, failed, err = downloadRemoteAssets(ctx, candidate, pages, assets, progress)
		if err != nil {
			return ImportResult{}, err
		}
	}
	for i := range pages {
		if err = writeJSON(pages[i].Path, pages[i].Raw); err != nil {
			return ImportResult{}, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current := filepath.Join(s.root, "current")
	backup := filepath.Join(s.root, "old")
	_ = os.RemoveAll(backup)
	if err = os.Rename(current, backup); err != nil {
		return ImportResult{}, err
	}
	if err = os.Rename(candidate, current); err != nil {
		_ = os.Rename(backup, current)
		return ImportResult{}, err
	}
	_ = os.RemoveAll(backup)
	for i := range pages {
		pages[i].Path = filepath.Join(current, "pages", filepath.Base(pages[i].Path))
	}
	for id, a := range assets {
		a.Path = filepath.Join(current, "assets", filepath.Base(a.Path))
		assets[id] = a
	}
	s.pages = pages
	s.assets = assets
	s.sourceName = filepath.Base(sourcePath)
	s.changed = downloaded > 0
	s.revision++
	status := s.statusLocked()
	return ImportResult{TotalPages: status.TotalPages, TotalBlocks: status.TotalBlocks, Downloaded: downloaded, Failed: failed}, nil
}

func (s *Service) Pages(from int) (PagesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.pages) == 0 {
		return PagesResponse{}, coded("not_loaded", "Import a Paddle JSON file first")
	}
	if from < 0 {
		return PagesResponse{}, coded("invalid_page", "The page number must not be negative")
	}
	to := min(from+49, len(s.pages)-1)
	result := PagesResponse{PageFrom: from, PageTo: to, TotalPages: len(s.pages), Pages: []PageSummary{}}
	if from >= len(s.pages) {
		return result, nil
	}
	for i := from; i <= to; i++ {
		counts := map[string]int{}
		for _, b := range s.pages[i].Blocks {
			counts[stringValue(b["block_label"])]++
		}
		type pair struct {
			key   string
			value int
		}
		pairs := []pair{}
		for k, v := range counts {
			pairs = append(pairs, pair{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].value > pairs[j].value })
		labels := []string{}
		for _, p := range pairs[:min(3, len(pairs))] {
			labels = append(labels, fmt.Sprintf("%s:%d", p.key, p.value))
		}
		result.Pages = append(result.Pages, PageSummary{Index: i, BlockCount: len(s.pages[i].Blocks), LabelsSummary: strings.Join(labels, " ")})
	}
	return result, nil
}

func (s *Service) View(index int) (PageView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	detail, err := s.pageLocked(index)
	if err != nil {
		return PageView{}, err
	}
	layout, err := s.layoutLocked(index)
	return PageView{Detail: detail, Layout: layout}, err
}

func (s *Service) pageLocked(index int) (PageDetail, error) {
	page, err := s.requirePage(index)
	if err != nil {
		return PageDetail{}, err
	}
	rows := make([]PageBlockRow, 0, len(page.Blocks))
	for _, b := range page.Blocks {
		rows = append(rows, PageBlockRow{BlockID: intValue(b["block_id"]), Label: stringValue(b["block_label"]), Preview: truncate(stringValue(b["block_content"]), 80), Order: optionalNumber(b["block_order"]), BBox: formatBBox(bboxValue(b["block_bbox"]))})
	}
	input := stringValue(page.Raw["inputImage"])
	url := s.resolveAsset(input)
	var ptr *string
	if url != "" {
		ptr = &url
	}
	return PageDetail{PageIndex: index, BlockCount: len(rows), Blocks: rows, ImageURL: ptr, InputImage: input, IsRemoteImage: isRemote(url)}, nil
}

func (s *Service) Block(pageIndex, blockID int) (BlockDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, b, err := s.requireBlock(pageIndex, blockID)
	if err != nil {
		return BlockDetail{}, err
	}
	return BlockDetail{PageIndex: pageIndex, BlockID: blockID, Label: stringValue(b["block_label"]), Content: stringValue(b["block_content"]), BBox: formatBBox(bboxValue(b["block_bbox"])), Order: optionalNumber(b["block_order"])}, nil
}
func (s *Service) layoutLocked(pageIndex int) (PageLayout, error) {
	p, err := s.requirePage(pageIndex)
	if err != nil {
		return PageLayout{}, err
	}
	layout := PageLayout{Images: map[string]string{}, Blocks: []LayoutBlock{}}
	for _, b := range p.Blocks {
		box := bboxValue(b["block_bbox"])
		var order *int
		if b["block_order"] != nil {
			v := intValue(b["block_order"])
			order = &v
		}
		layout.Blocks = append(layout.Blocks, LayoutBlock{BlockID: intValue(b["block_id"]), Label: stringValue(b["block_label"]), BBox: box, Content: stringValue(b["block_content"]), Order: order})
	}
	input := s.resolveAsset(stringValue(p.Raw["inputImage"]))
	if input != "" {
		layout.InputImage = &input
	}
	if images, _ := p.Raw["outputImages"].(map[string]any); images != nil {
		boxed := s.resolveAsset(stringValue(images["layout_det_res"]))
		if boxed != "" {
			layout.BoxedImage = &boxed
		}
	}
	if md, _ := p.Raw["markdown"].(map[string]any); md != nil {
		if imgs, _ := md["images"].(map[string]any); imgs != nil {
			for k, v := range imgs {
				layout.Images[k] = s.resolveAsset(stringValue(v))
			}
		}
	}
	return layout, nil
}

func (s *Service) Search(request SearchRequest) (SearchResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.pages) == 0 {
		return SearchResponse{}, coded("not_loaded", "Import a Paddle JSON file first")
	}
	from, to := 0, len(s.pages)-1
	if request.PageFrom != nil {
		from = *request.PageFrom
	}
	if request.PageTo != nil {
		to = *request.PageTo
	}
	result := SearchResponse{Hits: []SearchHit{}}
	for pi, p := range s.pages {
		if pi < from || pi > to {
			continue
		}
		for _, b := range p.Blocks {
			label, content := stringValue(b["block_label"]), stringValue(b["block_content"])
			if request.Label != nil && *request.Label != "" && label != *request.Label {
				continue
			}
			if request.Content != nil && *request.Content != "" && !strings.Contains(content, *request.Content) {
				continue
			}
			result.Total++
			if len(result.Hits) < 100 {
				result.Hits = append(result.Hits, SearchHit{PageIndex: pi, BlockID: intValue(b["block_id"]), Label: label, Preview: truncate(content, 60)})
			}
		}
	}
	return result, nil
}

func (s *Service) requirePage(index int) (*pageRecord, error) {
	if index < 0 || index >= len(s.pages) {
		return nil, coded("page_not_found", fmt.Sprintf("Page %d does not exist", index+1))
	}
	return &s.pages[index], nil
}
func (s *Service) requireBlock(pageIndex, blockID int) (int, map[string]any, error) {
	p, err := s.requirePage(pageIndex)
	if err != nil {
		return 0, nil, err
	}
	for i, b := range p.Blocks {
		if intValue(b["block_id"]) == blockID {
			return i, b, nil
		}
	}
	return 0, nil, coded("block_not_found", fmt.Sprintf("Block %d does not exist", blockID))
}

func validatePage(raw map[string]any) ([]map[string]any, error) {
	pr, ok := raw["prunedResult"].(map[string]any)
	if !ok {
		return nil, errors.New("prunedResult is missing")
	}
	values, ok := pr["parsing_res_list"].([]any)
	if !ok {
		return nil, errors.New("prunedResult.parsing_res_list is missing")
	}
	md, ok := raw["markdown"].(map[string]any)
	if !ok {
		return nil, errors.New("markdown is missing")
	}
	if _, ok = md["images"].(map[string]any); !ok {
		return nil, errors.New("markdown.images is missing")
	}
	blocks := make([]map[string]any, len(values))
	for i, v := range values {
		b, ok := v.(map[string]any)
		if !ok || stringValue(b["block_label"]) == "" {
			return nil, fmt.Errorf("block %d has an invalid label", i)
		}
		if _, ok = b["block_content"].(string); !ok {
			return nil, fmt.Errorf("block %d has invalid content", i)
		}
		if _, ok = b["block_id"].(json.Number); !ok {
			return nil, fmt.Errorf("block %d has an invalid ID", i)
		}
		if _, ok := validBBox(b["block_bbox"]); !ok {
			return nil, fmt.Errorf("block %d has an invalid bounding box", i)
		}
		blocks[i] = b
	}
	return blocks, nil
}
func writeJSON(path string, value any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err = encoder.Encode(value); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func writeJSONAtomic(path string, value any) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".write-")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err = encoder.Encode(value); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return replaceFile(name, path)
}

func replaceFile(source, target string) error {
	backup := target + ".backup"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
func validBBox(value any) (BBox, bool) {
	a, ok := value.([]any)
	if !ok || len(a) != 4 {
		return BBox{}, false
	}
	var b BBox
	for i, v := range a {
		n, ok := number(v)
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
			return BBox{}, false
		}
		b[i] = n
	}
	return b, true
}
func bboxValue(v any) BBox { b, _ := validBBox(v); return b }
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, e := n.Float64()
		return f, e == nil
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
}
func intValue(v any) int       { n, _ := number(v); return int(n) }
func stringValue(v any) string { s, _ := v.(string); return s }
func optionalNumber(v any) string {
	if v == nil {
		return ""
	}
	if n, ok := v.(json.Number); ok {
		return n.String()
	}
	return strconv.Itoa(intValue(v))
}
func formatBBox(b BBox) string {
	parts := make([]string, 4)
	for i, n := range b {
		parts[i] = strconv.FormatFloat(n, 'f', -1, 64)
	}
	return strings.Join(parts, ",")
}
func truncate(value string, limit int) string {
	value = strings.ReplaceAll(value, "\n", "")
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	r := []rune(value)
	return string(r[:limit]) + "..."
}
func isRemote(value string) bool {
	return strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://")
}
func (s *Service) resolveAsset(value string) string {
	if value == "" || isRemote(value) {
		return value
	}
	id := strings.TrimPrefix(strings.ReplaceAll(value, "\\", "/"), "assets/")
	if _, ok := s.assets[id]; ok {
		return "/session-assets/" + id
	}
	return ""
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string         { return e.Message }
func coded(code, message string) error { return &Error{code, message} }
