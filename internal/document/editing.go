package document

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

func (s *Service) UpdateBlock(request UpdateBlockRequest) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, _, err := s.requireBlock(request.Page, request.Block)
	if err != nil {
		return Message{}, err
	}
	copyPage, err := clonePage(s.pages[request.Page])
	if err != nil {
		return Message{}, err
	}
	block := copyPage.Blocks[index]
	if request.Label == "" {
		return Message{}, coded("invalid_label", "The block label is required")
	}
	block["block_label"] = request.Label
	block["block_content"] = request.Content
	if strings.TrimSpace(request.BBox) != "" {
		box, err := parseBBox(request.BBox)
		if err != nil {
			return Message{}, err
		}
		block["block_bbox"] = bboxAny(box)
		block["block_polygon_points"] = []any{[]any{box[0], box[1]}, []any{box[2], box[1]}, []any{box[2], box[3]}, []any{box[0], box[3]}}
	}
	if strings.TrimSpace(request.Order) != "" {
		order, err := strconv.Atoi(request.Order)
		if err != nil || order < 0 {
			return Message{}, coded("invalid_order", "The order must be a non-negative integer")
		}
		block["block_order"] = json.Number(strconv.Itoa(order))
	}
	syncLayout(copyPage.Raw, index, block)
	return s.commitPage(request.Page, copyPage, "Block saved")
}

func (s *Service) MoveBlock(pageIndex, blockID int, bbox string) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, _, err := s.requireBlock(pageIndex, blockID)
	if err != nil {
		return Message{}, err
	}
	box, err := parseBBox(bbox)
	if err != nil {
		return Message{}, err
	}
	copyPage, err := clonePage(s.pages[pageIndex])
	if err != nil {
		return Message{}, err
	}
	block := copyPage.Blocks[index]
	block["block_bbox"] = bboxAny(box)
	block["block_polygon_points"] = []any{[]any{box[0], box[1]}, []any{box[2], box[1]}, []any{box[2], box[3]}, []any{box[0], box[3]}}
	syncLayout(copyPage.Raw, index, block)
	return s.commitPage(pageIndex, copyPage, "Block moved")
}

func (s *Service) DeletePage(pageIndex int) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.requirePage(pageIndex); err != nil {
		return Message{}, err
	}
	if len(s.pages) == 1 {
		return Message{}, coded("last_page", "The document must keep at least one page")
	}
	page := s.pages[pageIndex]
	backup := page.Path + ".deleted"
	_ = os.Remove(backup)
	if err := os.Rename(page.Path, backup); err != nil {
		return Message{}, err
	}
	s.pages = append(s.pages[:pageIndex], s.pages[pageIndex+1:]...)
	used := map[string]bool{}
	for _, current := range s.pages {
		for _, value := range imageReferences(current.Raw) {
			id := strings.TrimPrefix(strings.ReplaceAll(value, "\\", "/"), "assets/")
			if _, ok := s.assets[id]; ok {
				used[id] = true
			}
		}
	}
	for id, asset := range s.assets {
		if !used[id] {
			_ = os.Remove(asset.Path)
			delete(s.assets, id)
		}
	}
	_ = os.Remove(backup)
	s.changed = true
	s.revision++
	status := s.statusLocked()
	return Message{Message: fmt.Sprintf("Deleted page %d", pageIndex), Status: &status}, nil
}

