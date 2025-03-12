// Package filen provides an interface to Filen cloud storage.
package filen

import (
	"context"
	"errors"
	"fmt"
	sdk "github.com/FilenCloudDienste/filen-sdk-go/filen"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/types"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rclone/rclone/fs/hash"
	"github.com/rclone/rclone/lib/encoder"
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
			{
				Name:     config.ConfigEncoding,
				Help:     config.ConfigEncodingHelp,
				Advanced: true,
				Default:  encoder.Standard | encoder.EncodeInvalidUtf8,
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
	maybeRootDir, err := filen.FindDirectory(ctx, root)
	if errors.Is(err, fs.ErrorIsFile) { // FsIsFile special case
		var err2 error
		root = pathModule.Dir(root)
		maybeRootDir, err2 = filen.FindDirectory(ctx, root)
		if err2 != nil {
			return nil, err2
		}
	} else if err != nil {
		return nil, err
	}

	fileSystem := &Fs{
		name:  name,
		root:  Directory{},
		filen: filen,
		Enc:   opt.Encoder,
	}

	fileSystem.root = Directory{
		fs:        fileSystem,
		directory: maybeRootDir, // could be null at this point
		path:      root,
	}

	// must return the error from FindDirectory (see FsIsFile)
	return fileSystem, err
}

type Fs struct {
	name  string
	root  Directory
	filen *sdk.Filen
	Enc   encoder.MultiEncoder
}

// resolvePath returns the absolute path specified by the input path, which is seen relative to the remote's root.
func (f *Fs) resolvePath(path string) string {
	return pathModule.Join(f.root.path, path)
}

type Options struct {
	Email    string               `config:"email"`
	Password string               `config:"password"`
	APIKey   string               `config:"api_key"`
	Encoder  encoder.MultiEncoder `config:"encoding"`
}

func (f *Fs) Name() string {
	return f.name
}

func (f *Fs) Root() string {
	return f.root.path
}

func (f *Fs) String() string {
	return fmt.Sprintf("Filen %s at /%s", f.filen.Email, f.root.String())
}

func (f *Fs) Precision() time.Duration {
	return time.Millisecond
}

func (f *Fs) Hashes() hash.Set {
	return hash.Set(hash.SHA512)
}

func (f *Fs) Features() *fs.Features {
	return &fs.Features{
		ReadMimeType: true,
		//WriteMimeType:           true, // requires parsing metadata options, todo later
		CanHaveEmptyDirectories: true,
		// ReadMetadata: true,
		// WriteMetadata: true,
		// ReadDirMetadata ?
		// WriteDirMetadata ?
		//TODO more optional features?
	}
}

func (f *Fs) List(ctx context.Context, dir string) (entries fs.DirEntries, err error) {
	dir = f.Enc.FromStandardPath(dir)
	// find directory uuid
	directory, err := f.filen.FindDirectory(ctx, f.resolvePath(dir))
	if err != nil {
		return nil, err
	}

	if directory == nil {
		return nil, fs.ErrorDirNotFound
	}

	// read directory content
	files, directories, err := f.filen.ReadDirectory(ctx, directory)
	if err != nil {
		return nil, err
	}
	entries = make(fs.DirEntries, 0, len(files)+len(directories))

	for _, directory := range directories {
		entries = append(entries, &Directory{
			fs:        f,
			path:      pathModule.Join(dir, directory.Name),
			directory: directory,
		})
	}
	for _, file := range files {
		file := &File{
			fs:   f,
			path: pathModule.Join(dir, file.Name),
			file: file,
		}
		entries = append(entries, file)
	}
	return entries, nil
}

func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	remote = f.Enc.FromStandardPath(remote)
	obj, err := f.filen.FindItem(ctx, f.resolvePath(remote))
	if err != nil {
		return nil, err
	}
	file, ok := obj.(*types.File)
	if !ok || obj == nil {
		return nil, fs.ErrorObjectNotFound
	}
	return &File{
		fs:   f,
		path: remote,
		file: file,
	}, nil
}

func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	path := f.Enc.FromStandardPath(src.Remote())
	resolvedPath := f.resolvePath(path)
	modTime := src.ModTime(ctx)
	parent, err := f.filen.FindDirectoryOrCreate(ctx, pathModule.Dir(resolvedPath))
	if err != nil {
		return nil, err
	}
	incompleteFile, err := types.NewIncompleteFile(f.filen.AuthVersion, pathModule.Base(resolvedPath), "", modTime, modTime, parent)
	if err != nil {
		return nil, err
	}
	uploadedFile, err := f.filen.UploadFile(ctx, incompleteFile, in)
	if err != nil {
		return nil, err
	}
	return &File{
		fs:   f,
		path: path,
		file: uploadedFile,
	}, nil
}

