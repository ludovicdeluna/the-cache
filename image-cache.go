/*
Package cache implements a simple library for image and other types files cache.
Author: Atom & Partners
License: MIT
*/

package cache

import (
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

type Image_cache struct {
	*Cache
	thumbPath string
}

func NewImageCache(maxItem, maxItemSize, maxSize int64, rootPath, thumbPath string) *Image_cache {
	return &Image_cache{NewCacheFile(maxItem, maxItemSize, maxSize, rootPath), thumbPath}
}

func (c *Image_cache) Handle(w http.ResponseWriter, r *http.Request, x, name string) {
	var id string
	i := strings.IndexByte(name, '.')
	if i > -1 {
		id = name[:i]
	} else {
		id = name
	}
	n := c.get(fmt.Sprintf("%s_%s", id, x))
	var rw *image_responseWriter
	if n.content == nil {
		rw = &image_responseWriter{responseWriter: &responseWriter{parent: n, cnf: c.Cache}, cnf: c, x: x, name: name}
		rw.content.contentLength = -1
		n.content = rw
	} else {
		rw = n.content.(*image_responseWriter)
	}
	rw.handle(w, r)
}

func (c *Image_cache) HandleEx(w http.ResponseWriter, r *http.Request, x, y, crop, name string) {
	var id string
	i := strings.IndexByte(name, '.')
	if i > -1 {
		id = name[:i]
	} else {
		id = name
	}
	n := c.get(fmt.Sprintf("%s_%sx%s-%s", id, x, y, crop))
	var rw *image_responseWriter
	if n.content == nil {
		rw = &image_responseWriter{responseWriter: &responseWriter{parent: n, cnf: c.Cache}, cnf: c, x: x, y: y, c: crop, name: name}
		rw.content.contentLength = -1
		n.content = rw
	} else {
		rw = n.content.(*image_responseWriter)
	}
	rw.handle(w, r)
}

func (c *Image_cache) Remove(key string) {
	c.tree.remove(key)
}

func (c *Image_cache) RemoveFiles(fileName interface{}) error {
	var file string
	switch fileName.(type) {
	case string:
		file = fileName.(string)
		i := strings.IndexByte(file, '.')
		if i > -1 { // remove ext
			file = file[:i]
		}
	case int:
		file = strconv.Itoa(fileName.(int))
	case int64:
		file = strconv.FormatInt(fileName.(int64), 10)
	default:
		return fmt.Errorf("unknown tyoe %v", fileName)
	}
	files, err := ioutil.ReadDir(c.thumbPath)
	if err == nil {
		for _, f := range files {
			name := f.Name()
			if strings.Index(name, file) == 0 {
				if err = os.Remove(c.thumbPath + name); err != nil {
					return err
				}
				i := strings.IndexByte(name, '.')
				if i > -1 { // remove ext
					name = name[:i]
				}
				c.tree.remove(name)
			}
		}
	}
	return err
}

type image_responseWriter struct {
	*responseWriter
	cnf           *Image_cache
	x, y, c, name string
}

func (rw *image_responseWriter) handle(w http.ResponseWriter, r *http.Request) {
	if rw.code == http.StatusNotFound {
		w.WriteHeader(rw.code)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=604800")
	if rw.valid() {
		//log.Println("cached", rw.getFilePath())
		w.Header().Set("Content-Type", rw.contentType)
		
		// If the file is in the cache it uses the http.ServeContent method.
		http.ServeContent(w, r, "", rw.modtime, &rw.content)
	} else {
		//log.Println("empty", rw.getFilePath())
		rw.w = w
		
		// If the file is not in the cache yet http.ServeFile used to save it into the cache.
		http.ServeFile(rw, r, rw.getFilePath())
	}
}

func (rw *image_responseWriter) getFilePath() string {
	var filePath string
	var err error
	limit := func(s *string) int {
		var i int
		i, err = strconv.Atoi(*s)
		if err == nil {
			if i > 19 {
				i = 19
				*s = "19"
			} else if i < 3 {
				i = 3
				*s = "3"
			}
		}
		return i * 100
	}
	x := limit(&rw.x)
	var y int
	if err == nil {
		if rw.y != "" {
			y = limit(&rw.y)
		}
		if err == nil {
			filePath = rw.cnf.thumbPath + rw.parent.key
			i := strings.IndexByte(rw.name, '.')
			if i > -1 {
				filePath += rw.name[i:]
			}
			if _, err = os.Stat(filePath); os.IsNotExist(err) {
				var img image.Image
				img, err = imaging.Open(rw.cnf.rootPath + rw.name)
				if err == nil {
					if y > 0 {
						if rw.c == "1" {
							img = imaging.Fill(img, x, y, imaging.Center, imaging.Lanczos)
						} else {
							img = imaging.Fit(img, x, y, imaging.Lanczos)
						}
					} else {
						img = imaging.Fit(img, x, x, imaging.Lanczos)
					}
					err = imaging.Save(img, filePath)
				}
			}
			if err != nil {
				log.Println("ImageCache", err)
				filePath = ""
			}
		}
	}
	return filePath
}
