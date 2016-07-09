/*
Package cache implements a simple library for image and other types files cache.
Author: Atom & Partners
License: MIT
*/

package cache

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

type Cache struct {
	*tree
	maxItemSize, maxSize, size int64
	rootPath                   string
}

func NewCacheFile(maxItem, maxItemSize, maxSize int64, rootPath string) *Cache {
	return &Cache{tree: newTree(maxItem), maxItemSize: maxItemSize * 1000000, maxSize: maxSize * 1000000, rootPath: rootPath}
}

func (c *Cache) Handle(w http.ResponseWriter, r *http.Request, path string) {
	n := c.get(path)
	var rw *responseWriter
	if n.content == nil {
		rw = &responseWriter{parent: n, cnf: c}
		rw.content.contentLength = -1
		n.content = rw
	} else {
		rw = n.content.(*responseWriter)
	}
	rw.handle(w, r)
}

func (c *Cache) Remove(path string) {
	//log.Println("remove", path)
	c.tree.remove(path)
}

type content struct {
	io.ReadSeeker
	contentLength, offset int64
	bb                    []byte
}

func (c *content) Read(p []byte) (int, error) {
	n := 0
	var err error
	if c.contentLength > 0 {
		l := int64(len(p))
		if l > 0 {
			size := c.offset + l
			if size <= c.contentLength {
				n = copy(p, c.bb[c.offset:size])
				if n != int(l) {
					c.offset += int64(n)
					err = errors.New(fmt.Sprintf("Written bytes %d less than all bytes %d", n, l))
				} else {
					c.offset = size
				}
			} else {
				err = errors.New(fmt.Sprintf("ContentLength %d smaller than written bytes %d", c.contentLength, size))
			}
		}
	} else {
		err = errors.New("Content not initialized ContentLength < 1")
	}
	if err != nil {
		log.Println(err)
	}
	return n, err
}

func (c *content) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case os.SEEK_SET:
		c.offset = offset
	case os.SEEK_CUR:
		c.offset += offset
	case os.SEEK_END:
		c.offset = c.contentLength - offset
	}
	//log.Println(whence, "Seek", c.offset)
	return c.offset, nil
}

type responseWriter struct {
	tree_content
	cnf               *Cache
	parent            *node
	w                 http.ResponseWriter
	content           content
	contentType, name string
	modtime           time.Time
	code, size        int
}

func (w *responseWriter) valid() bool {
	return w.size > 0 && len(w.content.bb) == w.size
}

func (w *responseWriter) delete() {
	//w.cnf.size -= int64(len(w.content.bb))
	//log.Println("delete", w.cnf.size, len(w.content.bb))
	if w.content.contentLength > 0 {
		atomic.AddInt64(&w.cnf.size, w.content.contentLength*-1)
		w.content.contentLength = -1
		w.content.bb = nil
	}
	//	w.w = nil
	//w.parent = nil
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.content.contentLength > 0 {
		//log.Println("saved", w.size, "load", size)
		w.size += copy(w.content.bb[w.size:], b)
		/*
			if w.size != len(w.content.bb) {
				log.Println("part", w.parent.key, w.size, len(w.content.bb))
			} else {
				log.Println("done", w.parent.key)
			}
		*/
	}
	return w.w.Write(b)
}

func (w *responseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *responseWriter) WriteHeader(i int) {
	w.code = i
	//log.Println("WriteHeader", w.parent.key)
	if i == http.StatusOK && w.content.contentLength == -1 {
		mutex.Lock()
		h := w.Header()
		var err error
		w.content.contentLength, err = strconv.ParseInt(h.Get("Content-Length"), 10, 64)
		if err == nil {
			w.contentType = h.Get("Content-Type")
			if w.cnf.maxItemSize >= w.content.contentLength {
				w.modtime, _ = time.Parse(http.TimeFormat, h.Get("Last-Modified"))
				atomic.AddInt64(&w.cnf.size, w.content.contentLength)
				if w.cnf.size > w.cnf.maxSize {
					log.Printf("Clear MaxSize=%d, Size=%d", w.cnf.maxSize, w.cnf.size)
					w.parent.tree.clear(w.parent)
				}
				w.content.bb = make([]byte, int(w.content.contentLength))
			}
		} else {
			w.content.contentLength = 0
		}
		mutex.Unlock()
		//log.Println("make", w.parent.key, size)
		//log.Println(w.parent.key, w.modtime.String())
		//log.Println(w.Header())
		// Cache-Control:[public, max-age=604800] Last-Modified:[Thu, 03 Dec 2015 17:46:28 GMT] Content-Type:[image/jpeg] Accept-Ranges:[bytes] Content-Length:[290849]
	}
	w.w.WriteHeader(i)
}

func (rw *responseWriter) handle(w http.ResponseWriter, r *http.Request) {
	if rw.code == http.StatusNotFound {
		//	log.Println("not found", path)
		w.WriteHeader(rw.code)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=604800")
	if rw.valid() {
		w.Header().Set("Content-Type", rw.contentType)
		//log.Println("cached", rw.parent.key)


		// If the file is in the cache it uses the http.ServeContent method.
		http.ServeContent(w, r, "", rw.modtime, &rw.content)
	} else {
		//log.Println("empty", rw.parent.key)
		rw.w = w

		// If the file is not in the cache yet http.ServeFile used to save it into the cache.
		http.ServeFile(rw, r, rw.cnf.rootPath+rw.parent.key)
	}
}
