package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const maxAssetBytes = int64(100 << 20)
const maxAssetsBytes = int64(10 << 30)

type imageReference struct {
	page int
	kind string
	get  func() string
	set  func(string)
}

func references(pages []pageRecord) []imageReference {
	var refs []imageReference
	for pageIndex := range pages {
		page := pages[pageIndex].Raw
		refs = append(refs, imageReference{pageIndex, "inputImage", func() string { return stringValue(page["inputImage"]) }, func(v string) { page["inputImage"] = v }})
		if markdown, _ := page["markdown"].(map[string]any); markdown != nil {
			if images, _ := markdown["images"].(map[string]any); images != nil {
				for key := range images {
					key := key
					refs = append(refs, imageReference{pageIndex, "markdown image", func() string { return stringValue(images[key]) }, func(v string) { images[key] = v }})
				}
			}
		}
		if images, _ := page["outputImages"].(map[string]any); images != nil {
			for key := range images {
				key := key
				refs = append(refs, imageReference{pageIndex, "output image", func() string { return stringValue(images[key]) }, func(v string) { images[key] = v }})
			}
		}
	}
	return refs
}

func imageReferences(page map[string]any) (out []string) {
	add := func(value any) {
		if value := stringValue(value); value != "" {
			out = append(out, value)
		}
	}
	add(page["inputImage"])
	if markdown, _ := page["markdown"].(map[string]any); markdown != nil {
		if images, _ := markdown["images"].(map[string]any); images != nil {
			for _, value := range images {
				add(value)
			}
		}
	}
	if images, _ := page["outputImages"].(map[string]any); images != nil {
		for _, value := range images {
			add(value)
		}
	}
	return
}

func importLocalAssets(sourcePath, candidate string, pages []pageRecord, assets map[string]assetRecord) error {
	base, err := filepath.Abs(filepath.Dir(sourcePath))
	if err != nil {
		return err
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return err
	}
	var total int64
	for _, ref := range references(pages) {
		value := filepath.FromSlash(ref.get())
		if value == "" || isRemote(ref.get()) {
			continue
		}
		if filepath.IsAbs(value) {
			return coded("unsafe_asset", fmt.Sprintf("Page %d contains an absolute image path", ref.page+1))
		}
		path, err := filepath.Abs(filepath.Join(base, value))
		if err != nil {
			return err
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return coded("asset_not_found", fmt.Sprintf("Page %d references a missing local image", ref.page+1))
		}
		rel, err := filepath.Rel(base, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return coded("unsafe_asset", fmt.Sprintf("Page %d contains an image path outside the JSON directory", ref.page+1))
		}
		info, err := os.Lstat(path)
		if err != nil {
			return coded("asset_not_found", fmt.Sprintf("Page %d references a missing local image", ref.page+1))
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return coded("unsafe_asset", fmt.Sprintf("Page %d references an unsafe local image", ref.page+1))
		}
		if info.Size() > maxAssetBytes || total+info.Size() > maxAssetsBytes {
			return coded("asset_too_large", fmt.Sprintf("Page %d exceeds the image size limit", ref.page+1))
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		id, record, size, err := storeImage(candidate, file, maxAssetBytes)
		file.Close()
		if err != nil {
			return coded("invalid_asset", fmt.Sprintf("Page %d contains an invalid image: %v", ref.page+1, err))
		}
		total += size
		assets[id] = record
		ref.set("assets/" + id)
	}
	return nil
}

func downloadRemoteAssets(ctx context.Context, candidate string, pages []pageRecord, assets map[string]assetRecord, progress func(int, int, string)) (int, int, error) {
	refs := references(pages)
	jobs := make(chan imageReference)
	var wg sync.WaitGroup
	var mu sync.Mutex
	downloaded, failed := 0, 0
	var total int64
	for _, asset := range assets {
		if info, err := os.Stat(asset.Path); err == nil {
			total += info.Size()
		}
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return validatePublicURL(req.Context(), req.URL)
	}}
	worker := func() {
		defer wg.Done()
		for ref := range jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}
			value := ref.get()
			if !isRemote(value) {
				continue
			}
			u, err := url.Parse(value)
			if err == nil {
				err = validatePublicURL(ctx, u)
			}
			if err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
				continue
			}
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, value, nil)
			resp, err := client.Do(req)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				mu.Lock()
				failed++
				mu.Unlock()
				continue
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				resp.Body.Close()
				mu.Lock()
				failed++
				mu.Unlock()
				continue
			}
			mu.Lock()
			remaining := maxAssetsBytes - total
			mu.Unlock()
			id, record, size, err := storeImage(candidate, resp.Body, min(maxAssetBytes, remaining))
			resp.Body.Close()
			mu.Lock()
			if err != nil {
				failed++
			} else if _, exists := assets[id]; exists {
				ref.set("assets/" + id)
			} else if total+size > maxAssetsBytes {
				_ = os.Remove(record.Path)
				failed++
			} else {
				total += size
				downloaded++
				assets[id] = record
				ref.set("assets/" + id)
			}
			done := downloaded + failed
			mu.Unlock()
			if progress != nil {
				progress(done, len(refs), "Downloading images")
			}
		}
	}
	for range 6 {
		wg.Add(1)
		go worker()
	}
	send := true
	for _, ref := range refs {
		if !isRemote(ref.get()) {
			continue
		}
		select {
		case <-ctx.Done():
			send = false
		case jobs <- ref:
		}
		if !send {
			break
		}
	}
	close(jobs)
	wg.Wait()
	if ctx.Err() != nil {
		return 0, 0, ctx.Err()
	}
	return downloaded, failed, nil
}

