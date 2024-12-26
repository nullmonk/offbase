package offbase

import (
	"encoding/xml"
	"net/url"
	"path/filepath"
	"strings"
)

// ...existing code...

// ...existing code...

type File struct {
	Name     string
	ID       string
	Parent   *Directory
	ParentID string
}

func (f *File) URL(u *url.URL) string {
	u2 := *u
	u2.Path = "/PDFProvider.ashx"
	ext := filepath.Ext(f.Name)
	ext = strings.Trim(ext, ".")
	name := strings.TrimSuffix(filepath.Base(f.Name), ext)
	q := u2.Query()
	q.Del("FolderID")
	q.Set("action", "PDFStream")
	q.Set("docID", f.ID)
	q.Set("docName", name)
	q.Set("nativeExt", strings.ToUpper(ext))
	q.Set("PromptToSave", "True")
	q.Set("ViewerMode", "1")
	u2.RawQuery = q.Encode()
	return u2.String()
}

func (f *File) FullPath() string {
	if f.Parent == nil {
		return f.Name
	} else {
		return filepath.Join(f.Parent.FullPath(), f.Name)
	}
}

// Used for the DocHitList response
type Request struct {
	DocumentCollection struct {
		Documents []struct {
			ID          string `xml:"ID"`
			Name        string `xml:"Name"`
			DisplayType string `xml:"DisplayType"`
		} `xml:"Document"`
	} `xml:"DocumentCollection"`
}

func ParseFilesFromResponse(folderId, response string) ([]*File, error) {
	var req Request
	err := xml.Unmarshal([]byte(response), &req)
	if err != nil {
		return nil, err
	}

	var files []*File

	for _, doc := range req.DocumentCollection.Documents {
		file := &File{
			Name:     doc.Name,
			ID:       doc.ID,
			ParentID: folderId,
		}
		files = append(files, file)
	}

	return files, nil
}
