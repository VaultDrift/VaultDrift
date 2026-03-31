//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package fuse

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// VaultDriftFS implements a FUSE filesystem for VaultDrift
type VaultDriftFS struct {
	vfs      *vfs.VFS
	db       *db.Manager
	userID   string
	uid      uint32
	gid      uint32
	rootNode *DirNode
}

// DirNode represents a directory in the filesystem
type DirNode struct {
	fs.Inode
	vfs      *vfs.VFS
	db       *db.Manager
	userID   string
	folderID string
}

// FileNode represents a file in the filesystem
type FileNode struct {
	fs.Inode
	vfs    *vfs.VFS
	db     *db.Manager
	userID string
	fileID string
	size   uint64
	mode   uint32
}

// FileHandle represents an open file
type FileHandle struct {
	node  *FileNode
	data  []byte
	dirty bool
}

// Ensure interfaces are implemented
var (
	_ fs.NodeOpener    = (*FileNode)(nil)
	_ fs.NodeReader    = (*FileNode)(nil)
	_ fs.NodeWriter    = (*FileNode)(nil)
	_ fs.NodeSetattrer = (*FileNode)(nil)
	_ fs.NodeGetattrer = (*FileNode)(nil)
	_ fs.NodeCreater   = (*DirNode)(nil)
	_ fs.NodeMkdirer   = (*DirNode)(nil)
	_ fs.NodeRmdirer   = (*DirNode)(nil)
	_ fs.NodeUnlinker  = (*DirNode)(nil)
	_ fs.NodeReaddirer = (*DirNode)(nil)
	_ fs.NodeLookuper  = (*DirNode)(nil)
	_ fs.NodeGetattrer = (*DirNode)(nil)
)

// NewVaultDriftFS creates a new FUSE filesystem
func NewVaultDriftFS(vfsService *vfs.VFS, database *db.Manager, userID string) *VaultDriftFS {
	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())

	root := &DirNode{
		vfs:      vfsService,
		db:       database,
		userID:   userID,
		folderID: "", // Root folder
	}

	return &VaultDriftFS{
		vfs:      vfsService,
		db:       database,
		userID:   userID,
		uid:      uid,
		gid:      gid,
		rootNode: root,
	}
}

// Mount mounts the filesystem at the given path
func (fsys *VaultDriftFS) Mount(mountPoint string) (*fuse.Server, error) {
	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			Name:          "vaultdrift",
			FsName:        "vaultdrift",
			DisableXAttrs: true,
			Debug:         false,
		},
		UID: fsys.uid,
		GID: fsys.gid,
	}

	server, err := fs.Mount(mountPoint, fsys.rootNode, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to mount: %w", err)
	}

	return server, nil
}

// Getattr implements fs.NodeGetattrer
func (n *DirNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = uint32(fuse.S_IFDIR | 0755)
	out.Uid = uint32(os.Getuid())
	out.Gid = uint32(os.Getgid())
	out.Size = 4096
	return fs.OK
}

// Lookup implements fs.NodeLookuper
func (n *DirNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	// List directory contents
	opts := db.ListOpts{Limit: 1000}
	entries, err := n.vfs.ListDirectory(ctx, n.userID, n.folderID, opts)
	if err != nil {
		return nil, syscall.EIO
	}

	// Find matching entry
	for _, entry := range entries {
		if entry.Name != name {
			continue
		}

		if entry.Type == "folder" {
			child := &DirNode{
				vfs:      n.vfs,
				db:       n.db,
				userID:   n.userID,
				folderID: entry.ID,
			}
			out.Mode = uint32(fuse.S_IFDIR | 0755)
			return n.NewInode(ctx, child, fs.StableAttr{Mode: fuse.S_IFDIR}), fs.OK
		}

		child := &FileNode{
			vfs:    n.vfs,
			db:     n.db,
			userID: n.userID,
			fileID: entry.ID,
			size:   uint64(entry.SizeBytes),
			mode:   uint32(fuse.S_IFREG | 0644),
		}
		out.Mode = child.mode
		out.Size = child.size
		return n.NewInode(ctx, child, fs.StableAttr{Mode: fuse.S_IFREG}), fs.OK
	}

	return nil, syscall.ENOENT
}

// Readdir implements fs.NodeReaddirer
func (n *DirNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	opts := db.ListOpts{Limit: 1000}
	entries, err := n.vfs.ListDirectory(ctx, n.userID, n.folderID, opts)
	if err != nil {
		return nil, syscall.EIO
	}

	var dirEntries []fuse.DirEntry
	for _, entry := range entries {
		mode := fuse.S_IFREG
		if entry.Type == "folder" {
			mode = fuse.S_IFDIR
		}
		dirEntries = append(dirEntries, fuse.DirEntry{
			Name: entry.Name,
			Mode: uint32(mode),
		})
	}

	return fs.NewListDirStream(dirEntries), fs.OK
}

