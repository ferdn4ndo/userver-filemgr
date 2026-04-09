package media_processor

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const jpegQualityResized = 90

// imageBox is a target bounding box from media_convert_configuration.image_resizer.sizes (legacy Django PR #1).
type imageBox struct {
	W, H int
}

// parseImageResizerSizes reads {"image_resizer":{"sizes":["1920x1080",...]}} or objects with width/height.
func parseImageResizerSizes(cfg json.RawMessage) ([]imageBox, error) {
	if len(cfg) == 0 || string(cfg) == "null" {
		return nil, nil
	}
	var top struct {
		ImageResizer *struct {
			Sizes json.RawMessage `json:"sizes"`
		} `json:"image_resizer"`
	}
	if err := json.Unmarshal(cfg, &top); err != nil {
		return nil, err
	}
	if top.ImageResizer == nil || len(top.ImageResizer.Sizes) == 0 {
		return nil, nil
	}
	var asStrings []string
	if err := json.Unmarshal(top.ImageResizer.Sizes, &asStrings); err == nil && len(asStrings) > 0 {
		return parseSizeStrings(asStrings), nil
	}
	var asAny []any
	if err := json.Unmarshal(top.ImageResizer.Sizes, &asAny); err != nil {
		return nil, err
	}
	var out []imageBox
	for _, v := range asAny {
		switch t := v.(type) {
		case string:
			b, err := parseOneSizeString(t)
			if err == nil {
				out = append(out, b)
			}
		case map[string]any:
			w := numFromAny(t["width"])
			h := numFromAny(t["height"])
			if w > 0 && h > 0 {
				out = append(out, imageBox{W: w, H: h})
			}
		}
	}
	return out, nil
}

func parseSizeStrings(ss []string) []imageBox {
	var out []imageBox
	for _, s := range ss {
		b, err := parseOneSizeString(s)
		if err == nil {
			out = append(out, b)
		}
	}
	return out
}

func parseOneSizeString(s string) (imageBox, error) {
	s = strings.TrimSpace(strings.ToLower(strings.ReplaceAll(s, " ", "")))
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return imageBox{}, fmt.Errorf("bad size %q", s)
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return imageBox{}, fmt.Errorf("bad size %q", s)
	}
	return imageBox{W: w, H: h}, nil
}

func numFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(t))
		return i
	default:
		return 0
	}
}

// computeNewImageDimensions matches legacy Django MediaImageProcessorService (fit inside expectedW x expectedH, preserve aspect).
func computeNewImageDimensions(origW, origH, expectedW, expectedH int) (rw, rh int) {
	var factor float64
	if origH > origW {
		factor = float64(expectedW) / float64(origW)
	} else {
		factor = float64(expectedH) / float64(origH)
	}
	return int(math.Round(float64(origW) * factor)), int(math.Round(float64(origH) * factor))
}

// sizeTagFromDimensions maps output pixel size to legacy SIZE_* tags (Django get_size_tag_from_dimensions).
func sizeTagFromDimensions(width, height int) string {
	if width >= 8192 && height >= 5472 {
		return "SIZE_8K"
	}
	if width >= 4096 && height >= 2752 {
		return "SIZE_4K"
	}
	if width >= 3200 && height >= 2144 {
		return "SIZE_3K"
	}
	if width >= 2048 && height >= 1376 {
		return "SIZE_2K"
	}
	if width >= 1280 && height >= 864 {
		return "SIZE_1K"
	}
	return "SIZE_VGA"
}
