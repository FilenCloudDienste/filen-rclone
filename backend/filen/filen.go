// Package filen provides an interface to Filen cloud storage.
package filen

import (
	"context"
	"errors"
	"fmt"
	sdk "github.com/FilenCloudDienste/filen-sdk-go/filen"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/types"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rclone/rclone/fs/hash"
	"io"
	pathModule "path"
	"time"
)

func init() {
	fs.Register(&fs.RegInfo{
		Name:        "filen",
		Description: "Filen",
		NewFs:       NewFs,
		Options: []fs.Option{
			{
				Name:     "email",
				Help:     "Filen account email",
				Required: true,
			},
			{
				Name:       "password",
				Help:       "Filen account password",
				Required:   true,
				IsPassword: true,
				Sensitive:  true,
			},
			{
				Name:       "api_key",
				Help:       "Filen account API Key",
				Required:   true,
				IsPassword: true,
				Sensitive:  true,
			},
		},
	})
}

func NewFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}

	password, err := obscure.Reveal(opt.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to reveal password: %w", err)
	}
	apiKey, err := obscure.Reveal(opt.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to reveal api key: %w", err)
	}
	filen, err := sdk.NewWithAPIKey(ctx, opt.Email, password, apiKey)
	if err != nil {
		return nil, err
	}
	return &Fs{name, root, filen}, nil
}

type Fs struct {
	name  string
	root  string
	filen *sdk.Filen
}

// resolvePath returns the absolute path specified by the input path, which is seen relative to the remote's root.
func (f *Fs) resolvePath(path string) string {
	return pathModule.Join(f.root, path)
}

type Options struct {
	Email    string `config:"email"`
	Password string `config:"password"`
	APIKey   string `config:"api_key"`
}

func (f *Fs) Name() string {
	return f.name
}

func (f *Fs) Root() string {
	return f.root
}

func (f *Fs) String() string {
	return fmt.Sprintf("Filen %s at /%s", f.filen.Email, f.root)
}

func (f *Fs) Precision() time.Duration {
	return time.Millisecond
}

func (f *Fs) Hashes() hash.Set {
	return hash.Set(hash.SHA512)
}

func (f *Fs) Features() *fs.Features {
	return &fs.Features{
		ReadMimeType:            true,
		WriteMimeType:           true,
		CanHaveEmptyDirectories: true,
		// ReadMetadata: true,
		// WriteMetadata: true,
		// ReadDirMetadata ?
		// WriteDirMetadata ?
		//TODO more optional features?
	}
}

func (f *Fs) List(ctx context.Context, dir string) (entries fs.DirEntries, err error) {
	// find directory uuid
	obj, err := f.filen.FindItem(ctx, f.resolvePath(dir), true)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, errors.New("directory not found")
	}

	directory, ok := obj.(types.DirectoryInterface)
	if !ok {
		return nil, errors.New("not a directory")
	}

	// read directory content
	files, directories, err := f.filen.ReadDirectory(ctx, directory.GetUUID())
	if err != nil {
		return nil, err
	}

	for _, directory := range directories {
		entries = append(entries, &Directory{f, pathModule.Join(dir, directory.Name), directory})
	}
	for _, file := range files {
		entries = append(entries, &File{f, pathModule.Join(dir, file.Name), file})
	}
	return entries, nil
}

func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	obj, err := f.filen.FindItem(ctx, f.resolvePath(remote), false)
	if err != nil {
		return nil, err
	}
	file, ok := obj.(*types.File)
	if !ok {
		return nil, fs.ErrorObjectNotFound
	}
	return &File{f, remote, file}, nil
}

func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	path := src.Remote()
	modTime := src.ModTime(ctx)
	parent, err := f.filen.FindDirectoryOrCreate(ctx, f.root)
	if err != nil {
		return nil, err
	}
	incompleteFile, err := types.NewIncompleteFile(f.filen.AuthVersion, path, "", modTime, modTime, parent.GetUUID())
	if err != nil {
		return nil, err
	}
	uploadedFile, err := f.filen.UploadFile(ctx, incompleteFile, in)
	if err != nil {
		return nil, err
	}
	return &File{f, path, uploadedFile}, nil
}

