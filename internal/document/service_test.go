package document

import (
	"context"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDocumentFlow(t *testing.T) {
	root := t.TempDir()
	service, err := New(filepath.Join(root, "session"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	result, err := service.Import(context.Background(), filepath.Join("testdata", "sample_page.json"), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalPages != 1 || result.TotalBlocks == 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if name := service.MarkdownFilename(1, 1); name != "sample_page-1-1.md" {
		t.Fatalf("unexpected Markdown filename: %q", name)
	}
	view, err := service.View(0)
	if err != nil || view.Layout.BoxedImage == nil {
		t.Fatalf("missing boxed output image: %+v, %v", view.Layout, err)
	}
	markdown, err := service.PageMarkdown(0)
	if err != nil || markdown == "" {
		t.Fatalf("missing page Markdown: %q, %v", markdown, err)
	}
	content, err := service.BlockContent(0, 0)
	if err != nil || content == "" {
		t.Fatalf("missing block content: %q, %v", content, err)
	}
	pages, err := service.Pages(0)
	if err != nil || len(pages.Pages) != 1 {
		t.Fatalf("unexpected pages: %+v, %v", pages, err)
	}
	block, err := service.Block(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	message, err := service.UpdateBlock(UpdateBlockRequest{Page: 0, Block: 0, Label: block.Label, Content: "changed content", BBox: block.BBox, Order: block.Order})
	if err != nil || message.Page == nil || message.Page.Blocks[0].Preview != "changed content" {
		t.Fatalf("unexpected update: %+v, %v", message, err)
	}
	if _, err = service.MoveBlock(0, 0, "1,2,11,12"); err != nil {
		t.Fatal(err)
	}
	moved, err := service.Block(0, 0)
	if err != nil || moved.BBox != "1,2,11,12" {
		t.Fatalf("unexpected moved bbox: %+v, %v", moved, err)
	}
	query := "changed"
	search, err := service.Search(SearchRequest{Content: &query})
	if err != nil || search.Total != 1 {
		t.Fatalf("unexpected search: %+v, %v", search, err)
	}
	target := filepath.Join(root, "export.json")
	if err = service.ExportJSON(context.Background(), target, nil); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(target); err != nil || info.Size() == 0 {
		t.Fatalf("missing export: %v", err)
	}
	if service.Status().Changed {
		t.Fatal("JSON export did not clear changed state")
	}
}

func TestMarkdownAssetsOnlyIncludesUsedImages(t *testing.T) {
	source := map[string]assetRecord{
		"used.png":    {Path: "used"},
		"ignored.png": {Path: "ignored"},
		"other.png":   {Path: "other"},
	}
	page := pageRecord{
		Raw: map[string]any{
			"prunedResult": map[string]any{"model_settings": map[string]any{"markdown_ignore_labels": []any{"footer"}}},
			"markdown": map[string]any{"images": map[string]any{
				"imgs/used.png":    "assets/used.png",
				"imgs/ignored.png": "assets/ignored.png",
			}},
		},
		Blocks: []map[string]any{
			{"block_label": "image", "block_content": `<img src="imgs/used.png">`},
			{"block_label": "footer", "block_content": `<img src="imgs/ignored.png">`},
		},
	}
	assets := markdownAssets([]pageRecord{page}, source)
	if len(assets) != 1 || assets["used.png"].Path != "used" {
		t.Fatalf("unexpected Markdown assets: %+v", assets)
	}
}

func TestMarkdownBlockContentUsesPaddleXLabels(t *testing.T) {
	tests := []struct {
		label   string
		content string
		extra   map[string]any
		want    string
	}{
		{"doc_title", "Document", nil, "# Document"},
		{"doc_title", "# Existing", nil, "# Existing"},
		{"paragraph_title", "Section", map[string]any{"title_level": json.Number("2")}, "### Section"},
		{"abstract_title", "1.2 Scope", nil, "### 1.2 Scope"},
		{"formula", "E=mc^2", nil, "$$E=mc^2$$"},
		{"formula", "$$E=mc^2$$", nil, "$$E=mc^2$$"},
		{"text", "first\nsecond", nil, "first\n\nsecond"},
		{"figure_title", "Figure 1", nil, `<div style="text-align: center;">Figure 1</div>` + "\n"},
	}
	for _, test := range tests {
		block := map[string]any{"block_label": test.label, "block_content": test.content}
		for key, value := range test.extra {
			block[key] = value
		}
		if got := markdownBlockContent(block); got != test.want {
			t.Errorf("%s: got %q, want %q", test.label, got, test.want)
		}
	}
}

func TestDeletePageReindexesAndKeepsOnePage(t *testing.T) {
	root := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "sample_page.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pages []map[string]any
	if err = json.Unmarshal(data, &pages); err != nil {
		t.Fatal(err)
	}
	pages = append(pages, pages[0])
	source := filepath.Join(root, "two-pages.json")
	if err = os.WriteFile(source, mustJSON(t, pages), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := New(filepath.Join(root, "session"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if _, err = service.Import(context.Background(), source, false, nil); err != nil {
		t.Fatal(err)
	}
	block, err := service.Block(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	block.Content = "second page"
	if _, err = service.UpdateBlock(UpdateBlockRequest{Page: 1, Block: 0, Label: block.Label, Content: block.Content, BBox: block.BBox, Order: block.Order}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.DeletePage(0); err != nil {
		t.Fatal(err)
	}
	shifted, err := service.Block(0, 0)
	if err != nil || shifted.Content != "second page" || service.Status().TotalPages != 1 {
		t.Fatalf("page was not reindexed: %+v, %v", shifted, err)
	}
	if _, err = service.DeletePage(0); err == nil {
		t.Fatal("last page was deleted")
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func importedService(t *testing.T) *Service {
	t.Helper()
	service, err := New(filepath.Join(t.TempDir(), "session"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { service.Close() })
	if _, err = service.Import(context.Background(), filepath.Join("testdata", "sample_page.json"), false, nil); err != nil {
		t.Fatal(err)
	}
	return service
}

func errorCode(err error) string {
	if coded, ok := err.(*Error); ok {
		return coded.Code
	}
	return ""
}

func TestBlockEditingOperations(t *testing.T) {
	service := importedService(t)
	first, _ := service.Block(0, 3)
	second, _ := service.Block(0, 4)
	if _, err := service.MergeBlocks(0, "3,4", " / "); err != nil {
		t.Fatal(err)
	}
	merged, err := service.Block(0, 3)
	if err != nil || merged.Content != first.Content+" / "+second.Content || merged.BBox != "111,137,1083,408" || merged.Order != "1" {
		t.Fatalf("unexpected merged block: %+v, %v", merged, err)
	}
	if _, err = service.Block(0, 4); errorCode(err) != "block_not_found" {
		t.Fatalf("merged block was not removed: %v", err)
	}

	merged.Content = "alpha\u00e9beta"
	if _, err = service.UpdateBlock(UpdateBlockRequest{Page: 0, Block: 3, Label: merged.Label, Content: merged.Content, BBox: merged.BBox, Order: merged.Order}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.SplitBlock(0, 3, 6); err != nil {
		t.Fatal(err)
	}
	left, _ := service.Block(0, 3)
	view, err := service.View(0)
	if err != nil || left.Content != "alpha\u00e9" || view.Detail.BlockCount != 16 || len(view.Layout.Blocks) != 16 {
		t.Fatalf("unexpected split result: left=%+v detail=%d layout=%d err=%v", left, view.Detail.BlockCount, len(view.Layout.Blocks), err)
	}
	right, err := service.Block(0, 16)
	if err != nil || right.Content != "beta" {
		t.Fatalf("split did not preserve Unicode content: %+v, %v", right, err)
	}

	ids := make([]string, len(view.Detail.Blocks))
	for i, block := range view.Detail.Blocks {
		ids[len(ids)-1-i] = blockIDString(block.BlockID)
	}
	if _, err = service.ReorderBlocks(0, strings.Join(ids, ",")); err != nil {
		t.Fatal(err)
	}
	for i, block := range service.pages[0].Blocks {
		want := len(service.pages[0].Blocks) - i
		if intValue(block["block_order"]) != want {
			t.Fatalf("block %d order=%v, want %d", intValue(block["block_id"]), block["block_order"], want)
		}
	}
}

func blockIDString(id int) string { return strconv.Itoa(id) }

func TestEditingValidationDoesNotMutateDocument(t *testing.T) {
	service := importedService(t)
	before := service.revision
	tests := []struct {
		name string
		code string
		run  func() error
	}{
		{"empty label", "invalid_label", func() error { _, err := service.UpdateBlock(UpdateBlockRequest{Page: 0, Block: 3}); return err }},
		{"invalid bbox", "invalid_bbox", func() error { _, err := service.MoveBlock(0, 3, "1,2,1,4"); return err }},
		{"duplicate merge", "merge_too_few", func() error { _, err := service.MergeBlocks(0, "3,3", ""); return err }},
		{"invalid split", "split_out_of_range", func() error { _, err := service.SplitBlock(0, 3, 99); return err }},
		{"incomplete reorder", "reorder_mismatch", func() error { _, err := service.ReorderBlocks(0, "3,4"); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if code := errorCode(test.run()); code != test.code {
				t.Fatalf("code=%q, want %q", code, test.code)
			}
		})
	}
	if service.revision != before || service.Status().Changed {
		t.Fatal("rejected edits changed the document")
	}
}

func TestSearchFiltersAndLimits(t *testing.T) {
	service := importedService(t)
	block, _ := service.Block(0, 3)
	if _, err := service.UpdateBlock(UpdateBlockRequest{Page: 0, Block: 3, Label: "test_label", Content: "unique search needle", BBox: block.BBox, Order: block.Order}); err != nil {
		t.Fatal(err)
	}
	label, content, from, to := "test_label", "search needle", 0, 0
	result, err := service.Search(SearchRequest{Label: &label, Content: &content, PageFrom: &from, PageTo: &to})
	if err != nil || result.Total != 1 || len(result.Hits) != 1 || result.Hits[0].BlockID != 3 {
		t.Fatalf("unexpected filtered search: %+v, %v", result, err)
	}
	from = 1
	result, err = service.Search(SearchRequest{PageFrom: &from})
	if err != nil || result.Total != 0 || len(result.Hits) != 0 {
		t.Fatalf("out-of-range search returned hits: %+v, %v", result, err)
	}
}

func TestMarkdownExportRangeAndContent(t *testing.T) {
	service := importedService(t)
	target := filepath.Join(t.TempDir(), "export")
	var progressDone, progressTotal int
	if err := service.ExportMarkdown(context.Background(), target, 1, 1, func(done, total int, _ string) {
		progressDone, progressTotal = done, total
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target + ".md")
	if err != nil || len(data) == 0 || progressDone != 1 || progressTotal != 1 {
		t.Fatalf("unexpected Markdown export: bytes=%d progress=%d/%d err=%v", len(data), progressDone, progressTotal, err)
	}
	if err = service.ExportMarkdown(context.Background(), target, 0, 1, nil); errorCode(err) != "invalid_range" {
		t.Fatalf("invalid range error=%v", err)
	}
}

func TestFailedImportKeepsCurrentDocument(t *testing.T) {
	service := importedService(t)
	invalid := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalid, []byte(`[{"prunedResult":{}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Import(context.Background(), invalid, false, nil); errorCode(err) != "json_error" {
		t.Fatalf("unexpected import error: %v", err)
	}
	if status := service.Status(); !status.Loaded || status.TotalPages != 1 || status.TotalBlocks != 16 {
		t.Fatalf("failed import replaced current document: %+v", status)
	}
}

func TestLocalAssetImportAndTraversalProtection(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "page.png")
	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(file, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	file.Close()

	data, err := os.ReadFile(filepath.Join("testdata", "sample_page.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pages []map[string]any
	if err = json.Unmarshal(data, &pages); err != nil {
		t.Fatal(err)
	}
	pages[0]["inputImage"] = "page.png"
	source := filepath.Join(root, "document.json")
	if err = os.WriteFile(source, mustJSON(t, pages), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := New(filepath.Join(root, "session"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if _, err = service.Import(context.Background(), source, false, nil); err != nil {
		t.Fatal(err)
	}
	view, err := service.View(0)
	if err != nil || view.Detail.ImageURL == nil || !strings.HasPrefix(*view.Detail.ImageURL, "/session-assets/") {
		t.Fatalf("local asset was not imported: %+v, %v", view.Detail.ImageURL, err)
	}
	id := strings.TrimPrefix(*view.Detail.ImageURL, "/session-assets/")
	path, mime, err := service.Asset(id)
	if err != nil || mime != "image/png" {
		t.Fatalf("asset lookup failed: %q %q %v", path, mime, err)
	}
	if _, _, err = service.Asset("../page.png"); err == nil {
		t.Fatal("asset traversal was accepted")
	}

	pages[0]["inputImage"] = "../page.png"
	outside := filepath.Join(root, "child")
	if err = os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	unsafe := filepath.Join(outside, "unsafe.json")
	if err = os.WriteFile(unsafe, mustJSON(t, pages), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Import(context.Background(), unsafe, false, nil); errorCode(err) != "unsafe_asset" {
		t.Fatalf("unsafe asset error=%v", err)
	}
}

func TestDownloadAssetsDoesNothingWithoutRemoteImages(t *testing.T) {
	service, err := New(filepath.Join(t.TempDir(), "session"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if _, err = service.Import(context.Background(), filepath.Join("testdata", "sample_page.json"), false, nil); err != nil {
		t.Fatal(err)
	}
	service.mu.Lock()
	for i := range service.pages {
		for _, ref := range references([]pageRecord{service.pages[i]}) {
			if isRemote(ref.get()) {
				ref.set("")
			}
		}
	}
	revision := service.revision
	service.mu.Unlock()
	result, err := service.DownloadAssets(context.Background(), nil)
	if err != nil || result.Downloaded != 0 || service.revision != revision {
		t.Fatalf("local-only download was not a no-op: %+v, %v", result, err)
	}
}

func TestExportDoesNotBlockPageBrowsing(t *testing.T) {
	root := t.TempDir()
	service, err := New(filepath.Join(root, "session"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if _, err = service.Import(context.Background(), filepath.Join("testdata", "sample_page.json"), false, nil); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	resume := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- service.ExportJSON(context.Background(), filepath.Join(root, "export.json"), func(_, _ int, _ string) {
			close(started)
			<-resume
		})
	}()
	<-started

	viewDone := make(chan error, 1)
	go func() {
		_, err := service.View(0)
		viewDone <- err
	}()
	select {
	case err = <-viewDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("page view was blocked by export")
	}
	close(resume)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}

func TestLimitsAndValidation(t *testing.T) {
	root := t.TempDir()
	service, err := New(filepath.Join(root, "session"))
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	invalid := filepath.Join(root, "invalid.json")
	if err = os.WriteFile(invalid, []byte(`{"not":"an array"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Import(context.Background(), invalid, false, nil); err == nil {
		t.Fatal("invalid root was accepted")
	} else if coded, ok := err.(*Error); !ok || coded.Code != "json_error" || coded.Message != "This isn't a supported PaddleOCR JSON file" {
		t.Fatalf("unexpected invalid-root error: %#v", err)
	}
	large := filepath.Join(root, "large.json")
	file, err := os.Create(large)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(MaxDocumentBytes + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if _, err = service.Import(context.Background(), large, false, nil); err == nil {
		t.Fatal("oversized input was accepted")
	}
}
