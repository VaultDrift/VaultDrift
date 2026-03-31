package sync

import (
	"testing"
)

func TestBuild(t *testing.T) {
	files := []FileInfo{
		{Path: "docs/readme.md", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("readme")},
		{Path: "docs/guide.md", IsFile: true, Size: 200, ModTime: 2000, Hash: hashString("guide")},
		{Path: "src/main.go", IsFile: true, Size: 300, ModTime: 3000, Hash: hashString("main")},
		{Path: "src/lib/helper.go", IsFile: true, Size: 150, ModTime: 1500, Hash: hashString("helper")},
	}

	tree := Build(files)
	if tree == nil {
		t.Fatal("Build() returned nil")
	}

	// Root should be a directory
	if tree.IsFile {
		t.Error("root should be a directory")
	}

	// Root should have 2 children: docs, src
	if len(tree.Children) != 2 {
		t.Errorf("expected 2 children at root, got %d", len(tree.Children))
	}

	// Check total size
	expectedSize := int64(100 + 200 + 300 + 150)
	if tree.Size != expectedSize {
		t.Errorf("expected total size %d, got %d", expectedSize, tree.Size)
	}

	// Check that hash is computed
	if tree.Hash == "" {
		t.Error("root hash should not be empty")
	}
}

func TestBuildEmpty(t *testing.T) {
	tree := Build([]FileInfo{})
	if tree != nil {
		t.Error("Build() with empty files should return nil")
	}
}

func TestBuildSingleFile(t *testing.T) {
	// At root level, single files are now wrapped in a directory node
	// for consistency with multi-file trees
	files := []FileInfo{
		{Path: "file.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("content")},
	}

	tree := Build(files)
	if tree == nil {
		t.Fatal("Build() returned nil")
	}

	// At root level, this should be a directory with one child
	if tree.IsFile {
		t.Error("root should be a directory node (not file)")
	}

	if len(tree.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(tree.Children))
	}

	if tree.Children[0].Path != "file.txt" {
		t.Errorf("expected child path file.txt, got %s", tree.Children[0].Path)
	}

	if !tree.Children[0].IsFile {
		t.Error("child should be a file node")
	}
}

func TestDiffIdentical(t *testing.T) {
	files := []FileInfo{
		{Path: "file1.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("file1")},
		{Path: "file2.txt", IsFile: true, Size: 200, ModTime: 2000, Hash: hashString("file2")},
	}

	local := Build(files)
	remote := Build(files)

	diff := Diff(local, remote)

	if len(diff.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(diff.Added))
	}
	if len(diff.Modified) != 0 {
		t.Errorf("expected 0 modified, got %d", len(diff.Modified))
	}
	if len(diff.Deleted) != 0 {
		t.Errorf("expected 0 deleted, got %d", len(diff.Deleted))
	}
}

