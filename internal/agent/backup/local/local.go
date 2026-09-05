package local

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GVALFER/WEBYCP/internal/agent/backup"
	"github.com/GVALFER/WEBYCP/internal/agent/hostuser"
	"github.com/GVALFER/WEBYCP/internal/backupfmt"
	"github.com/GVALFER/WEBYCP/internal/execx"
	"github.com/GVALFER/WEBYCP/internal/fsx"
	"github.com/GVALFER/WEBYCP/internal/validate"
)

const (
	defaultRoot = "/var/backups/webycp"
	defaultHome = "/home"
	dumpPath    = "/usr/bin/mysqldump"
	mysqlPath   = "/usr/bin/mysql"
)

var checksumPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

var _ backup.Driver = (*Driver)(nil)

type Driver struct {
	root        string
	home        string
	dump        func(context.Context, io.Writer, string, ...string) error
	input       func(context.Context, io.Reader, string, ...string) error
	lookup      func(string) (*user.User, error)
	lookupGroup func(string) (*user.Group, error)
}

func New() *Driver {
	return &Driver{
		root: defaultRoot, home: defaultHome, dump: execx.Write, input: execx.Input,
		lookup: user.Lookup, lookupGroup: user.LookupGroup,
	}
}

func (d *Driver) Create(ctx context.Context, request backup.CreateRequest) (backup.Artifact, error) {
	if err := d.validateAccount(request.AccountID, request.SystemUser); err != nil {
		return backup.Artifact{}, err
	}
	if validate.ID("runId", request.RunID) != nil || len(request.Metadata) > 4<<20 {
		return backup.Artifact{}, &validate.Error{Field: "runId", Message: "The backup request is invalid"}
	}
	accountDir := filepath.Join(d.root, request.AccountID)
	if err := os.MkdirAll(accountDir, 0o700); err != nil {
		return backup.Artifact{}, fmt.Errorf("create backup account directory: %w", err)
	}
	stage, err := os.MkdirTemp(accountDir, ".stage-*")
	if err != nil {
		return backup.Artifact{}, fmt.Errorf("create backup staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	for _, database := range request.Databases {
		if err := validate.DatabaseSystemName(database); err != nil {
			return backup.Artifact{}, err
		}
		file, err := os.OpenFile(filepath.Join(stage, database+".sql"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return backup.Artifact{}, fmt.Errorf("create database dump: %w", err)
		}
		dumpErr := d.dump(ctx, file, dumpPath, "--protocol=socket", "--single-transaction", "--routines", "--events", "--databases", database)
		closeErr := file.Close()
		if dumpErr != nil || closeErr != nil {
			return backup.Artifact{}, errors.Join(dumpErr, closeErr)
		}
	}
	temporary := filepath.Join(accountDir, "."+request.RunID+".tmp")
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return backup.Artifact{}, fmt.Errorf("create backup artifact: %w", err)
	}
	manifest := backupfmt.Manifest{Version: backupfmt.Version, RunID: request.RunID, AccountID: request.AccountID, CreatedAt: time.Now().UTC(), Files: request.IncludeFiles, Databases: len(request.Databases) > 0, Metadata: request.Metadata != "", Entries: []backupfmt.Entry{}}
	archiveErr := d.writeArchive(file, request, stage, &manifest)
	closeErr := file.Close()
	if archiveErr != nil || closeErr != nil {
		os.Remove(temporary)
		return backup.Artifact{}, errors.Join(archiveErr, closeErr)
	}
	checksum, size, err := fileChecksum(temporary)
	if err != nil {
		os.Remove(temporary)
		return backup.Artifact{}, err
	}
	target := filepath.Join(accountDir, request.RunID+".tar.gz")
	if err := os.Rename(temporary, target); err != nil {
		os.Remove(temporary)
		return backup.Artifact{}, fmt.Errorf("install backup artifact: %w", err)
	}
	return backup.Artifact{Path: target, Checksum: checksum, Size: size, Manifest: manifest}, nil
}

func (d *Driver) Preview(_ context.Context, request backup.ArtifactRequest) (backupfmt.Manifest, error) {
	if err := d.validateArtifact(request); err != nil {
		return backupfmt.Manifest{}, err
	}
	actual, _, err := fileChecksum(request.Path)
	if err != nil {
		return backupfmt.Manifest{}, err
	}
	if actual != request.Checksum {
		return backupfmt.Manifest{}, fmt.Errorf("backup artifact checksum does not match")
	}
	return inspectArchive(request.Path, request.AccountID)
}

func (d *Driver) Restore(ctx context.Context, request backup.RestoreRequest) (string, error) {
	manifest, err := d.Preview(ctx, request.ArtifactRequest)
	if err != nil {
		return "", err
	}
	if err := manifest.ValidateScope(request.Files, request.Databases, request.Metadata); err != nil {
		return "", err
	}
	if err := d.validateAccount(request.AccountID, request.SystemUser); err != nil {
		return "", err
	}
	identity, webGID, err := d.restoreIdentity(request.AccountID, request.SystemUser)
	if err != nil {
		return "", err
	}
	file, gz, reader, err := openArchive(request.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	defer gz.Close()
	metadata := ""
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read backup entry: %w", err)
		}
		switch {
		case request.Files && strings.HasPrefix(header.Name, "files/"):
			if err := d.restoreFile(reader, header, identity, webGID); err != nil {
				return "", err
			}
		case request.Databases && strings.HasPrefix(header.Name, "databases/") && header.Typeflag == tar.TypeReg:
			name := strings.TrimSuffix(path.Base(header.Name), ".sql")
			if err := validate.DatabaseSystemName(name); err != nil {
				return "", err
			}
			if err := d.input(ctx, io.LimitReader(reader, header.Size), mysqlPath, "--protocol=socket"); err != nil {
				return "", fmt.Errorf("restore database %s: %w", name, err)
			}
		case request.Metadata && header.Name == "metadata.json" && header.Typeflag == tar.TypeReg:
			value, err := io.ReadAll(io.LimitReader(reader, 4<<20))
			if err != nil {
				return "", err
			}
			metadata = string(value)
		}
	}
	if request.Metadata && manifest.Metadata && metadata == "" {
		return "", fmt.Errorf("backup metadata is missing")
	}
	return metadata, nil
}