// Mkdir implements fs.NodeMkdirer
func (n *DirNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	_, err := n.vfs.CreateFolder(ctx, n.userID, n.folderID, name)
	if err != nil {
		return nil, syscall.EIO
	}

	// Get the newly created folder
	opts := db.ListOpts{Limit: 100}
	entries, err := n.vfs.ListDirectory(ctx, n.userID, n.folderID, opts)
	if err != nil {
		return nil, syscall.EIO
	}

	// Find the folder we just created
	var folderID string
	for _, entry := range entries {
		if entry.Name == name && entry.Type == "folder" {
			folderID = entry.ID
			break
		}
	}

	child := &DirNode{
		vfs:      n.vfs,
		db:       n.db,
		userID:   n.userID,
		folderID: folderID,
	}

	out.Mode = uint32(fuse.S_IFDIR | 0755)
	return n.NewInode(ctx, child, fs.StableAttr{Mode: fuse.S_IFDIR}), fs.OK
}

// Create implements fs.NodeCreater
func (n *DirNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (inode *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// Create empty file in VFS
	file, err := n.vfs.CreateFile(ctx, n.userID, n.folderID, name, "application/octet-stream", 0)
	if err != nil {
		return nil, nil, 0, syscall.EIO
	}

	child := &FileNode{
		vfs:    n.vfs,
		db:     n.db,
		userID: n.userID,
		fileID: file.ID,
		size:   0,
		mode:   uint32(fuse.S_IFREG | 0644),
	}

	out.Mode = child.mode
	inode = n.NewInode(ctx, child, fs.StableAttr{Mode: fuse.S_IFREG})

	// Return file handle for writing
	fh = &FileHandle{
		node: child,
		data: []byte{},
	}

	return inode, fh, 0, fs.OK
}

// Unlink implements fs.NodeUnlinker (delete file)
func (n *DirNode) Unlink(ctx context.Context, name string) syscall.Errno {
	// Find the file
	opts := db.ListOpts{Limit: 1000}
	entries, err := n.vfs.ListDirectory(ctx, n.userID, n.folderID, opts)
	if err != nil {
		return syscall.EIO
	}

	for _, entry := range entries {
		if entry.Name == name && entry.Type == "file" {
			if err := n.vfs.Delete(ctx, entry.ID); err != nil {
				return syscall.EIO
			}
			return fs.OK
		}
	}

	return syscall.ENOENT
}

// Rmdir implements fs.NodeRmdirer (delete folder)
func (n *DirNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	// Find the folder
	opts := db.ListOpts{Limit: 1000}
	entries, err := n.vfs.ListDirectory(ctx, n.userID, n.folderID, opts)
	if err != nil {
		return syscall.EIO
	}

	for _, entry := range entries {
		if entry.Name == name && entry.Type == "folder" {
			if err := n.vfs.Delete(ctx, entry.ID); err != nil {
				return syscall.EIO
			}
			return fs.OK
		}
	}

	return syscall.ENOENT
}

// Getattr implements fs.NodeGetattrer for files
func (n *FileNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = n.mode
	out.Size = n.size
	out.Uid = uint32(os.Getuid())
	out.Gid = uint32(os.Getgid())
	return fs.OK
}

// Open implements fs.NodeOpener
func (n *FileNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	fh := &FileHandle{
		node: n,
		data: nil, // Will be loaded on read
	}
	return fh, 0, fs.OK
}

// Read implements fs.NodeReader
func (n *FileNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// Note: Full implementation would use vfs.ReadFile
	// For now, return empty data
	return fuse.ReadResultData([]byte{}), fs.OK
}

// Write implements fs.NodeWriter
func (n *FileNode) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	handle := fh.(*FileHandle)

	// Ensure buffer is large enough
	end := off + int64(len(data))
	if end > int64(len(handle.data)) {
		newData := make([]byte, end)
		copy(newData, handle.data)
		handle.data = newData
	}

	copy(handle.data[off:], data)
	handle.dirty = true
	n.size = uint64(len(handle.data))

	return uint32(len(data)), fs.OK
}

// Setattr implements fs.NodeSetattrer
func (n *FileNode) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if size, ok := in.GetSize(); ok {
		n.size = size
		if fh != nil {
			handle := fh.(*FileHandle)
			if int64(len(handle.data)) != int64(size) {
				newData := make([]byte, size)
				copy(newData, handle.data)
				handle.data = newData
				handle.dirty = true
			}
		}
	}

	out.Mode = n.mode
	out.Size = n.size
	return fs.OK
}

// Release is called when the file is closed
func (fh *FileHandle) Release(ctx context.Context) syscall.Errno {
	// Note: Full implementation would write data back to VFS
	fh.dirty = false
	return fs.OK
}

// Mount mounts the VaultDrift filesystem at the given mount point
func Mount(vfsService *vfs.VFS, database *db.Manager, userID, mountPoint string) (*fuse.Server, error) {
	// Ensure mount point exists
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mount point: %w", err)
	}

	fsys := NewVaultDriftFS(vfsService, database, userID)
	return fsys.Mount(mountPoint)
}

// Unmount unmounts the filesystem using the server
func Unmount(server *fuse.Server) error {
	return server.Unmount()
}
