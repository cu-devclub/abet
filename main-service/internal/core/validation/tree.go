package validation

import (
	"archive/zip"
	"sort"
	"strings"
)

// node is one entry of the (virtual) extracted submission. The tree mirrors what the worker sees after
// `zipfile.extractall`, so every rule below can be a direct port of runtime_pipeline.py.
type node struct {
	name     string
	dir      bool
	file     *zip.File
	children map[string]*node
}

func newDir(name string) *node { return &node{name: name, dir: true, children: map[string]*node{}} }

// isJunkName ports runtime_pipeline.is_archive_junk_name.
func isJunkName(name string) bool {
	return name == ".DS_Store" || strings.HasPrefix(name, "._") || strings.EqualFold(name, "__macosx")
}

// buildTree turns zip entries into a tree: junk is stripped, path components are sanitised like
// Python's ZipFile.extract (no empty, "." or ".." components).
func buildTree(files []*zip.File) *node {
	root := newDir("")
	for _, f := range files {
		name := f.Name
		isDir := strings.HasSuffix(name, "/") || f.FileInfo().IsDir()
		var parts []string
		for _, p := range strings.Split(name, "/") {
			if p == "" || p == "." || p == ".." {
				continue
			}
			parts = append(parts, p)
		}
		if len(parts) == 0 {
			continue
		}
		cur := root
		for i, p := range parts {
			last := i == len(parts)-1
			child, ok := cur.children[p]
			switch {
			case last && !isDir:
				if ok && child.dir {
					continue // a directory of the same name already exists
				}
				cur.children[p] = &node{name: p, file: f}
			case !ok || !child.dir:
				child = newDir(p)
				cur.children[p] = child
				cur = child
			default:
				cur = child
			}
		}
	}
	stripJunk(root)
	return root
}

func stripJunk(n *node) {
	for name, c := range n.children {
		if isJunkName(name) {
			delete(n.children, name)
			continue
		}
		if c.dir {
			stripJunk(c)
		}
	}
}

func (n *node) sortedChildren() []*node {
	out := make([]*node, 0, len(n.children))
	for _, c := range n.children {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (n *node) subdirs() []*node {
	var out []*node
	for _, c := range n.sortedChildren() {
		if c.dir {
			out = append(out, c)
		}
	}
	return out
}

func (n *node) hasDir(name string) bool {
	c, ok := n.children[name]
	return ok && c.dir
}

// fileAt returns the file at a slash-separated relative path (nil when missing or not a file).
func (n *node) fileAt(relpath string) *node {
	cur := n
	for _, p := range strings.Split(relpath, "/") {
		c, ok := cur.children[p]
		if !ok {
			return nil
		}
		cur = c
	}
	if cur.dir {
		return nil
	}
	return cur
}

// hasRequiredLayout ports has_required_course_layout.
func (n *node) hasRequiredLayout() bool {
	for _, s := range requiredSubdirs {
		if !n.hasDir(s) {
			return false
		}
	}
	return true
}

// looksLikeLayout ports looks_like_course_layout.
func (n *node) looksLikeLayout() bool {
	for _, s := range requiredSubdirs {
		if n.hasDir(s) {
			return true
		}
	}
	for _, rel := range requiredFiles {
		if n.fileAt(rel) != nil {
			return true
		}
	}
	return false
}

// findWrapped ports find_wrapped_course_layout: collapse up to 5 single-folder wrapper levels.
func findWrapped(root *node) *node {
	cur := root
	for depth := 0; depth < 5; depth++ {
		if cur.hasRequiredLayout() || cur.looksLikeLayout() {
			return cur
		}
		items := cur.sortedChildren()
		if len(items) != 1 || !items[0].dir {
			return root
		}
		cur = items[0]
	}
	return root
}
