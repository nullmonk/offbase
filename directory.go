package offbase

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type Directory struct {
	// Directory is a struct that represents a directory
	// on the CMS
	Name        string
	ID          string
	ParentID    string
	Parent      *Directory
	Files       []*File
	Directories []*Directory
}

func NewDirectory(name, id string) *Directory {
	return &Directory{
		Name:        strings.TrimSpace(name),
		ID:          strings.TrimSpace(id),
		Files:       make([]*File, 0),
		Directories: make([]*Directory, 0),
	}
}

func NewDirectoryFromURL(name, urlStr string) (*Directory, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	id := u.Query().Get("FolderID")
	if id == "" {
		return nil, fmt.Errorf("could not find FolderID in URL")
	}

	return &Directory{
		Name:        strings.TrimSpace(name),
		ID:          strings.TrimSpace(id),
		Files:       make([]*File, 0),
		Directories: make([]*Directory, 0),
	}, nil
}

func (d *Directory) String() string {
	if d.ParentID != "" {
		if len(d.Files) > 0 {
			return fmt.Sprintf("%s/%s (%s) - %d files", d.ParentID, d.Name, d.ID, len(d.Files))
		} else {
			return fmt.Sprintf("%s/%s (%s)", d.ParentID, d.Name, d.ID)
		}
	} else {
		if len(d.Files) > 0 {
			return fmt.Sprintf("[ROOT]/%s (%s) - %d files", d.Name, d.ID, len(d.Files))
		} else {
			return fmt.Sprintf("[ROOT]/%s (%s)", d.Name, d.ID)
		}
	}
}

func (d *Directory) FullPath() string {
	if d.Parent == nil {
		return d.Name
	}
	return filepath.Join(d.Parent.FullPath(), d.Name)
}

func (d *Directory) URL(u *url.URL) string {
	u2 := *u
	u2.Path = "/GetFolder"
	u2.RawQuery = fmt.Sprintf("FolderID=%s", d.ID)
	return u2.String()
}

func (d *Directory) PrintTree(indent string) {
	fmt.Println(indent + d.Name)
	for _, child := range d.Directories {
		child.PrintTree(indent + "  ")
	}
	for _, file := range d.Files {
		fmt.Println(indent + "  " + file.Name)
	}
}