func (d *Driver) Delete(_ context.Context, request backup.ArtifactRequest) error {
	if err := d.validateArtifact(request); err != nil {
		return err
	}
	if err := os.Remove(request.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete backup artifact: %w", err)
	}
	return nil
}

func (d *Driver) writeArchive(output io.Writer, request backup.CreateRequest, stage string, manifest *backupfmt.Manifest) error {
	gz := gzip.NewWriter(output)
	tarWriter := tar.NewWriter(gz)
	closeAll := func() error { return errors.Join(tarWriter.Close(), gz.Close()) }
	if request.IncludeFiles {
		root := filepath.Join(d.home, request.SystemUser)
		if err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(root, filePath)
			if err != nil || relative == "." {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("backup source contains a symlink: %s", relative)
			}
			return addPath(tarWriter, filePath, path.Join("files", filepath.ToSlash(relative)), manifest)
		}); err != nil {
			return errors.Join(err, closeAll())
		}
	}
	entries, err := os.ReadDir(stage)
	if err != nil {
		return errors.Join(err, closeAll())
	}
	for _, entry := range entries {
		if err := addPath(tarWriter, filepath.Join(stage, entry.Name()), path.Join("databases", entry.Name()), manifest); err != nil {
			return errors.Join(err, closeAll())
		}
	}
	if request.Metadata != "" {
		if err := addBytes(tarWriter, "metadata.json", []byte(request.Metadata), manifest); err != nil {
			return errors.Join(err, closeAll())
		}
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return errors.Join(err, closeAll())
	}
	if err := writeTarBytes(tarWriter, "manifest.json", data, 0o600); err != nil {
		return errors.Join(err, closeAll())
	}
	return closeAll()
}

