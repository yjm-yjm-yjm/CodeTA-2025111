package covergen

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
)

// FromOfficeZIP 从 pptx/docx 中抽取首页相关图片作为封面（PNG/JPEG 原样返回）。
// 返回 contentType, bytes。
func FromOfficeZIP(fileType string, r io.ReaderAt, size int64) (string, []byte, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return "", nil, err
	}
	ft := strings.ToLower(strings.TrimPrefix(fileType, "."))
	switch ft {
	case "pptx", "ppt":
		return fromPPTX(zr)
	case "docx", "doc":
		return fromDOCX(zr)
	default:
		return "", nil, fmt.Errorf("unsupported type %s", fileType)
	}
}

func fromPPTX(zr *zip.Reader) (string, []byte, error) {
	rels := map[string]string{}
	if f, err := zr.Open("ppt/slides/_rels/slide1.xml.rels"); err == nil {
		defer f.Close()
		type Relationship struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		}
		type Rels struct {
			Relationship []Relationship `xml:"Relationship"`
		}
		var doc Rels
		if err := xml.NewDecoder(f).Decode(&doc); err == nil {
			for _, rel := range doc.Relationship {
				t := rel.Target
				if strings.HasPrefix(t, "/") {
					t = strings.TrimPrefix(t, "/")
				} else if strings.HasPrefix(t, "../") {
					t = "ppt/" + strings.TrimPrefix(t, "../")
				} else if !strings.HasPrefix(t, "ppt/") {
					t = path.Clean("ppt/slides/" + t)
					t = strings.ReplaceAll(t, "\\", "/")
				}
				rels[rel.ID] = t
			}
		}
	}
	if f, err := zr.Open("ppt/slides/slide1.xml"); err == nil {
		defer f.Close()
		raw, _ := io.ReadAll(f)
		// 粗解析 r:embed
		const mark = `r:embed="`
		s := string(raw)
		for {
			i := strings.Index(s, mark)
			if i < 0 {
				break
			}
			s = s[i+len(mark):]
			j := strings.IndexByte(s, '"')
			if j < 0 {
				break
			}
			rid := s[:j]
			s = s[j+1:]
			if target, ok := rels[rid]; ok {
				if ct, b, err := readMedia(zr, target); err == nil {
					return ct, b, nil
				}
			}
		}
	}
	return largestMedia(zr, "ppt/media/")
}

func fromDOCX(zr *zip.Reader) (string, []byte, error) {
	if ct, b, err := readMedia(zr, "word/media/image1.png"); err == nil {
		return ct, b, nil
	}
	if ct, b, err := readMedia(zr, "word/media/image1.jpeg"); err == nil {
		return ct, b, nil
	}
	if ct, b, err := readMedia(zr, "word/media/image1.jpg"); err == nil {
		return ct, b, nil
	}
	return largestMedia(zr, "word/media/")
}

func readMedia(zr *zip.Reader, name string) (string, []byte, error) {
	f, err := zr.Open(name)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return "", nil, err
	}
	return contentTypeByExt(name), b, nil
}

func largestMedia(zr *zip.Reader, prefix string) (string, []byte, error) {
	var best *zip.File
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		ext := strings.ToLower(path.Ext(f.Name))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
			continue
		}
		if best == nil || f.UncompressedSize64 > best.UncompressedSize64 {
			best = f
		}
	}
	if best == nil {
		return "", nil, fmt.Errorf("no media image")
	}
	rc, err := best.Open()
	if err != nil {
		return "", nil, err
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return "", nil, err
	}
	return contentTypeByExt(best.Name), b, nil
}

func contentTypeByExt(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// EnsureReaderAt 把流读入内存以便 zip 随机读。
func EnsureReaderAt(r io.Reader) (*bytes.Reader, int64, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, err
	}
	return bytes.NewReader(b), int64(len(b)), nil
}