func TestDiffAdded(t *testing.T) {
	localFiles := []FileInfo{
		{Path: "file1.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("file1")},
	}

	remoteFiles := []FileInfo{
		{Path: "file1.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("file1")},
		{Path: "file2.txt", IsFile: true, Size: 200, ModTime: 2000, Hash: hashString("file2")},
	}

	local := Build(localFiles)
	remote := Build(remoteFiles)

	diff := Diff(local, remote)

	if len(diff.Added) != 1 || diff.Added[0] != "file2.txt" {
		t.Errorf("expected 1 added (file2.txt), got %v", diff.Added)
	}
	if len(diff.Modified) != 0 {
		t.Errorf("expected 0 modified, got %d", len(diff.Modified))
	}
	if len(diff.Deleted) != 0 {
		t.Errorf("expected 0 deleted, got %d", len(diff.Deleted))
	}
}

func TestDiffDeleted(t *testing.T) {
	localFiles := []FileInfo{
		{Path: "file1.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("file1")},
		{Path: "file2.txt", IsFile: true, Size: 200, ModTime: 2000, Hash: hashString("file2")},
	}

	remoteFiles := []FileInfo{
		{Path: "file1.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("file1")},
	}

	local := Build(localFiles)
	remote := Build(remoteFiles)

	diff := Diff(local, remote)

	if len(diff.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(diff.Added))
	}
	if len(diff.Modified) != 0 {
		t.Errorf("expected 0 modified, got %d", len(diff.Modified))
	}
	if len(diff.Deleted) != 1 || diff.Deleted[0] != "file2.txt" {
		t.Errorf("expected 1 deleted (file2.txt), got %v", diff.Deleted)
	}
}

func TestDiffModified(t *testing.T) {
	localFiles := []FileInfo{
		{Path: "file1.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("old")},
	}

	remoteFiles := []FileInfo{
		{Path: "file1.txt", IsFile: true, Size: 150, ModTime: 2000, Hash: hashString("new")},
	}

	local := Build(localFiles)
	remote := Build(remoteFiles)

	diff := Diff(local, remote)

	if len(diff.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(diff.Added))
	}
	if len(diff.Modified) != 1 || diff.Modified[0] != "file1.txt" {
		t.Errorf("expected 1 modified (file1.txt), got %v", diff.Modified)
	}
	if len(diff.Deleted) != 0 {
		t.Errorf("expected 0 deleted, got %d", len(diff.Deleted))
	}
}

func TestDiffSubtreeSkip(t *testing.T) {
	// If a directory hash matches, the entire subtree should be skipped
	localFiles := []FileInfo{
		{Path: "docs/readme.md", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("readme")},
		{Path: "src/main.go", IsFile: true, Size: 300, ModTime: 3000, Hash: hashString("main")},
	}

	remoteFiles := []FileInfo{
		{Path: "docs/readme.md", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("readme")},
		{Path: "src/main.go", IsFile: true, Size: 400, ModTime: 4000, Hash: hashString("main-modified")},
	}

	local := Build(localFiles)
	remote := Build(remoteFiles)

	diff := Diff(local, remote)

	// docs/ should be skipped entirely (no changes reported within it)
	// only src/main.go should be reported as modified
	found := false
	for _, path := range diff.Modified {
		if path == "src/main.go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected src/main.go in modified, got %v", diff.Modified)
	}

	// docs/readme.md should NOT be in modified (subtree skipped)
	for _, path := range diff.Modified {
		if path == "docs/readme.md" {
			t.Error("docs/readme.md should not be in modified (subtree should be skipped)")
		}
	}
}

func TestFindNode(t *testing.T) {
	files := []FileInfo{
		{Path: "docs/readme.md", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("readme")},
		{Path: "src/main.go", IsFile: true, Size: 300, ModTime: 3000, Hash: hashString("main")},
	}

	tree := &MerkleTree{Root: Build(files), UserID: "user1"}

	node := tree.FindNode("docs/readme.md")
	if node == nil {
		t.Fatal("FindNode() returned nil for existing path")
	}
	if node.Path != "docs/readme.md" {
		t.Errorf("expected path docs/readme.md, got %s", node.Path)
	}

	node = tree.FindNode("nonexistent")
	if node != nil {
		t.Error("FindNode() should return nil for nonexistent path")
	}
}

func TestGetRootHash(t *testing.T) {
	tree := &MerkleTree{Root: nil, UserID: "user1"}
	if tree.GetRootHash() != "" {
		t.Error("GetRootHash() should return empty string for nil root")
	}

	files := []FileInfo{
		{Path: "file.txt", IsFile: true, Size: 100, ModTime: 1000, Hash: hashString("content")},
	}
	tree.Root = Build(files)

	if tree.GetRootHash() == "" {
		t.Error("GetRootHash() should not be empty for valid tree")
	}
}

func BenchmarkBuild(b *testing.B) {
	files := make([]FileInfo, 1000)
	for i := range files {
		files[i] = FileInfo{
			Path:    "dir" + string(rune('0'+i%10)) + "/file" + string(rune('0'+i/10)) + ".txt",
			IsFile:  true,
			Size:    int64(i * 100),
			ModTime: int64(i * 1000),
			Hash:    hashString(string(rune(i))),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Build(files)
	}
}

func BenchmarkDiff(b *testing.B) {
	localFiles := make([]FileInfo, 1000)
	remoteFiles := make([]FileInfo, 1000)

	for i := range localFiles {
		localFiles[i] = FileInfo{
			Path:    "dir" + string(rune('0'+i%10)) + "/file" + string(rune('0'+i/10)) + ".txt",
			IsFile:  true,
			Size:    int64(i * 100),
			ModTime: int64(i * 1000),
			Hash:    hashString("local" + string(rune(i))),
		}
		remoteFiles[i] = FileInfo{
			Path:    "dir" + string(rune('0'+i%10)) + "/file" + string(rune('0'+i/10)) + ".txt",
			IsFile:  true,
			Size:    int64(i * 100),
			ModTime: int64(i * 1000),
			Hash:    hashString("remote" + string(rune(i))),
		}
	}

	local := Build(localFiles)
	remote := Build(remoteFiles)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Diff(local, remote)
	}
}