func storeImage(candidate string, source io.Reader, limit int64) (string, assetRecord, int64, error) {
	if limit <= 0 {
		return "", assetRecord{}, 0, errors.New("the total image limit is exceeded")
	}
	temp, err := os.CreateTemp(filepath.Join(candidate, "assets"), ".asset-")
	if err != nil {
		return "", assetRecord{}, 0, err
	}
	name := temp.Name()
	defer os.Remove(name)
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(source, limit+1))
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", assetRecord{}, 0, err
	}
	if written > limit {
		return "", assetRecord{}, 0, errors.New("the image is larger than the limit")
	}
	file, err := os.Open(name)
	if err != nil {
		return "", assetRecord{}, 0, err
	}
	_, format, err := image.DecodeConfig(file)
	file.Close()
	if err != nil {
		return "", assetRecord{}, 0, errors.New("the file is not a supported image")
	}
	ext, mimeType := imageFormat(format)
	if ext == "" {
		return "", assetRecord{}, 0, errors.New("the image format is not supported")
	}
	id := hex.EncodeToString(hash.Sum(nil)) + ext
	target := filepath.Join(candidate, "assets", id)
	if _, err = os.Stat(target); os.IsNotExist(err) {
		if err = os.Rename(name, target); err != nil {
			if _, statErr := os.Stat(target); statErr != nil {
				return "", assetRecord{}, 0, err
			}
		}
	}
	return id, assetRecord{Path: target, MIME: mimeType}, written, nil
}
func imageFormat(format string) (string, string) {
	switch format {
	case "png":
		return ".png", "image/png"
	case "jpeg":
		return ".jpg", "image/jpeg"
	case "gif":
		return ".gif", "image/gif"
	case "webp":
		return ".webp", "image/webp"
	case "bmp":
		return ".bmp", "image/bmp"
	}
	return "", ""
}
func validatePublicURL(ctx context.Context, u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only HTTP and HTTPS are allowed")
	}
	if u.Hostname() == "" {
		return errors.New("the URL has no host")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, u.Hostname())
	if err != nil {
		return err
	}
	for _, address := range addresses {
		ip := address.IP
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return errors.New("private network addresses are not allowed")
		}
	}
	return nil
}
