// Command synccli mirrors a local directory to a Sapphire SFTP server over the
// REST API — the "desktop sync" agent. It walks the folder recursively,
// recreates the tree remotely, and uploads any file that is new or changed
// (compared by SHA-256). Re-uploading a changed file creates a new version on
// the server. With -watch it stays running and syncs on change.
//
// Auth uses an API key (generate one in the web app → API Keys).
//
//	synccli -server http://localhost -key sftp_xxx -dir ~/Documents/work
//	synccli -server http://localhost -key sftp_xxx -dir ~/work -watch
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type client struct {
	base   string
	key    string
	http   *http.Client
	dirIDs map[string]string // rel dir path ("" = root) -> remote folder id
}

type listing struct {
	Data struct {
		Folders []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"folders"`
		Files []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Checksum string `json:"checksum_sha256"`
		} `json:"files"`
	} `json:"data"`
}

func main() {
	var (
		server = flag.String("server", envOr("SFTP_SERVER", "http://localhost"), "server base URL")
		key    = flag.String("key", os.Getenv("SFTP_API_KEY"), "API key (sftp_...)")
		dir    = flag.String("dir", "", "local directory to mirror")
		watch  = flag.Bool("watch", false, "keep running and sync on change")
	)
	flag.Parse()

	if *key == "" || *dir == "" {
		log.Fatal("usage: synccli -server URL -key sftp_xxx -dir /path [-watch]")
	}
	abs, err := filepath.Abs(*dir)
	if err != nil || !isDir(abs) {
		log.Fatalf("not a directory: %s", *dir)
	}

	c := &client{
		base:   strings.TrimRight(*server, "/") + "/api/v1",
		key:    *key,
		http:   &http.Client{Timeout: 0},
		dirIDs: map[string]string{"": ""},
	}

	log.Printf("mirroring %s → %s", abs, *server)
	if err := c.syncAll(abs); err != nil {
		log.Fatalf("sync failed: %v", err)
	}
	log.Println("initial sync complete")

	if *watch {
		c.watch(abs)
	}
}

// syncAll walks the tree and uploads new/changed files.
func (c *client) syncAll(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") || info.Name() == ".sftpsync" {
			return nil // skip dotfiles
		}
		rel, _ := filepath.Rel(root, path)
		if err := c.syncFile(root, rel); err != nil {
			log.Printf("  ! %s: %v", rel, err)
		}
		return nil
	})
}

// syncFile uploads one file if it is new or its checksum differs from the server.
func (c *client) syncFile(root, rel string) error {
	relDir := filepath.Dir(rel)
	if relDir == "." {
		relDir = ""
	}
	folderID, err := c.ensureDir(relDir)
	if err != nil {
		return err
	}
	local, err := checksumFile(filepath.Join(root, rel))
	if err != nil {
		return err
	}
	remote, err := c.listDir(folderID)
	if err != nil {
		return err
	}
	name := filepath.Base(rel)
	for _, f := range remote.Data.Files {
		if f.Name == name && strings.EqualFold(f.Checksum, local) {
			return nil // unchanged
		}
	}
	if err := c.upload(filepath.Join(root, rel), name, folderID); err != nil {
		return err
	}
	log.Printf("  ↑ %s", rel)
	return nil
}

// ensureDir resolves (creating as needed) the remote folder id for a rel path.
func (c *client) ensureDir(relDir string) (string, error) {
	relDir = filepath.ToSlash(relDir)
	if id, ok := c.dirIDs[relDir]; ok {
		return id, nil
	}
	parent := ""
	name := relDir
	if i := strings.LastIndex(relDir, "/"); i >= 0 {
		parent, name = relDir[:i], relDir[i+1:]
	}
	parentID, err := c.ensureDir(parent)
	if err != nil {
		return "", err
	}
	// Reuse an existing folder if present.
	if l, err := c.listDir(parentID); err == nil {
		for _, f := range l.Data.Folders {
			if f.Name == name {
				c.dirIDs[relDir] = f.ID
				return f.ID, nil
			}
		}
	}
	id, err := c.createFolder(name, parentID)
	if err != nil {
		return "", err
	}
	c.dirIDs[relDir] = id
	return id, nil
}

func (c *client) listDir(folderID string) (*listing, error) {
	url := c.base + "/files/?limit=1000"
	if folderID != "" {
		url += "&folder_id=" + folderID
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list: %s", resp.Status)
	}
	var l listing
	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (c *client) createFolder(name, parentID string) (string, error) {
	body := map[string]any{"name": name}
	if parentID != "" {
		body["parent_id"] = parentID
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, c.base+"/folders/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data.ID == "" {
		return "", fmt.Errorf("create folder %q: %s", name, resp.Status)
	}
	return out.Data.ID, nil
}

func (c *client) upload(path, name, folderID string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		if folderID != "" {
			_ = mw.WriteField("folder_id", folderID)
		}
		part, err := mw.CreateFormFile("file", name)
		if err != nil {
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			return
		}
		_ = mw.Close()
	}()

	req, _ := http.NewRequest(http.MethodPost, c.base+"/files/upload", pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("upload %s: %s: %s", name, resp.Status, string(b))
	}
	return nil
}

// watch re-syncs changed files (debounced) until interrupted.
func (c *client) watch(root string) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
			_ = w.Add(p)
		}
		return nil
	})

	log.Println("watching for changes… (Ctrl+C to stop)")
	pending := map[string]time.Time{}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if isDir(ev.Name) {
					_ = w.Add(ev.Name)
					continue
				}
				pending[ev.Name] = time.Now()
			}
		case <-tick.C:
			now := time.Now()
			for p, t := range pending {
				if now.Sub(t) < 700*time.Millisecond {
					continue // debounce
				}
				delete(pending, p)
				if rel, err := filepath.Rel(root, p); err == nil && fileExists(p) {
					if err := c.syncFile(root, rel); err != nil {
						log.Printf("  ! %s: %v", rel, err)
					}
				}
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Printf("watch error: %v", err)
		}
	}
}

func (c *client) auth(r *http.Request) { r.Header.Set("X-API-Key", c.key) }

func checksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isDir(p string) bool     { i, err := os.Stat(p); return err == nil && i.IsDir() }
func fileExists(p string) bool { i, err := os.Stat(p); return err == nil && !i.IsDir() }

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