func addPath(writer *tar.Writer, source, name string, manifest *backupfmt.Manifest) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	header.Uid, header.Gid = 0, 0
	header.Uname, header.Gname = "", ""
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(writer, hash), file); err != nil {
		return err
	}
	manifest.Entries = append(manifest.Entries, backupfmt.Entry{Path: name, Size: info.Size(), Checksum: hex.EncodeToString(hash.Sum(nil))})
	return nil
}

func addBytes(writer *tar.Writer, name string, data []byte, manifest *backupfmt.Manifest) error {
	if err := writeTarBytes(writer, name, data, 0o600); err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	manifest.Entries = append(manifest.Entries, backupfmt.Entry{Path: name, Size: int64(len(data)), Checksum: hex.EncodeToString(hash[:])})
	return nil
}

func writeTarBytes(writer *tar.Writer, name string, data []byte, mode int64) error {
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data)), Typeflag: tar.TypeReg, ModTime: time.Now().UTC()}); err != nil {
		return err
	}
	_, err := writer.Write(data)
	return err
}

func inspectArchive(filePath, accountID string) (backupfmt.Manifest, error) {
	file, gz, reader, err := openArchive(filePath)
	if err != nil {
		return backupfmt.Manifest{}, err
	}
	defer file.Close()
	defer gz.Close()
	computed := make(map[string]backupfmt.Entry)
	seen := make(map[string]bool)
	hasFiles, hasDatabases, hasMetadata := false, false, false
	var manifest backupfmt.Manifest
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return backupfmt.Manifest{}, err
		}
		if !validEntryPath(header.Name) || (header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir) {
			return backupfmt.Manifest{}, fmt.Errorf("unsafe backup entry: %s", header.Name)
		}
		if seen[header.Name] {
			return backupfmt.Manifest{}, fmt.Errorf("duplicate backup entry: %s", header.Name)
		}
		seen[header.Name] = true
		hasFiles = hasFiles || strings.HasPrefix(header.Name, "files/")
		hasDatabases = hasDatabases || strings.HasPrefix(header.Name, "databases/") && header.Typeflag == tar.TypeReg
		hasMetadata = hasMetadata || header.Name == "metadata.json" && header.Typeflag == tar.TypeReg
		if header.Name == "manifest.json" {
			if header.Size > 1<<20 || json.NewDecoder(io.LimitReader(reader, header.Size)).Decode(&manifest) != nil {
				return backupfmt.Manifest{}, fmt.Errorf("backup manifest is invalid")
			}
			continue
		}
		if header.Typeflag == tar.TypeReg {
			hash := sha256.New()
			if _, err := io.Copy(hash, io.LimitReader(reader, header.Size)); err != nil {
				return backupfmt.Manifest{}, err
			}
			computed[header.Name] = backupfmt.Entry{Path: header.Name, Size: header.Size, Checksum: hex.EncodeToString(hash.Sum(nil))}
		}
	}
	// tar EOF can occur before gzip's checksum and size trailer have been read.
	if _, err := io.Copy(io.Discard, gz); err != nil {
		return backupfmt.Manifest{}, fmt.Errorf("read backup trailer: %w", err)
	}
	if manifest.Version != backupfmt.Version || manifest.AccountID != accountID || validate.ID("runId", manifest.RunID) != nil {
		return backupfmt.Manifest{}, fmt.Errorf("backup manifest identity is invalid")
	}
	if hasFiles && !manifest.Files || hasDatabases != manifest.Databases || hasMetadata != manifest.Metadata {
		return backupfmt.Manifest{}, fmt.Errorf("backup manifest scope does not match its content")
	}
	if len(manifest.Entries) != len(computed) {
		return backupfmt.Manifest{}, fmt.Errorf("backup manifest entry count does not match")
	}
	for _, entry := range manifest.Entries {
		actual, ok := computed[entry.Path]
		if !ok || actual.Size != entry.Size || actual.Checksum != entry.Checksum {
			return backupfmt.Manifest{}, fmt.Errorf("backup entry checksum does not match: %s", entry.Path)
		}
		delete(computed, entry.Path)
	}
	return manifest, nil
}

