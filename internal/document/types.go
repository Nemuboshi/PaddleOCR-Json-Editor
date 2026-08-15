package document

import (
	"bytes"
	"encoding/json"
)

const (
	MaxDocumentBytes = int64(1 << 30)
	MaxPages         = 10_000
)

type BBox [4]float64

type Status struct {
	Loaded      bool   `json:"loaded"`
	Changed     bool   `json:"changed"`
	TotalPages  int    `json:"total_pages"`
	TotalBlocks int    `json:"total_blocks"`
	Source      string `json:"source"`
}

type ImportResult struct {
	TotalPages  int `json:"total_pages"`
	TotalBlocks int `json:"total_blocks"`
	Downloaded  int `json:"downloaded"`
	Failed      int `json:"failed"`
}

type PageSummary struct {
	Index         int    `json:"index"`
	BlockCount    int    `json:"block_count"`
	LabelsSummary string `json:"labels_summary"`
}

type PagesResponse struct {
	Pages      []PageSummary `json:"pages"`
	PageFrom   int           `json:"page_from"`
	PageTo     int           `json:"page_to"`
	TotalPages int           `json:"total_pages"`
}

type PageBlockRow struct {
	BlockID int    `json:"block_id"`
	Label   string `json:"label"`
	Preview string `json:"preview"`
	Order   string `json:"order"`
	BBox    string `json:"bbox"`
}

type PageDetail struct {
	PageIndex     int            `json:"page_index"`
	BlockCount    int            `json:"block_count"`
	Blocks        []PageBlockRow `json:"blocks"`
	ImageURL      *string        `json:"image_url"`
	InputImage    string         `json:"input_image"`
	IsRemoteImage bool           `json:"is_remote_image"`
}

type BlockDetail struct {
	PageIndex int    `json:"page_index"`
	BlockID   int    `json:"block_id"`
	Label     string `json:"label"`
	Content   string `json:"content"`
	BBox      string `json:"bbox"`
	Order     string `json:"order"`
}

type LayoutBlock struct {
	BlockID int    `json:"block_id"`
	Label   string `json:"label"`
	BBox    BBox   `json:"bbox"`
	Content string `json:"content"`
	Order   *int   `json:"order"`
}

type PageLayout struct {
	InputImage *string           `json:"input_image"`
	BoxedImage *string           `json:"boxed_image"`
	Images     map[string]string `json:"images"`
	Blocks     []LayoutBlock     `json:"blocks"`
}

type PageView struct {
	Detail PageDetail `json:"detail"`
	Layout PageLayout `json:"layout"`
}

type UpdateBlockRequest struct {
	Page    int    `json:"page"`
	Block   int    `json:"block"`
	Label   string `json:"label"`
	Content string `json:"content"`
	BBox    string `json:"bbox"`
	Order   string `json:"order"`
}

type SearchRequest struct {
	Label    *string `json:"label"`
	Content  *string `json:"content"`
	PageFrom *int    `json:"pageFrom"`
	PageTo   *int    `json:"pageTo"`
}

type SearchHit struct {
	PageIndex int    `json:"page_index"`
	BlockID   int    `json:"block_id"`
	Label     string `json:"label"`
	Preview   string `json:"preview"`
}

type SearchResponse struct {
	Hits  []SearchHit `json:"hits"`
	Total int         `json:"total"`
}

type Message struct {
	Message string      `json:"message"`
	Page    *PageDetail `json:"page,omitempty"`
	Status  *Status     `json:"status,omitempty"`
}

type MarkdownRequest struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type pageRecord struct {
	Path   string
	Raw    map[string]any
	Blocks []map[string]any
}

type assetRecord struct {
	Path string
	MIME string
}

func decodeMap(data []byte) (map[string]any, error) {
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err := decoder.Decode(&value)
	return value, err
}