func (s *Service) DeleteBlock(pageIndex, blockID int) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, _, err := s.requireBlock(pageIndex, blockID)
	if err != nil {
		return Message{}, err
	}
	p, err := clonePage(s.pages[pageIndex])
	if err != nil {
		return Message{}, err
	}
	p.Blocks = append(p.Blocks[:index], p.Blocks[index+1:]...)
	setBlocks(p.Raw, p.Blocks)
	removeLayoutBox(p.Raw, index)
	return s.commitPage(pageIndex, p, fmt.Sprintf("Deleted block %d", blockID))
}
func (s *Service) MergeBlocks(pageIndex int, value, separator string) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := parseIDs(value)
	if err != nil || len(ids) < 2 {
		return Message{}, coded("merge_too_few", "Enter at least two distinct block IDs")
	}
	seen := map[int]bool{}
	indices := make([]int, len(ids))
	for i, id := range ids {
		if seen[id] {
			return Message{}, coded("merge_too_few", "Enter distinct block IDs")
		}
		seen[id] = true
		index, _, e := s.requireBlock(pageIndex, id)
		if e != nil {
			return Message{}, e
		}
		indices[i] = index
	}
	p, err := clonePage(s.pages[pageIndex])
	if err != nil {
		return Message{}, err
	}
	primary := p.Blocks[indices[0]]
	contents := make([]string, len(indices))
	boxes := make([]BBox, len(indices))
	minOrder := -1
	for i, index := range indices {
		b := p.Blocks[index]
		contents[i] = stringValue(b["block_content"])
		boxes[i] = bboxValue(b["block_bbox"])
		if b["block_order"] != nil {
			order := intValue(b["block_order"])
			if minOrder < 0 || order < minOrder {
				minOrder = order
			}
		}
	}
	primary["block_content"] = strings.Join(contents, separator)
	box := mergeBoxes(boxes)
	primary["block_bbox"] = bboxAny(box)
	if minOrder >= 0 {
		primary["block_order"] = json.Number(strconv.Itoa(minOrder))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(indices[1:])))
	for _, index := range indices[1:] {
		p.Blocks = append(p.Blocks[:index], p.Blocks[index+1:]...)
		removeLayoutBox(p.Raw, index)
	}
	setBlocks(p.Raw, p.Blocks)
	return s.commitPage(pageIndex, p, "Blocks merged")
}
func (s *Service) SplitBlock(pageIndex, blockID, at int) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, source, err := s.requireBlock(pageIndex, blockID)
	if err != nil {
		return Message{}, err
	}
	content := []rune(stringValue(source["block_content"]))
	if at <= 0 || at >= len(content) {
		return Message{}, coded("split_out_of_range", fmt.Sprintf("The split position must be from 1 to %d", len(content)-1))
	}
	p, err := clonePage(s.pages[pageIndex])
	if err != nil {
		return Message{}, err
	}
	source = p.Blocks[index]
	next, err := cloneMap(source)
	if err != nil {
		return Message{}, err
	}
	maxID := 0
	for _, b := range p.Blocks {
		maxID = max(maxID, intValue(b["block_id"]))
	}
	next["block_id"] = json.Number(strconv.Itoa(maxID + 1))
	next["block_content"] = string(content[at:])
	source["block_content"] = string(content[:at])
	if source["block_order"] != nil {
		next["block_order"] = json.Number(strconv.Itoa(intValue(source["block_order"]) + 1))
	}
	p.Blocks = append(p.Blocks, nil)
	copy(p.Blocks[index+2:], p.Blocks[index+1:])
	p.Blocks[index+1] = next
	setBlocks(p.Raw, p.Blocks)
	duplicateLayoutBox(p.Raw, index)
	return s.commitPage(pageIndex, p, "Block split")
}
func (s *Service) ReorderBlocks(pageIndex int, value string) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := parseIDs(value)
	if err != nil {
		return Message{}, coded("reorder_mismatch", "Enter every block ID once")
	}
	p, err := clonePage(s.pages[pageIndex])
	if err != nil {
		return Message{}, err
	}
	if len(ids) != len(p.Blocks) {
		return Message{}, coded("reorder_mismatch", "Enter every block ID once")
	}
	byID := map[int]map[string]any{}
	for _, b := range p.Blocks {
		byID[intValue(b["block_id"])] = b
	}
	if len(byID) != len(ids) {
		return Message{}, coded("reorder_mismatch", "Enter every block ID once")
	}
	for i, id := range ids {
		b, ok := byID[id]
		if !ok {
			return Message{}, coded("reorder_mismatch", "Enter every block ID once")
		}
		b["block_order"] = json.Number(strconv.Itoa(i + 1))
	}
	return s.commitPage(pageIndex, p, "Blocks reordered")
}