func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	dirObj, err := f.filen.FindDirectoryOrCreate(ctx, f.resolvePath(f.Enc.FromStandardPath(dir)))
	if err != nil {
		return err
	}
	if dir == f.root.path {
		f.root.directory = dirObj
	}
	return nil
}

func (f *Fs) Rmdir(ctx context.Context, dir string) error {
	// find directory
	resolvedPath := f.resolvePath(f.Enc.FromStandardPath(dir))
	//if resolvedPath == f.root.path {
	//	return fs.ErrorDirNotFound
	//}
	directory, err := f.filen.FindDirectory(ctx, resolvedPath)
	if err != nil {
		return err
	}
	if directory == nil {
		return errors.New("directory not found")
	}

	files, dirs, err := f.filen.ReadDirectory(ctx, directory)
	if err != nil {
		return err
	}
	if len(files) > 0 || len(dirs) > 0 {
		return errors.New("directory is not empty")
	}

	// trash directory
	err = f.filen.TrashDirectory(ctx, directory)
	if err != nil {
		return err
	}
	return nil
}

// Directory

type Directory struct {
	fs        *Fs
	path      string
	directory types.DirectoryInterface
}

func (dir *Directory) Fs() fs.Info {
	return dir.fs
}

func (dir *Directory) String() string {
	if dir == nil {
		return "<nil>"
	}
	return dir.Remote()
}

func (dir *Directory) Remote() string {
	return dir.fs.Enc.ToStandardPath(dir.path)
}

func (dir *Directory) ModTime(ctx context.Context) time.Time {
	directory, ok := dir.directory.(*types.Directory)
	if !ok {
		return time.Time{} // todo add account creation time?
	}

	if directory.Created.IsZero() {
		obj, err := dir.fs.filen.FindDirectory(ctx, dir.path)
		newDir, ok := obj.(*types.Directory)
		if err != nil || !ok {
			return time.Now()
		}
		directory = newDir
		dir.directory = newDir
	}
	return directory.Created
}

func (dir *Directory) Size() int64 {
	return -1
}

func (dir *Directory) Items() int64 {
	return -1
}

func (dir *Directory) ID() string {
	return dir.directory.GetUUID()
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
	return file.Remote()
}

func (file *File) Remote() string {
	return file.fs.Enc.ToStandardPath(file.path)
}

func (file *File) ModTime(ctx context.Context) time.Time {
	// doing this 'properly' is annoying
	// we'd have to call FindItem which can be pretty slow
	// if the backend API gets changed allowing for single call FindItem calls
	// then we should probably swap over to that
	if file.file.LastModified.IsZero() {
		obj, err := file.fs.filen.FindItem(ctx, file.path)
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
		maybeFile, err := file.fs.filen.FindItem(ctx, file.path)
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
	fs.FixRangeOption(options, file.Size())
	// Create variables to hold our options
	var offset int64 = 0
	var limit int64 = -1 // -1 means no limit

	// Parse the options
	for _, option := range options {
		switch opt := option.(type) {
		case *fs.RangeOption:
			offset = opt.Start
			limit = opt.End + 1 // +1 because End is inclusive
		}
	}

	// Get the base reader
	readCloser := file.fs.filen.GetDownloadReaderWithOffset(ctx, file.file, int(offset), int(limit))
	return readCloser, nil
}

func (file *File) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	newModTime := src.ModTime(ctx)
	newIncomplete, err := file.file.NewFromBase(file.fs.filen.AuthVersion)
	if err != nil {
		return err
	}
	newIncomplete.LastModified = newModTime
	newIncomplete.Created = newModTime
	uploadedFile, err := file.fs.filen.UploadFile(ctx, newIncomplete, in)
	if err != nil {
		return err
	}
	file.file = uploadedFile
	return nil
}

func (file *File) Remove(ctx context.Context) error {
	err := file.fs.filen.TrashFile(ctx, *file.file)
	if err != nil {
		return err
	}
	return nil
}

func (file *File) MimeType(_ context.Context) string {
	return file.file.MimeType
}

func (file *File) ID() string {
	return file.file.GetUUID()
}

func (file *File) ParentID() string {
	return file.file.GetParent()
}
