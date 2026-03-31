package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// MerkleNode represents a node in the Merkle tree.
type MerkleNode struct {
	Hash     string       `json:"hash"`
	Path     string       `json:"path"`
	IsFile   bool         `json:"is_file"`
	Size     int64        `json:"size"`
	ModTime  int64        `json:"mod_time"`
	Children []*MerkleNode `json:"children,omitempty"`
}

// MerkleTree represents the complete Merkle tree for a user's filesystem.
type MerkleTree struct {
	Root   *MerkleNode `json:"root"`
	UserID string      `json:"user_id"`
}

// DiffResult represents the difference between two Merkle trees.
type DiffResult struct {
	Added    []string `json:"added"`    // Paths that exist in remote but not local
	Modified []string `json:"modified"` // Paths with different hashes
	Deleted  []string `json:"deleted"`  // Paths that exist in local but not remote
}

// BuildMerkleTree builds a Merkle tree from a list of file metadata.
// Each file should have a path, size, modification time, and content hash.
type FileInfo struct {
	Path    string
	IsFile  bool
	Size    int64
	ModTime int64
	Hash    string // Content hash for files
}

// Build constructs a Merkle tree from a flat list of file information.
func Build(files []FileInfo) *MerkleNode {
	if len(files) == 0 {
		return nil
	}

	// Sort files by path for consistent tree structure
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Build tree recursively
	return buildNode("", files)
}

// buildNode recursively builds a Merkle node from files sharing a common prefix.
func buildNode(prefix string, files []FileInfo) *MerkleNode {
	if len(files) == 0 {
		return nil
	}

	// At root level (prefix == ""), always create a directory even for single files
	// This ensures consistent tree structure for diff operations
	isRootLevel := prefix == ""

	// Single file at non-root level = leaf node
	if !isRootLevel && len(files) == 1 && files[0].Path == prefix {
		return &MerkleNode{
			Hash:    files[0].Hash,
			Path:    files[0].Path,
			IsFile:  true,
			Size:    files[0].Size,
			ModTime: files[0].ModTime,
		}
	}

	// Group files by their immediate child directory
	groups := make(map[string][]FileInfo)
	var directChildren []FileInfo

	for _, f := range files {
		// Get the relative path from prefix
		relPath := f.Path
		if prefix != "" {
			relPath = f.Path[len(prefix)+1:] // +1 for the separator
		}

		// Check if this is a direct child
		if idx := indexOf(relPath, '/'); idx == -1 {
			// Direct child file or directory
			directChildren = append(directChildren, f)
		} else {
			// Nested file - group by first directory
			dirName := relPath[:idx]
			if prefix != "" {
				dirName = prefix + "/" + dirName
			}
			groups[dirName] = append(groups[dirName], f)
		}
	}

	// Build children nodes
	var children []*MerkleNode

	// Process direct children first
	for _, f := range directChildren {
		if f.IsFile {
			// Leaf file node
			children = append(children, &MerkleNode{
				Hash:    f.Hash,
				Path:    f.Path,
				IsFile:  true,
				Size:    f.Size,
				ModTime: f.ModTime,
			})
		} else {
			// Empty directory - create node with hash of name
			children = append(children, &MerkleNode{
				Hash:    hashString(f.Path),
				Path:    f.Path,
				IsFile:  false,
				Size:    0,
				ModTime: f.ModTime,
			})
		}
	}

	// Process grouped directories
	var dirNames []string
	for name := range groups {
		dirNames = append(dirNames, name)
	}
	sort.Strings(dirNames)

	for _, dirName := range dirNames {
		child := buildNode(dirName, groups[dirName])
		if child != nil {
			children = append(children, child)
		}
	}

	// Sort children by path for consistent hashing
	sort.Slice(children, func(i, j int) bool {
		return children[i].Path < children[j].Path
	})

	// Compute directory hash from sorted child hashes
	node := &MerkleNode{
		Path:     prefix,
		IsFile:   false,
		Children: children,
	}

	// Calculate total size and latest mod time
	var totalSize int64
	var latestModTime int64
	for _, child := range children {
		totalSize += child.Size
		if child.ModTime > latestModTime {
			latestModTime = child.ModTime
		}
	}
	node.Size = totalSize
	node.ModTime = latestModTime

	// Compute hash from children
	node.Hash = computeNodeHash(children)

	return node
}

