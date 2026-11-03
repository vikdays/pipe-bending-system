package documents

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"changeme/internal/apperror"
	"changeme/internal/storage"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const MaxFileSize = 50 * 1024 * 1024

var errStorage = apperror.New(apperror.InternalError, "Не удалось сохранить или прочитать оригинал документа.")
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type File struct {
	ID           string `json:"id"`
	SHA256       string `json:"sha256"`
	Kind         string `json:"kind"`
	OriginalName string `json:"originalName"`
	SizeBytes    int64  `json:"sizeBytes"`
	PageCount    *int   `json:"pageCount"`
	CreatedAt    string `json:"createdAt"`
}

type Store struct {
	db   *sql.DB
	root string
}

func New(db *sql.DB, dataDirectory string) (*Store, error) {
	root := filepath.Join(dataDirectory, "documents")
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	if err := storage.ProtectManagedPath(root, true); err != nil {
		return nil, err
	}
	return &Store{db: db, root: root}, nil
}

// Ingest copies first, validates that exact private copy, then records immutable content.
// Validator must not calculate workbook formulas or contact external services.
func (s *Store) Ingest(sourcePath, kind, actorID string, validate func(*os.File) error) (File, error) {
	if (kind != "pdf" && kind != "xlsx") || !strings.EqualFold(filepath.Ext(sourcePath), "."+kind) {
		return File{}, apperror.New(apperror.ValidationError, "Выберите файл нужного формата: .xlsx или .pdf.")
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return File{}, apperror.New(apperror.ValidationError, "Не удалось прочитать выбранный файл.")
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > MaxFileSize {
		return File{}, apperror.New(apperror.ValidationError, "Нужен непустой обычный файл размером не более 50 МБ.")
	}
	if err = storage.ProtectManagedPath(s.root, true); err != nil {
		return File{}, errStorage
	}
	temporary, err := os.CreateTemp(s.root, "incoming-*")
	if err != nil {
		return File{}, errStorage
	}
	defer func() { temporary.Close(); os.Remove(temporary.Name()) }()
	if err = storage.ProtectManagedPath(temporary.Name(), false); err != nil {
		return File{}, errStorage
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(source, MaxFileSize+1))
	if err != nil || size <= 0 || size > MaxFileSize {
		return File{}, apperror.New(apperror.ValidationError, "Файл не удалось скопировать или его размер превышает 50 МБ.")
	}
	if err = temporary.Sync(); err != nil {
		return File{}, errStorage
	}
	if _, err = temporary.Seek(0, io.SeekStart); err != nil {
		return File{}, errStorage
	}
	var pageCount *int
	if kind == "pdf" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		conf := model.NewStatelessConfiguration()
		conf.Offline = true
		pages, parseErr := api.PageCount(ctx, temporary, conf)
		if parseErr != nil || pages <= 0 {
			return File{}, apperror.New(apperror.ValidationError, "PDF повреждён, защищён паролем или не поддерживается. Выберите другой оригинал.")
		}
		pageCount = &pages
	}
	if validate != nil {
		if _, err = temporary.Seek(0, io.SeekStart); err != nil {
			return File{}, errStorage
		}
		if err = validate(temporary); err != nil {
			return File{}, err
		}
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	relative := filepath.Join(kind, digest[:2], digest+"."+kind)
	parent := filepath.Join(s.root, kind)
	if err = secureDirectory(parent); err != nil {
		return File{}, errStorage
	}
	parent = filepath.Join(parent, digest[:2])
	if err = secureDirectory(parent); err != nil {
		return File{}, errStorage
	}
	target := filepath.Join(s.root, relative)
	if err = temporary.Close(); err != nil {
		return File{}, errStorage
	}
	if err = publish(temporary.Name(), target, digest, size); err != nil {
		return File{}, errStorage
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	name := safeName(filepath.Base(sourcePath))
	_, err = s.db.Exec("INSERT INTO document_files(id,sha256,kind,original_name,relative_path,size_bytes,page_count,created_at,created_by) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(sha256) DO NOTHING", "d_"+rand.Text(), digest, kind, name, filepath.ToSlash(relative), size, pageCount, now, actorID)
	if err != nil {
		return File{}, errStorage
	}
	var id string
	if err = s.db.QueryRow("SELECT id FROM document_files WHERE sha256=? AND kind=?", digest, kind).Scan(&id); err != nil {
		return File{}, errStorage
	}
	file, err := s.Get(id)
	if err != nil || file.SizeBytes != size {
		return File{}, errStorage
	}
	return file, nil
}

func safeName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '_'
		}
		return r
	}, name)
}
func secureDirectory(path string) error {
	if err := os.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return storage.ProtectManagedPath(path, true)
}
func publish(source, target, digest string, size int64) error {
	// Both paths are on the private data volume. Linking publishes the already
	// synced original atomically, without replacing existing content. A crash
	// can leave a complete orphan, never a half-written content-addressed file.
	err := os.Link(source, target)
	if errors.Is(err, os.ErrExist) {
		return verifyFile(target, digest, size)
	}
	if err != nil {
		return err
	}
	return verifyFile(target, digest, size)
}
func verifyFile(path, digest string, size int64) error {
	if err := storage.ProtectManagedPath(path, false); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() != size {
		return errStorage
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != digest {
		return errStorage
	}
	return nil
}
func (s *Store) Get(id string) (File, error) {
	var file File
	err := s.db.QueryRow("SELECT id,sha256,kind,original_name,size_bytes,page_count,created_at FROM document_files WHERE id=?", id).Scan(&file.ID, &file.SHA256, &file.Kind, &file.OriginalName, &file.SizeBytes, &file.PageCount, &file.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, apperror.New(apperror.NotFound, "Документ не найден.")
	}
	if err != nil {
		return File{}, errStorage
	}
	return file, nil
}

// ManagedPath accepts a document ID only; no filesystem path comes from the WebView.
func (s *Store) ManagedPath(id string) (string, error) {
	file, err := s.Get(id)
	if err != nil {
		return "", err
	}
	if !digestPattern.MatchString(file.SHA256) || (file.Kind != "xlsx" && file.Kind != "pdf") {
		return "", errStorage
	}
	for _, path := range []string{s.root, filepath.Join(s.root, file.Kind), filepath.Join(s.root, file.Kind, file.SHA256[:2])} {
		if err = storage.ProtectManagedPath(path, true); err != nil {
			return "", errStorage
		}
	}
	path := filepath.Join(s.root, file.Kind, file.SHA256[:2], file.SHA256+"."+file.Kind)
	if err = verifyFile(path, file.SHA256, file.SizeBytes); err != nil {
		return "", apperror.New(apperror.InternalError, "Оригинал документа отсутствует или изменён. Восстановите файл из резервной копии.")
	}
	return path, nil
}
