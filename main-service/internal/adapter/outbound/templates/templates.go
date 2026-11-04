// Package templates serves the downloadable templates: the instructor package folder (zipped on the fly)
// and the sample course/instructor mapping CSV.
package templates

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ZipRoot is the folder name every entry of the instructor package zip lives in.
const ZipRoot = "course-id-course-name"

// mappingCSV is the documented format of the mapping import (api.md): header + 2 example rows.
const mappingCSV = "course id,course name,semester,user email,user role,upload permission,download permission\r\n" +
	"2301173,Calculus I,1,instructor@chula.ac.th,Instructor,Yes,Yes\r\n" +
	"2301173,Calculus I,1,staff@chula.ac.th,Staff,No,Yes\r\n"

// Service implements port.TemplateService.
type Service struct {
	packageDir string
	storage    port.Storage // holds the custom template (System Config); nil = always the default
}

var _ port.TemplateService = (*Service)(nil)

// New returns a service that serves the custom template from storage when one was uploaded, else zips packageDir
// (INSTRUCTOR_PACKAGE_DIR).
func New(packageDir string, storage port.Storage) *Service {
	return &Service{packageDir: packageDir, storage: storage}
}

func (s *Service) CourseMappingCSV() []byte { return []byte(mappingCSV) }

// WriteInstructorPackage streams the custom template if one exists; otherwise (or on any storage error, so the
// download never breaks) the package directory as a zip below "course-id-course-name/".
func (s *Service) WriteInstructorPackage(w io.Writer) error {
	if s.storage != nil {
		rc, _, err := s.storage.Get(context.Background(), port.CourseTemplateKey)
		if err == nil {
			defer func() { _ = rc.Close() }()
			_, err = io.Copy(w, rc)
			return err
		}
		if !errors.Is(err, domain.ErrNotFound) {
			slog.Error("custom course template unavailable, serving the default", "error", err)
		}
	}
	info, err := os.Stat(s.packageDir)
	if err != nil {
		return fmt.Errorf("instructor package directory %q is not available: %w", s.packageDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("instructor package path %q is not a directory", s.packageDir)
	}
	zw := zip.NewWriter(w)
	root := os.DirFS(s.packageDir)
	err = fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		name := path.Join(ZipRoot, p)
		fi, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			_, err := zw.CreateHeader(&zip.FileHeader{Name: name + "/", Method: zip.Store, Modified: fi.ModTime()})
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		method := zip.Deflate
		if strings.HasSuffix(strings.ToLower(p), ".pdf") {
			method = zip.Store
		}
		ew, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method, Modified: fi.ModTime()})
		if err != nil {
			return err
		}
		f, err := root.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(ew, f)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}
