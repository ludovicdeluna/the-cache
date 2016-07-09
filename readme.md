The Cache - A file cache developed in Go
========

One http request uses storage three times for a file:
* once for the modification date
* again to determine file’s type
* and then to read the file content

The essence of The Cache is that all of these datas are stored in a memory (in a binary tree) without using storage.

Our solution is **thread safe** and optimal for a few thousand frequently used photos and other files.


### How The Cache works

* image cache: Resize the images by the given URL parameters (width / height / exact crop) and store the thumbnails in a binary tree (best for JPG and PNG). The original image isn’t even necessary to be stored in the cache.
* file cache: Works in the same way as image cache but stores the original file (good for SVG, GIF etc.)

When the memory reaches a previously set limit, it deletes the oldest items which weren’t used from the binary tree.
Hereby the most frequently used items get to the beginning of the binary tree, so they become reachable faster.

If the file is not in the cache yet **http.ServeFile** method is used to save it into the cache.
If the file is in the cache it uses the **http.ServeContent** method.

* dependency: github.com/disintegration/imaging
* also works well with github.com/julienschmidt/httprouter


### Example

```go

import (
"net/http"
"github.com/julienschmidt/httprouter"
)

var (

// Configure: items number, individual file size, cache size, the path of the original images, the path of the thumbnails (existing folder required)
CacheImg = cache.NewImageCache(maxItems, maxImgSize, maxCacheSize, imagePath,, thumbPath )

// Configure: same without thumbnail folder path
CacheFile = cache.NewCacheFile(maxItems, maxFileSize, maxCacheSize, rootPath)
)

// file mode example
func File(w http.ResponseWriter, r *http.Request, rp httprouter.Params) {
CacheFile.Handle(w, r, rp.ByName("name"))
}

// simple image mode example (uses only one parameter for width and height: value multiplied by 100, no crop)
func Img(w http.ResponseWriter, r *http.Request, rp httprouter.Params) {
CacheImg.Handle(w, r, rp.ByName("size"), rp.ByName("name"))
}

// advanced image mode example (3 parameters: width, height: value multiplied by 100, and crop: 1 = crop to the exact size, 0 = fit into the requested size)
func ImgEx(w http.ResponseWriter, r *http.Request, rp httprouter.Params) {
CacheImg.HandleEx(w, r, rp.ByName("x"), rp.ByName("y"), rp.ByName("corp"), rp.ByName("name"))
}

// Remove files exmaple (fileName = the removable file's name with or without extension or a number if the file name an ID from the database)
CacheImg.RemoveFiles(fileName)

```

## License 

(The MIT License)

Copyright (c) 2016 Atom & Partners question&atom.partners

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.