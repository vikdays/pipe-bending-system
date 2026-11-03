package documents

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"changeme/internal/apperror"
	"changeme/internal/storage"
)

func storeFixture(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if _, e = db.Exec("INSERT INTO users(id,username,password_hash,role,created_at) VALUES('actor','test.user','unused','admin',1)"); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"c-one", "c-two"} {
		if _, e = db.Exec("INSERT INTO certificates(id,reference,created_at,created_by) VALUES(?,?,?,?)", id, "Fictional "+id, "2026-01-01", "actor"); e != nil {
			t.Fatal(e)
		}
	}
	store, e := New(db, dir)
	if e != nil {
		t.Fatal(e)
	}
	return store, dir
}
func pdfFixture(t *testing.T, dir string) string {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Resources << >> /Contents 4 0 R >>", "<< /Length 0 >>\nstream\n\nendstream"}
	for i, obj := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 5\n0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)
	path := filepath.Join(dir, "Вымышленный сертификат.pdf")
	if e := os.WriteFile(path, b.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	return path
}
func TestAttachmentsDedupDetachRestartAndIntegrity(t *testing.T) {
	s, dir := storeFixture(t)
	source := pdfFixture(t, dir)
	first, e := s.AttachPDF("c-one", source, "actor")
	if e != nil {
		t.Fatal(e)
	}
	second, e := s.AttachPDF("c-two", source, "actor")
	if e != nil || first.ID != second.ID || *first.PageCount != 1 {
		t.Fatal("same original not deduplicated", e)
	}
	if _, e = s.AttachPDF("c-one", source, "actor"); e != nil {
		t.Fatal(e)
	}
	var n int
	s.db.QueryRow("SELECT COUNT(*) FROM document_link_events").Scan(&n)
	if n != 2 {
		t.Fatal("duplicate attach produced duplicate audit", n)
	}
	path, e := s.CertificatePDFPath("c-one", first.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Detach("c-one", first.ID, "actor"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CertificatePDFPath("c-one", first.ID); apperror.Describe(e).Code != apperror.NotFound {
		t.Fatal("detached association is still visible", e)
	}
	if _, e = s.CertificatePDFPath("c-two", first.ID); e != nil {
		t.Fatal("detaching removed other association", e)
	}
	if _, e = s.AttachPDF("c-one", source, "actor"); e != nil {
		t.Fatal("cannot reattach", e)
	}
	if e = s.db.Close(); e != nil {
		t.Fatal(e)
	}
	db, e := storage.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s, e = New(db, dir)
	if e != nil {
		t.Fatal(e)
	}
	files, e := s.ListAttachments("c-two")
	if e != nil || len(files) != 1 || files[0].ID != first.ID {
		t.Fatal("restart lost file", e)
	}
	original, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	tampered := append([]byte(nil), original...)
	tampered[len(tampered)-2] = 'X'
	if e = os.WriteFile(path, tampered, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CertificatePDFPath("c-two", first.ID); e == nil {
		t.Fatal("tampered original opened")
	}
	os.Remove(path)
	if e = os.Symlink(source, path); e != nil {
		t.Skip("Система не разрешает создание тестовой символической ссылки: ", e)
	}
	if _, e = s.ManagedPath(first.ID); e == nil {
		t.Fatal("symlink managed file accepted")
	}
}
func TestRejectInvalidAndOversizedDocuments(t *testing.T) {
	s, dir := storeFixture(t)
	bad := filepath.Join(dir, "fake.pdf")
	os.WriteFile(bad, []byte("%PDF-1.7\nnot a PDF"), 0600)
	if _, e := s.AttachPDF("c-one", bad, "actor"); apperror.Describe(e).Code != apperror.ValidationError {
		t.Fatal("corrupt PDF accepted", e)
	}
	wrong := filepath.Join(dir, "fake.txt")
	os.WriteFile(wrong, []byte("data"), 0600)
	if _, e := s.AttachPDF("c-one", wrong, "actor"); apperror.Describe(e).Code != apperror.ValidationError {
		t.Fatal("wrong format accepted", e)
	}
	huge := filepath.Join(dir, "huge.pdf")
	f, _ := os.Create(huge)
	f.Truncate(MaxFileSize + 1)
	f.Close()
	if _, e := s.AttachPDF("c-one", huge, "actor"); apperror.Describe(e).Code != apperror.ValidationError {
		t.Fatal("oversized accepted", e)
	}
	var n int
	s.db.QueryRow("SELECT COUNT(*) FROM document_files").Scan(&n)
	if n != 0 {
		t.Fatal("rejected originals left DB records")
	}
	if _, e := s.AttachPDF("missing", bad, "actor"); apperror.Describe(e).Code != apperror.NotFound {
		t.Fatal(e)
	}
}
func TestLocalPDFIntegration(t *testing.T) {
	first, second := os.Getenv("PIPE_TEST_PDF_ONE"), os.Getenv("PIPE_TEST_PDF_TWO")
	if first == "" || second == "" {
		t.Skip("set PIPE_TEST_PDF_ONE/TWO to check local originals")
	}
	s, _ := storeFixture(t)
	for i, path := range []string{first, second} {
		a, e := s.AttachPDF("c-one", path, "actor")
		if e != nil {
			t.Fatal(e)
		}
		want := []int{8, 4}[i]
		if a.PageCount == nil || *a.PageCount != want {
			t.Fatal("unexpected original page count", a.PageCount)
		}
		if _, e = s.CertificatePDFPath("c-one", a.ID); e != nil {
			t.Fatal(e)
		}
	}
	t.Log("both local scan originals validated, stored and reopened with matching hashes")
}
