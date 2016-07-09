/*
Package cache implements a simple library for image and other types files cache.
Author: Atom & Partners
License: MIT
*/

package cache

import (
	"log"
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

type tree struct {
	maxItem, items int64
	root           *node
}

type tree_content interface {
	delete()
	value() interface{}
}

func newTree(maxItem int64) *tree {
	return &tree{maxItem: maxItem}
}

type node struct {
	tree *tree
	key  string
	//key     interface{}
	count   int
	parent  *node
	left    *node
	right   *node
	content tree_content
}

func (n *node) delete() {
	atomic.AddInt64(&n.tree.items, -1)
	n.tree = nil
	n.parent = nil
	n.left = nil
	n.right = nil
	n.content.delete()
	n.content = nil
	//	n.key = ""
}

var mutex = sync.Mutex{}

func (t *tree) addUsage(root *node, ls *[]int) {
	if root != nil {
		*ls = append(*ls, root.count)
		t.addUsage(root.right, ls)
		t.addUsage(root.left, ls)
	}
}

func (t *tree) removeNotUsed(root *node, min int, exclude *node) {
	if root != nil {
		t.removeNotUsed(root.left, min, exclude)
		t.removeNotUsed(root.right, min, exclude)
		if root != exclude && root.count <= min {
			t.removeNode(root)
		}
		if root.count > 9 {
			root.count = int(math.Sqrt(float64(root.count + 100))) // reduce counter
		}
	}
}

func (t *tree) clear(exclude *node) {
	//log.Printf("root1 %+v", c.root)
	var ls []int
	t.addUsage(t.root, &ls)
	sort.Ints(ls)
	//log.Println(ls)
	log.Printf("removeNotUsed count=%d", ls[len(ls)/10])
	t.removeNotUsed(t.root, ls[len(ls)/10], exclude)
	//log.Printf("root2 %+v", c.root)
}

func (t *tree) getNew(key string, parent *node) *node {
	//log.Println("add", key)
	n := &node{key: key, parent: parent, tree: t}
	atomic.AddInt64(&t.items, 1)
	if t.maxItem < t.items {
		log.Printf("Clear MaxItem=%d, Items=%d", t.maxItem, t.items)
		t.clear(parent)
	}
	return n
}

func (t *tree) getKey(root *node, key string) *node {
	if root.key == key {
		//   atomic.AddInt64(&n.count, 1)
		root.count++
		return root
	}
	if root.key < key {
		if root.right != nil {
			return t.getKey(root.right, key)
		}
		root.right = t.getNew(key, root)
		return root.right
	}
	if root.left != nil {
		return t.getKey(root.left, key)
	}
	root.left = t.getNew(key, root)
	return root.left
}

func (t *tree) get(key string) *node {
	//log.Println("get", key)
	mutex.Lock()
	if t.root == nil {
		t.root = t.getNew(key, nil)
		mutex.Unlock()
		return t.root
	}
	n := t.getKey(t.root, key)
	mutex.Unlock()
	return n
}

func (n *node) insert(node *node) {
	if node != nil {
		if n.key < node.key {
			if n.right == nil {
				node.parent = n
				n.right = node
			} else {
				n.right.insert(node)
			}
		} else {
			if n.left == nil {
				node.parent = n
				n.left = node
			} else {
				n.left.insert(node)
			}
		}
		//log.Printf("inserted %+v", n)
	}
}

func (t *tree) removeNode(n *node) {
	//	log.Printf("remove %+v", n)
	if t.root == n {
		if n.left != nil {
			n.left.parent = nil
			t.root = n.left
			t.root.insert(n.right)
			n.delete()
		} else if n.right != nil {
			n.right.parent = nil
			t.root = n.right
			t.root.insert(n.left)
			n.delete()
		} // else single root node do not delete
	} else if n.parent != nil {
		if n.parent.left == n {
			n.parent.left = n.left
			if n.left != nil {
				n.left.parent = n.parent
			}
			t.root.insert(n.right)
		} else {
			n.parent.right = n.right
			if n.right != nil {
				n.right.parent = n.parent
			}
			t.root.insert(n.left)
		}
		n.delete()
	}
}

func (t *tree) removeKey(root *node, key string) {
	if root != nil {
		if root.key == key {
			t.removeNode(root)
		} else if root.key < key {
			t.removeKey(root.right, key)
		} else {
			t.removeKey(root.left, key)
		}
	}
}
func (t *tree) remove(key string) {
	mutex.Lock()
	t.removeKey(t.root, key)
	mutex.Unlock()
}