func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	_, err := f.filen.FindDirectoryOrCreate(ctx, f.resolvePath(dir))
	if err != nil {
		return err
	}
	return nil
}

func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	// find directory
	directory, err := f.filen.FindItem(ctx, f.resolvePath(dir), true)
	if err != nil {
		return err
	}
	if directory == nil {
		return errors.New("directory not found")
	}

	files, dirs, err := f.filen.ReadDirectory(ctx, directory.GetUUID())
	if err != nil {
		return err
	}
	if len(files) > 0 || len(dirs) > 0 {
		return errors.New("directory is not empty")
	}

	// trash directory
	err = f.filen.TrashDirectory(ctx, directory.GetUUID())
	if err != nil {
		return err
	}
	return nil
}

// Directory

type Directory struct {
	fs        *Fs
	path      string
	directory *types.Directory
}

func (dir *Directory) Fs() fs.Info {
	return dir.fs
}

func (dir *Directory) String() string {
	return dir.path
}

func (dir *Directory) Remote() string {
	return dir.path
}

func (dir *Directory) ModTime(ctx context.Context) time.Time {
	if dir.directory.Created.IsZero() {
		obj, err := dir.fs.filen.FindItem(ctx, dir.path, true)
		newDir, ok := obj.(*types.Directory)
		if err != nil || !ok {
			return time.Now()
		}
		dir.directory = newDir
	}
	return dir.directory.Created
}

func (dir *Directory) Size() int64 {
	return -1
}

func (dir *Directory) Items() int64 {
	return -1
}

func (dir *Directory) ID() string {
	return dir.directory.UUID
}

// File

type File struct {
	fs   *Fs
	path string
	file *types.File
}

func (file *File) Fs() fs.Info {
	return file.fs
}

func (file *File) String() string {
	if file == nil {
		return "<nil>"
	}
	return file.path
}

func (file *File) Remote() string {
	return file.path
}

func (file *File) ModTime(ctx context.Context) time.Time {
	// doing this 'properly' is annoying
	// we'd have to call FindItem which can be pretty slow
	// if the backend API gets changed allowing for single call FindItem calls
	// then we should probably swap over to that
	if file.file.LastModified.IsZero() {
		obj, err := file.fs.filen.FindItem(ctx, file.path, false)
		newFile, ok := obj.(*types.File)
		if err == nil && ok {
			file.file = newFile
		}
	}
	return file.file.LastModified
}

func (file *File) Size() int64 {
	return int64(file.file.Size)
}

func (file *File) Hash(ctx context.Context, ty hash.Type) (string, error) {
	if ty != hash.SHA512 {
		return "", hash.ErrUnsupported
	}
	if file.file.Hash == "" {
		maybeFile, err := file.fs.filen.FindItem(ctx, file.path, false)
		if err != nil {
			return "", err
		}
		foundFile, ok := maybeFile.(*types.File)
		if !ok {
			return "", errors.New("not a file")
		}
		file.file = foundFile
		return "", nil
	}
	return file.file.Hash, nil
}

func (file *File) Storable() bool {
	return true
}

func (file *File) SetModTime(ctx context.Context, t time.Time) error {
	file.file.LastModified = t
	return file.fs.filen.UpdateMeta(ctx, file.file)
}

func (file *File) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	readCloser := file.fs.filen.GetDownloadReader(ctx, file.file)
	return readCloser, nil
}

func (file *File) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	uploadedFile, err := file.fs.filen.UploadFile(ctx, &file.file.IncompleteFile, in)
	if err != nil {
		return err
	}
	file.file = uploadedFile
	return nil
}

func (file *File) Remove(ctx context.Context) error {
	err := file.fs.filen.TrashFile(ctx, file.file.UUID)
	if err != nil {
		return err
	}
	return nil
}

func (file *File) MimeType(_ context.Context) string {
	return file.file.MimeType
}
