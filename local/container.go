package local

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/graymeta/stow"
)

type container struct {
	name string
	path string
}

func (c *container) ID() string {
	return c.path
}

func (c *container) Name() string {
	return c.name
}

func (c *container) URL() *url.URL {
	return &url.URL{
		Scheme: "file",
		Path:   filepath.Clean(c.path),
	}
}

func (c *container) CreateItem(name string) (stow.Item, io.WriteCloser, error) {
	path := filepath.Join(c.path, filepath.FromSlash(name))
	item := &item{
		path:          path,
		contPrefixLen: len(c.path) + 1,
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return item, f, nil
}

func (c *container) RemoveItem(id string) error {
	return os.Remove(id)
}

func (c *container) Put(name string, r io.Reader, expectedSize int64, metadata map[string]interface{}) (stow.Item, error) {
	if len(metadata) > 0 {
		return nil, stow.NotSupported("metadata")
	}

	path := filepath.Join(c.path, filepath.FromSlash(name))
	item := &item{
		path:          path,
		contPrefixLen: len(c.path) + 1,
	}
	err := os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actualSize, err := io.Copy(f, r)
	if err != nil {
		return nil, err
	}

	if expectedSize > stow.SizeUnknown && actualSize != expectedSize {
		return nil, errors.Errorf("Put was told size was %d but actual stream size was %d", expectedSize, actualSize)
	}
	return item, nil
}

func (c *container) Items(prefix, cursor string, count int) ([]stow.Item, string, error) {
	prefix = filepath.FromSlash(prefix)
	files, err := flatdirs(c.path)
	if err != nil {
		return nil, "", err
	}
	if cursor != stow.CursorStart {
		// seek to the cursor
		ok := false
		for i, file := range files {
			if file.Name() == cursor {
				files = files[i:]
				ok = true
				break
			}
		}
		if !ok {
			return nil, "", stow.ErrBadCursor
		}
	}
	if len(files) > count {
		cursor = files[count].Name()
		files = files[:count]
	} else if len(files) <= count {
		cursor = "" // end
	}
	var items []stow.Item
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path, err := filepath.Abs(filepath.Join(c.path, f.Name()))
		if err != nil {
			return nil, "", err
		}
		if !strings.HasPrefix(f.Name(), prefix) {
			continue
		}
		item := &item{
			path:          path,
			contPrefixLen: len(c.path) + 1,
		}
		items = append(items, item)
	}
	return items, cursor, nil
}

func (c *container) Item(id string) (stow.Item, error) {
	path := id
	if !filepath.IsAbs(id) {
		path = filepath.Join(c.path, filepath.FromSlash(id))
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, stow.ErrNotFound
	}
	if info.IsDir() {
		return nil, errors.New("unexpected directory")
	}
	_, err = filepath.Rel(c.path, path)
	if err != nil {
		return nil, err
	}
	item := &item{
		path:          path,
		contPrefixLen: len(c.path) + 1,
	}
	return item, nil
}

// flatdirs walks the entire tree returning a list of
// os.FileInfo for all items encountered.
func flatdirs(path string) ([]os.FileInfo, error) {
	var list []os.FileInfo
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		flatname, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		list = append(list, fileinfo{
			FileInfo: info,
			name:     flatname,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

type fileinfo struct {
	os.FileInfo
	name string
}

func (f fileinfo) Name() string {
	return f.name
}