// computeNodeHash computes the hash of a directory from its children.
func computeNodeHash(children []*MerkleNode) string {
	h := sha256.New()
	for _, child := range children {
		h.Write([]byte(child.Hash))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// hashString computes a hash of a string.
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// indexOf returns the index of the first occurrence of c in s, or -1 if not found.
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// contains returns true if s contains c.
func contains(s string, c byte) bool {
	return indexOf(s, c) != -1
}

// Diff compares two Merkle trees and returns the differences.
// It uses subtree comparison to skip unchanged directories.
func Diff(local, remote *MerkleNode) *DiffResult {
	result := &DiffResult{
		Added:    []string{},
		Modified: []string{},
		Deleted:  []string{},
	}

	if local == nil && remote == nil {
		return result
	}
	if local == nil {
		// Everything is added
		collectPaths(remote, &result.Added)
		return result
	}
	if remote == nil {
		// Everything is deleted
		collectPaths(local, &result.Deleted)
		return result
	}

	// Compare hashes at root
	if local.Hash == remote.Hash {
		// Trees are identical
		return result
	}

	// Recursively compare children
	diffNodes(local, remote, result)

	return result
}

// diffNodes recursively compares two nodes.
func diffNodes(local, remote *MerkleNode, result *DiffResult) {
	if local.Hash == remote.Hash {
		// Subtrees are identical
		return
	}

	if local.IsFile && remote.IsFile {
		// Both are files but hashes differ
		result.Modified = append(result.Modified, local.Path)
		return
	}

	if local.IsFile != remote.IsFile {
		// Type changed (file <-> directory)
		result.Deleted = append(result.Deleted, local.Path)
		result.Added = append(result.Added, remote.Path)
		return
	}

	// Both are directories - compare children
	localChildren := make(map[string]*MerkleNode)
	for _, child := range local.Children {
		localChildren[child.Path] = child
	}

	remoteChildren := make(map[string]*MerkleNode)
	for _, child := range remote.Children {
		remoteChildren[child.Path] = child
	}

	// Find added and modified
	for path, remoteChild := range remoteChildren {
		if localChild, exists := localChildren[path]; exists {
			// Path exists in both - recurse
			diffNodes(localChild, remoteChild, result)
		} else {
			// Added
			collectPaths(remoteChild, &result.Added)
		}
	}

	// Find deleted
	for path, localChild := range localChildren {
		if _, exists := remoteChildren[path]; !exists {
			// Deleted
			collectPaths(localChild, &result.Deleted)
		}
	}
}

// collectPaths collects all paths under a node.
func collectPaths(node *MerkleNode, paths *[]string) {
	if node == nil {
		return
	}
	*paths = append(*paths, node.Path)
	for _, child := range node.Children {
		collectPaths(child, paths)
	}
}

// FindNode finds a node by path in the tree.
func (t *MerkleTree) FindNode(path string) *MerkleNode {
	if t.Root == nil {
		return nil
	}
	return findNodeRecursive(t.Root, path)
}

// findNodeRecursive recursively searches for a node by path.
func findNodeRecursive(node *MerkleNode, path string) *MerkleNode {
	if node.Path == path {
		return node
	}
	for _, child := range node.Children {
		if found := findNodeRecursive(child, path); found != nil {
			return found
		}
	}
	return nil
}

// GetRootHash returns the root hash of the tree.
func (t *MerkleTree) GetRootHash() string {
	if t.Root == nil {
		return ""
	}
	return t.Root.Hash
}