func (s *Service) commitPage(index int, page pageRecord, message string) (Message, error) {
	if err := writeJSONAtomic(s.pages[index].Path, page.Raw); err != nil {
		return Message{}, err
	}
	page.Path = s.pages[index].Path
	s.pages[index] = page
	s.changed = true
	s.revision++
	detail, err := s.pageLocked(index)
	if err != nil {
		return Message{}, err
	}
	status := s.statusLocked()
	return Message{Message: message, Page: &detail, Status: &status}, nil
}
func clonePage(page pageRecord) (pageRecord, error) {
	raw, err := cloneMap(page.Raw)
	if err != nil {
		return pageRecord{}, err
	}
	blocks, err := validatePage(raw)
	if err != nil {
		return pageRecord{}, err
	}
	return pageRecord{Path: page.Path, Raw: raw, Blocks: blocks}, nil
}
func cloneMap(value map[string]any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return decodeMap(data)
}
func parseBBox(value string) (BBox, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return BBox{}, coded("invalid_bbox", "The bounding box must contain four numbers")
	}
	var box BBox
	for i, p := range parts {
		n, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return BBox{}, coded("invalid_bbox", "The bounding box must contain four numbers")
		}
		box[i] = n
	}
	box = BBox{min(box[0], box[2]), min(box[1], box[3]), max(box[0], box[2]), max(box[1], box[3])}
	if box[2] <= box[0] || box[3] <= box[1] {
		return BBox{}, coded("invalid_bbox", "The bounding box must have positive width and height")
	}
	return box, nil
}
func bboxAny(box BBox) []any { return []any{box[0], box[1], box[2], box[3]} }
func mergeBoxes(boxes []BBox) BBox {
	result := BBox{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, b := range boxes {
		result[0] = min(result[0], b[0])
		result[1] = min(result[1], b[1])
		result[2] = max(result[2], b[2])
		result[3] = max(result[3], b[3])
	}
	return result
}
func parseIDs(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	ids := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 0 {
			return nil, err
		}
		ids[i] = n
	}
	return ids, nil
}
func setBlocks(raw map[string]any, blocks []map[string]any) {
	pr := raw["prunedResult"].(map[string]any)
	values := make([]any, len(blocks))
	for i, b := range blocks {
		values[i] = b
	}
	pr["parsing_res_list"] = values
}
func layoutBoxes(raw map[string]any) (map[string]any, []any) {
	pr, _ := raw["prunedResult"].(map[string]any)
	layout, _ := pr["layout_det_res"].(map[string]any)
	boxes, _ := layout["boxes"].([]any)
	return layout, boxes
}
func removeLayoutBox(raw map[string]any, index int) {
	layout, boxes := layoutBoxes(raw)
	if layout != nil && index < len(boxes) {
		layout["boxes"] = append(boxes[:index], boxes[index+1:]...)
	}
}
func duplicateLayoutBox(raw map[string]any, index int) {
	layout, boxes := layoutBoxes(raw)
	if layout == nil || index >= len(boxes) {
		return
	}
	copyValue := boxes[index]
	if box, ok := copyValue.(map[string]any); ok {
		copyValue = make(map[string]any, len(box))
		for key, value := range box {
			copyValue.(map[string]any)[key] = value
		}
	}
	boxes = append(boxes, nil)
	copy(boxes[index+2:], boxes[index+1:])
	boxes[index+1] = copyValue
	layout["boxes"] = boxes
}
func syncLayout(raw map[string]any, index int, block map[string]any) {
	layout, boxes := layoutBoxes(raw)
	if layout == nil || index >= len(boxes) {
		return
	}
	box, _ := boxes[index].(map[string]any)
	if box == nil {
		return
	}
	box["label"] = block["block_label"]
	box["coordinate"] = block["block_bbox"]
	if polygon := block["block_polygon_points"]; polygon != nil {
		box["polygon_points"] = polygon
	}
}