func openArchive(filePath string) (*os.File, *gzip.Reader, *tar.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, nil, err
	}
	gz, err := gzip.NewReader(file)
	if err != nil {
		file.Close()
		return nil, nil, nil, err
	}
	return file, gz, tar.NewReader(gz), nil
}

func (d *Driver) restoreFile(reader io.Reader, header *tar.Header, identity hostuser.Identity, webGID int) error {
	relative := strings.TrimPrefix(header.Name, "files/")
	target, err := safeTarget(identity.Home, relative)
	if err != nil {
		return err
	}
	gid := identity.GID
	if relative == "web" || strings.HasPrefix(relative, "web/") {
		gid = webGID
	}
	if header.Typeflag == tar.TypeDir {
		mode := uint32(header.Mode) & 0o770
		if header.Mode&0o2000 != 0 {
			mode |= 0o2000
		}
		if err := os.MkdirAll(target, os.FileMode(mode)); err != nil {
			return err
		}
		directory, err := fsx.OpenDir(target)
		if err != nil {
			return err
		}
		defer directory.Close()
		return directory.Configure(mode, identity.UID, gid)
	}
	if header.Typeflag != tar.TypeReg {
		return fmt.Errorf("unsupported file backup entry: %s", header.Name)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&0o660)
	if err != nil {
		return err
	}
	if err := file.Chown(identity.UID, gid); err != nil {
		file.Close()
		return fmt.Errorf("set restored file owner: %w", err)
	}
	if err := file.Chmod(os.FileMode(header.Mode) & 0o660); err != nil {
		file.Close()
		return fmt.Errorf("set restored file mode: %w", err)
	}
	_, copyErr := io.Copy(file, io.LimitReader(reader, header.Size))
	return errors.Join(copyErr, file.Close())
}

func (d *Driver) restoreIdentity(accountID, systemUser string) (hostuser.Identity, int, error) {
	found, err := d.lookup(systemUser)
	if err != nil {
		return hostuser.Identity{}, 0, fmt.Errorf("lookup backup account: %w", err)
	}
	identity, err := hostuser.Validate(found, d.home, accountID, systemUser)
	if err != nil {
		return hostuser.Identity{}, 0, err
	}
	group, err := d.lookupGroup(hostuser.WebGroup)
	if err != nil {
		return hostuser.Identity{}, 0, fmt.Errorf("lookup web server group: %w", err)
	}
	webGID, err := strconv.Atoi(group.Gid)
	if err != nil || webGID <= 0 {
		return hostuser.Identity{}, 0, fmt.Errorf("invalid web server GID")
	}
	return identity, webGID, nil
}

func (d *Driver) validateAccount(accountID, systemUser string) error {
	if validate.ID("accountId", accountID) != nil || validate.SystemUser(systemUser) != nil || systemUser != "wcp_"+accountID[:12] {
		return &validate.Error{Field: "accountId", Message: "The backup account identity is invalid"}
	}
	return nil
}

func (d *Driver) validateArtifact(request backup.ArtifactRequest) error {
	if validate.ID("accountId", request.AccountID) != nil || !checksumPattern.MatchString(request.Checksum) {
		return &validate.Error{Field: "artifact", Message: "The backup artifact identity is invalid"}
	}
	root := filepath.Join(d.root, request.AccountID)
	relative, err := filepath.Rel(root, request.Path)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.Ext(request.Path) != ".gz" {
		return &validate.Error{Field: "artifact", Message: "The backup artifact path is invalid"}
	}
	return nil
}

func validEntryPath(value string) bool {
	return value != "" && value == path.Clean(value) && !path.IsAbs(value) && value != ".." && !strings.HasPrefix(value, "../")
}

func safeTarget(root, relative string) (string, error) {
	if !validEntryPath(filepath.ToSlash(relative)) {
		return "", fmt.Errorf("unsafe restore path: %s", relative)
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	value, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(value, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("restore path escapes account home")
	}
	current := root
	for _, part := range strings.Split(value, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("restore target contains a symlink: %s", current)
		}
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}
	}
	return target, nil
}

func fileChecksum(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}
