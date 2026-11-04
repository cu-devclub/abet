package http_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	httpAdapter "abet/main-service/internal/adapter/inbound/http"
	"abet/main-service/internal/adapter/inbound/http/handler"
	"abet/main-service/internal/adapter/outbound/templates"
	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
	"abet/main-service/internal/core/validation"
	"abet/main-service/internal/testsupport/memstore"
)

var _ = Describe("course template (System Config) over HTTP", func() {
	const maxMB = 1
	var (
		router  *gin.Engine
		loader  *mocks.MockActorLoader
		files   *memstore.Store
		tmpDir  string
		userID  = uuid.New()
		pkgDir  string
		ctrl    *gomock.Controller
		tmplSvc *templates.Service
	)

	zipOf := func(names ...string) []byte {
		var b bytes.Buffer
		zw := zip.NewWriter(&b)
		for _, n := range names {
			w, err := zw.Create(n)
			Expect(err).NotTo(HaveOccurred())
			_, _ = w.Write([]byte("x"))
		}
		Expect(zw.Close()).To(Succeed())
		return b.Bytes()
	}
	core := func(prefix string) []string {
		var out []string
		for _, p := range validation.CoreCSVPaths() {
			out = append(out, prefix+p)
		}
		return out
	}

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		loader = mocks.NewMockActorLoader(ctrl)
		files = memstore.New()
		tmpDir = GinkgoT().TempDir()
		pkgDir = GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(pkgDir, "default.txt"), []byte("d"), 0o644)).To(Succeed())
		tmplSvc = templates.New(pkgDir, files)
		routes := httpAdapter.Routes{API: []httpAdapter.Registrar{
			handler.NewTemplateHandler(tmplSvc),
			handler.NewCourseTemplateHandler(
				commands.NewGetCourseTemplateHandler(files),
				commands.NewPutCourseTemplateHandler(files, tmpDir, maxMB<<20),
				commands.NewDeleteCourseTemplateHandler(files), maxMB<<20),
		}}
		router = httpAdapter.SetupRouter(httpAdapter.RouterConfig{ActorLoader: loader}, handler.NewHealthHandler(mocks.NewMockPinger(ctrl)), routes)
	})
	AfterEach(func() {
		ctrl.Finish()
		left, _ := filepath.Glob(filepath.Join(tmpDir, "*"))
		Expect(left).To(BeEmpty(), "temp files must always be removed")
	})

	call := func(role domain.Role, method string, body []byte) *httptest.ResponseRecorder {
		loader.EXPECT().LoadActor(gomock.Any(), userID).Return(&domain.Actor{UserID: userID, Email: "root@chula.ac.th", Role: role}, nil)
		var rd io.Reader
		ct := ""
		if body != nil {
			var b bytes.Buffer
			mw := multipart.NewWriter(&b)
			fw, _ := mw.CreateFormFile("file", "t.zip")
			_, _ = fw.Write(body)
			_ = mw.Close()
			rd, ct = &b, mw.FormDataContentType()
		}
		req := httptest.NewRequest(method, "/api/v1/config/course-template", rd)
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		req.Header.Set("X-User-Id", userID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	download := func() *httptest.ResponseRecorder {
		loader.EXPECT().LoadActor(gomock.Any(), userID).Return(&domain.Actor{UserID: userID, Role: domain.RoleStaff}, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/instructor-package.zip", nil)
		req.Header.Set("X-User-Id", userID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	errMsg := func(w *httptest.ResponseRecorder) string {
		var m map[string]string
		Expect(json.Unmarshal(w.Body.Bytes(), &m)).To(Succeed())
		return m["error"]
	}

	It("answers 403 to every non-super role on GET, PUT and DELETE", func() {
		for _, role := range []domain.Role{domain.RoleAdmin, domain.RoleInstructor, domain.RoleStaff} {
			for _, m := range []string{"GET", "PUT", "DELETE"} {
				var body []byte
				if m == "PUT" {
					body = zipOf(core("")...)
				}
				w := call(role, m, body)
				Expect(w.Code).To(Equal(http.StatusForbidden), "%s %s", role, m)
				Expect(w.Body.String()).To(MatchJSON(`{"error":"forbidden"}`))
			}
		}
	})

	It("reports the default when nothing was uploaded", func() {
		w := call(domain.RoleSuperAdmin, "GET", nil)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(MatchJSON(`{"source":"default","size_bytes":null,"uploaded_at":null,"uploaded_by":null}`))
	})

	DescribeTable("rejects an invalid upload with 400 and keeps the default",
		func(body []byte, contains ...string) {
			w := call(domain.RoleSuperAdmin, "PUT", body)
			Expect(w.Code).To(Equal(http.StatusBadRequest), w.Body.String())
			for _, c := range contains {
				Expect(errMsg(w)).To(ContainSubstring(c))
			}
			Expect(files.Has(port.CourseTemplateKey)).To(BeFalse())
		},
		Entry("not a zip", []byte("this is not a zip"), "not a valid zip"),
		Entry("zip-slip ..", zipOf(append(core(""), "../evil.txt")...), "unsafe path"),
		Entry("absolute path", zipOf(append(core(""), "/etc/passwd")...), "unsafe path"),
		Entry("backslash", zipOf(append(core(""), `a\b.txt`)...), "unsafe path"),
		Entry("missing core csv", zipOf(core("")[:3]...), "missing required files", validation.CoreCSVPaths()[3]),
		Entry("csvs split over two folders", zipOf(append(core("a/")[:2], core("b/")[2:]...)...), "missing required files"),
	)

	It("rejects a file over MAX_TEMPLATE_MB with 413", func() {
		big := zipOf(core("")...)
		big = append(big, make([]byte, maxMB<<20)...)
		w := call(domain.RoleSuperAdmin, "PUT", big)
		Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
		Expect(errMsg(w)).To(Equal("file too large (max 1 MB)"))
	})

	DescribeTable("accepts the core CSVs at the root or under one top folder",
		func(prefix string) {
			w := call(domain.RoleSuperAdmin, "PUT", zipOf(core(prefix)...))
			Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
		},
		Entry("root", ""),
		Entry("one top folder", "course-id-course-name/"),
	)

	It("PUT -> GET custom -> download streams the custom bytes -> DELETE -> default", func() {
		custom := zipOf(core("top/")...)
		w := call(domain.RoleSuperAdmin, "PUT", custom)
		Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
		var got map[string]any
		Expect(json.Unmarshal(w.Body.Bytes(), &got)).To(Succeed())
		Expect(got["source"]).To(Equal("custom"))
		Expect(got["size_bytes"]).To(BeNumerically("==", len(custom)))
		Expect(got["uploaded_by"]).To(Equal("root@chula.ac.th"))
		Expect(got["uploaded_at"]).NotTo(BeNil())

		w = call(domain.RoleSuperAdmin, "GET", nil)
		Expect(w.Body.String()).To(ContainSubstring(`"source":"custom"`))

		d := download()
		Expect(d.Code).To(Equal(http.StatusOK))
		Expect(d.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="course-template.zip"`))
		Expect(d.Body.Bytes()).To(Equal(custom))

		w = call(domain.RoleSuperAdmin, "DELETE", nil)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(MatchJSON(`{"source":"default","size_bytes":null,"uploaded_at":null,"uploaded_by":null}`))
		Expect(call(domain.RoleSuperAdmin, "DELETE", nil).Code).To(Equal(http.StatusOK)) // idempotent
		d = download()
		zr, err := zip.NewReader(bytes.NewReader(d.Body.Bytes()), int64(d.Body.Len()))
		Expect(err).NotTo(HaveOccurred())
		Expect(zr.File[len(zr.File)-1].Name).To(HaveSuffix("default.txt"))
	})

	It("falls back to the default download when the store fails", func() {
		broken := templates.New(pkgDir, brokenGet{files})
		var buf bytes.Buffer
		Expect(broken.WriteInstructorPackage(&buf)).To(Succeed())
		zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Join(zipNames(zr), ",")).To(ContainSubstring("default.txt"))
	})
})

// brokenGet fails Get with a non-"not found" error, like a MinIO outage.
type brokenGet struct{ *memstore.Store }

func (brokenGet) Get(_ context.Context, _ string) (io.ReadCloser, port.ObjectInfo, error) {
	return nil, port.ObjectInfo{}, errors.New("minio down")
}

func zipNames(zr *zip.Reader) []string {
	var n []string
	for _, f := range zr.File {
		n = append(n, f.Name)
	}
	return n
}
