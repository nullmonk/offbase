package offbase

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
)

type Scraper struct {
	baseurl     *url.URL
	destination string
	collector   *colly.Collector
	directories sync.Map
	files       sync.Map
	rootID      string
}

func NewScraper(urlStr, destination string) (*Scraper, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	domain := parsedURL.Hostname()

	c := colly.NewCollector(
		colly.AllowedDomains(domain),
		colly.IgnoreRobotsTxt(),
		colly.Async(true),
		colly.CacheDir("./cache"),
	)
	// Suppress error messages
	c.OnError(func(_ *colly.Response, _ error) {
		// Do nothing
	})
	c.SetRequestTimeout(30 * time.Second)

	return &Scraper{
		baseurl:     parsedURL,
		destination: destination,
		collector:   c,
	}, nil
}

func (s *Scraper) Scrape() error {
	// Parse directories from the /GetFolder page
	s.collector.OnHTML("tbody#docTableBody tr", func(e *colly.HTMLElement) {
		d := NewDirectory(e.ChildText("td a"), e.ChildText("td.hidden"))
		d.ParentID = e.Request.URL.Query().Get("FolderID")
		s.visitDirectory(d)
	})

	// Parse directories from the root page
	s.collector.OnHTML("i.fa-folder-o + a", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		// Parent ID is unknown for root directories
		d, err := NewDirectoryFromURL(e.Text, href)
		if err != nil {
			return
		}
		if _, loaded := s.directories.LoadOrStore(d.ID, d); !loaded {
			// Recursively scrape the new directory
			s.visitDirectory(d)
		}
	})

	s.collector.OnResponse(func(r *colly.Response) {
		// Parse the files from the /PublicAccessProvider // getDocHitList page
		// This endpoints returns each file in a directory in an XML format
		if r.Request.URL.Path == "/PublicAccessProvider.ashx" {
			bd := string(r.Body)
			folderId := r.Request.URL.Query().Get("folderID")
			files, fetchErr := ParseFilesFromResponse(folderId, bd)
			if fetchErr != nil {
				return
			}
			var dir *Directory
			dirRaw, ok := s.directories.Load(folderId)
			if !ok {
				dir = NewDirectory("[UNKNOWN]", folderId)
				s.directories.Store(folderId, dir)
			} else {
				dir = dirRaw.(*Directory)
			}

			for _, f := range files {
				f.Parent = dir
				s.files.Store(f.ID, f)
				//dir.Files = append(dir.Files, f)
				//fmt.Println("Found file: ", f.FullPath())
				s.collector.Visit(f.URL(s.baseurl))
			}
		} else if r.Request.URL.Path == "/PDFProvider.ashx" {
			//GetDocument is a file download endpoint
			// TODO: Actually save
			//fmt.Println(string(r.Body))
			pth, ok := s.files.Load(r.Request.URL.Query().Get("docID"))
			if !ok {
				return
			}

			f := pth.(*File).FullPath()

			if err := os.MkdirAll(filepath.Dir(f), 0755); err != nil {
				fmt.Printf("[!] Could not make directory: %s: %s\n", f, err)
				return
			}
			fil, err := os.Create(f)
			if err != nil {
				fmt.Printf("[!] Could not create file: %s: %s\n", f, err)
				return
			}
			defer fil.Close()
			fmt.Println("Saved file: ", f)
			fil.Write(r.Body)
		}
	})

	d, _ := NewDirectoryFromURL("", s.baseurl.String())
	if d.ID != "" {
		// Visit the root directory bc we have the ID
		s.rootID = d.ID
		s.visitDirectory(d)
	} else {
		s.collector.Visit(s.baseurl.String())
	}
	s.collector.Wait()
	return nil
}

/* Organize the directories into a tree structure based on the parent ID */
func (s *Scraper) GetRootDirectory() (*Directory, error) {
	root := NewDirectory("", "")
	if s.rootID != "" {
		if d, ok := s.directories.Load(s.rootID); ok {
			root = d.(*Directory)
		}
	}
	dirMap := make(map[string]*Directory)
	s.directories.Range(func(_, v interface{}) bool {
		d := v.(*Directory)
		dirMap[d.ID] = d
		return true
	})
	for _, d := range dirMap {
		if d.ID == s.rootID {
			continue
		}
		if parent, ok := dirMap[d.ParentID]; ok {
			//parent.Directories = append(parent.Directories, d)
			d.Parent = parent
		} else {
			//root.Directories = append(root.Directories, d)
		}
	}
	return root, nil
}

func (s *Scraper) GetFiles() []*File {
	files := make([]*File, 0)
	s.files.Range(func(_, v interface{}) bool {
		files = append(files, v.(*File))
		return true
	})
	return files
}

/* Visit a directory and scrape its contents if it has not been collected yet */
func (s *Scraper) visitDirectory(d *Directory) {
	if _, loaded := s.directories.LoadOrStore(d.ID, d); !loaded {
		// Recursively scrape the new directory
		s.collector.Visit(d.URL(s.baseurl))

		u := *s.baseurl
		u.Path = "/PublicAccessProvider.ashx"
		vals := url.Values{}

		vals.Add("action", "getDocHitList")
		vals.Add("folderID", d.ID)
		vals.Add("EncryptFolderID", "False")
		u.RawQuery = vals.Encode()
		s.collector.Request("POST", u.String(), strings.NewReader(u.Query().Encode()), nil, nil)
	}
}
