package http_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	httpAdapter "abet/main-service/internal/adapter/inbound/http"
	"abet/main-service/internal/adapter/inbound/http/handler"
	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/application/queries"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
	"abet/main-service/internal/testsupport/memstore"
)

type fakeTemplates struct{}

func (fakeTemplates) WriteInstructorPackage(w io.Writer) error {
	zw := zip.NewWriter(w)
	f, _ := zw.Create("course-id-course-name/README_FIRST.txt")
	_, _ = f.Write([]byte("hi"))
	return zw.Close()
}
func (fakeTemplates) CourseMappingCSV() []byte { return []byte("a,b\r\n") }

var _ = Describe("submissions and templates over HTTP", func() {
	const maxUploadMB = 1
	var (
		ctrl      *gomock.Controller
		loader    *mocks.MockActorLoader
		repo      *mocks.MockSubmissionRepository
		publisher *mocks.MockJobPublisher
		files     *memstore.Store
		router    *gin.Engine

		userID = uuid.New()
		offID  = uuid.New()
		off    domain.OfferingInfo
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		loader = mocks.NewMockActorLoader(ctrl)
		repo = mocks.NewMockSubmissionRepository(ctrl)
		publisher = mocks.NewMockJobPublisher(ctrl)
		files = memstore.New()
		off = domain.OfferingInfo{ID: offID, CourseCode: "1234567", CourseName: "Calc", AcademicYear: 2026, Semester: 1,
			Curriculum: domain.Ref{ID: uuid.New(), Code: "CS", Name: "CS"}, DepartmentID: uuid.New(),
			Department: domain.Ref{ID: uuid.New(), Code: "D", Name: "Dept"}}

		routes := httpAdapter.Routes{API: []httpAdapter.Registrar{
			handler.NewSubmissionHandler(
				commands.NewUploadSubmissionHandler(repo, files, publisher, GinkgoT().TempDir()),
				queries.NewGetSubmissionHandler(repo), queries.NewListSubmissionsHandler(repo),
				queries.NewGetPortfolioPDFHandler(repo, files), queries.NewListPortfoliosHandler(repo),
				queries.NewDownloadPortfoliosHandler(repo, files), maxUploadMB),
			handler.NewTemplateHandler(fakeTemplates{}),
		}}
		router = httpAdapter.SetupRouter(httpAdapter.RouterConfig{ActorLoader: loader}, handler.NewHealthHandler(mocks.NewMockPinger(ctrl)), routes)
	})
	AfterEach(func() { ctrl.Finish() })

	login := func(role domain.Role) {
		loader.EXPECT().LoadActor(gomock.Any(), userID).Return(&domain.Actor{UserID: userID, Email: "u@chula.ac.th", Role: role}, nil)
	}
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		req.Header.Set("X-User-Id", userID.String())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	multipartBody := func(field, filename string, content []byte) (*bytes.Buffer, string) {
		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		fw, err := mw.CreateFormFile(field, filename)
		Expect(err).NotTo(HaveOccurred())
		_, _ = fw.Write(content)
		Expect(mw.Close()).To(Succeed())
		return &b, mw.FormDataContentType()
	}
	upload := func(field, filename string, content []byte) *httptest.ResponseRecorder {
		body, ct := multipartBody(field, filename, content)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/offerings/"+offID.String()+"/submissions", body)
		req.Header.Set("Content-Type", ct)
		return serve(req)
	}
	decode := func(w *httptest.ResponseRecorder) map[string]any {
		var m map[string]any
		Expect(json.Unmarshal(w.Body.Bytes(), &m)).To(Succeed(), w.Body.String())
		return m
	}

	Describe("POST /offerings/{id}/submissions", func() {
		It("answers 413 with the contract message right away when Content-Length is over the limit", func() {
			login(domain.RoleSuperAdmin)
			big := make([]byte, 3<<20)
			_, _ = rand.Read(big)
			w := upload("file", "big.zip", big)
			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(decode(w)).To(Equal(map[string]any{"error": "file exceeds 1 MB limit"}))
		})

		It("answers 413 for a streamed (chunked) body that grows past the limit and keeps nothing", func() {
			login(domain.RoleSuperAdmin)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			big := make([]byte, 3<<20)
			_, _ = rand.Read(big)
			body, ct := multipartBody("file", "big.zip", big)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/offerings/"+offID.String()+"/submissions", body)
			req.ContentLength = -1
			req.Header.Set("Content-Type", ct)
			w := serve(req)
			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(decode(w)).To(Equal(map[string]any{"error": "file exceeds 1 MB limit"}))
			Expect(files.Keys("")).To(BeEmpty())
		})

		It("answers 400 when the file field is missing or the body is not multipart", func() {
			login(domain.RoleSuperAdmin)
			w := upload("other", "a.zip", []byte("x"))
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(decode(w)).To(HaveKeyWithValue("error", "file is required"))

			login(domain.RoleSuperAdmin)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/offerings/"+offID.String()+"/submissions", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			Expect(serve(req).Code).To(Equal(http.StatusBadRequest))
		})

		It("answers 400 for a file that is not a zip and 403 without upload permission", func() {
			login(domain.RoleSuperAdmin)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			w := upload("file", "a.zip", []byte("plain text"))
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(decode(w)).To(HaveKeyWithValue("error", "the file is not a valid zip archive"))

			login(domain.RoleInstructor)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			repo.EXPECT().Grants(gomock.Any(), userID, gomock.Any()).Return(nil, nil)
			w = upload("file", "a.zip", []byte("plain text"))
			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(decode(w)).To(Equal(map[string]any{"error": "forbidden"}))
		})

		It("answers 201 with the exact Submission JSON (invalid package: immediate issues, no job)", func() {
			login(domain.RoleSuperAdmin)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			var created *domain.Submission
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.Submission) error {
				created = s
				return nil
			})
			repo.EXPECT().GetByID(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, uuid.UUID) (*domain.Submission, error) {
				created.UploadedBy.Name = "Alice"
				return created, nil
			})

			var zb bytes.Buffer
			zw := zip.NewWriter(&zb)
			f, _ := zw.Create("notes.txt")
			_, _ = f.Write([]byte("x"))
			Expect(zw.Close()).To(Succeed())

			w := upload("file", "mine.zip", zb.Bytes())
			Expect(w.Code).To(Equal(http.StatusCreated), w.Body.String())
			m := decode(w)
			Expect(m).To(HaveKey("id"))
			Expect(m).To(HaveKeyWithValue("offering_id", offID.String()))
			Expect(m).To(HaveKeyWithValue("status", "invalid"))
			Expect(m).To(HaveKeyWithValue("original_filename", "mine.zip"))
			Expect(m["size_bytes"]).To(BeNumerically("==", zb.Len()))
			Expect(m["uploaded_by"]).To(Equal(map[string]any{"id": userID.String(), "name": "Alice", "email": "u@chula.ac.th"}))
			Expect(m["error_count"]).To(BeNumerically(">", 0))
			Expect(m).To(HaveKey("warning_count"))
			Expect(m["created_at"]).To(MatchRegexp(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$`))
			Expect(m["completed_at"]).NotTo(BeNil())
			issues := m["issues"].([]any)
			Expect(issues).NotTo(BeEmpty())
			first := issues[0].(map[string]any)
			Expect(first).To(HaveLen(7))
			for _, k := range []string{"severity", "stage", "code", "title", "description", "location", "suggested_fix"} {
				Expect(first).To(HaveKey(k))
			}
		})
	})

	Describe("GET /submissions/{id} and history", func() {
		It("renders [] issues in the history list and null/empty fields explicitly", func() {
			login(domain.RoleSuperAdmin)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			repo.EXPECT().ListByOffering(gomock.Any(), offID).Return([]domain.Submission{
				{ID: uuid.New(), OfferingID: offID, Status: domain.SubmissionQueued, CreatedAt: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)},
			}, nil)
			w := serve(httptest.NewRequest(http.MethodGet, "/api/v1/offerings/"+offID.String()+"/submissions", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			items := decode(w)["items"].([]any)
			Expect(items).To(HaveLen(1))
			item := items[0].(map[string]any)
			Expect(item["issues"]).To(Equal([]any{}))
			Expect(item["uploaded_by"]).To(BeNil())
			Expect(item["completed_at"]).To(BeNil())
			Expect(item["created_at"]).To(Equal("2026-10-09T10:00:00Z"))
		})

		It("is 403 for a user without access and 400 for a bad id", func() {
			sub := &domain.Submission{ID: uuid.New(), OfferingID: offID}
			login(domain.RoleInstructor)
			repo.EXPECT().GetByID(gomock.Any(), sub.ID).Return(sub, nil)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			repo.EXPECT().Grants(gomock.Any(), userID, gomock.Any()).Return(nil, nil)
			Expect(serve(httptest.NewRequest(http.MethodGet, "/api/v1/submissions/"+sub.ID.String(), nil)).Code).To(Equal(http.StatusForbidden))

			login(domain.RoleInstructor)
			Expect(serve(httptest.NewRequest(http.MethodGet, "/api/v1/submissions/zzz", nil)).Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GET /offerings/{id}/portfolio.pdf", func() {
		It("sends the PDF as an attachment with the contract file name, 404 when none", func() {
			sid := uuid.New()
			files.Seed("portfolios/"+sid.String()+".pdf", "%PDF-1.4 body")
			login(domain.RoleSuperAdmin)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			repo.EXPECT().LatestGenerated(gomock.Any(), offID).Return(sid, nil)
			w := serve(httptest.NewRequest(http.MethodGet, "/api/v1/offerings/"+offID.String()+"/portfolio.pdf", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/pdf"))
			Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="1234567_2026_S1_portfolio.pdf"`))
			Expect(w.Header().Get("Content-Length")).To(Equal("13"))
			Expect(w.Body.String()).To(Equal("%PDF-1.4 body"))

			login(domain.RoleSuperAdmin)
			repo.EXPECT().GetOffering(gomock.Any(), offID).Return(&off, nil)
			repo.EXPECT().LatestGenerated(gomock.Any(), offID).Return(uuid.Nil, domain.ErrNotFound)
			w = serve(httptest.NewRequest(http.MethodGet, "/api/v1/offerings/"+offID.String()+"/portfolio.pdf", nil))
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(decode(w)).To(HaveKey("error"))
		})
	})

	Describe("POST /portfolios/download", func() {
		post := func(body string) *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolios/download", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			return serve(req)
		}
		It("is 400 for an empty list, 404 when nothing is available", func() {
			login(domain.RoleSuperAdmin)
			Expect(post(`{"offering_ids":[]}`).Code).To(Equal(http.StatusBadRequest))
			login(domain.RoleSuperAdmin)
			Expect(post(`{}`).Code).To(Equal(http.StatusBadRequest))

			login(domain.RoleSuperAdmin)
			repo.EXPECT().ListOfferings(gomock.Any(), []uuid.UUID{offID}).Return([]domain.OfferingInfo{off}, nil)
			repo.EXPECT().LatestGeneratedFor(gomock.Any(), gomock.Any()).Return(map[uuid.UUID]uuid.UUID{}, nil)
			Expect(post(`{"offering_ids":["` + offID.String() + `"]}`).Code).To(Equal(http.StatusNotFound))
		})
		It("streams a zip with the contract name", func() {
			sid := uuid.New()
			files.Seed("portfolios/"+sid.String()+".pdf", "%PDF-1.4")
			login(domain.RoleSuperAdmin)
			repo.EXPECT().ListOfferings(gomock.Any(), gomock.Any()).Return([]domain.OfferingInfo{off}, nil)
			repo.EXPECT().LatestGeneratedFor(gomock.Any(), gomock.Any()).Return(map[uuid.UUID]uuid.UUID{offID: sid}, nil)
			w := post(`{"offering_ids":["` + offID.String() + `"]}`)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/zip"))
			Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="ABET_Course_Portfolios_2026.zip"`))
			zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
			Expect(err).NotTo(HaveOccurred())
			Expect(zr.File).To(HaveLen(1))
			Expect(zr.File[0].Name).To(Equal("1234567_2026_S1_portfolio.pdf"))
		})
	})

	Describe("GET /portfolios", func() {
		It("returns PortfolioRow items with flags, latest submission and pdf_available", func() {
			login(domain.RoleSuperAdmin)
			subID := uuid.New()
			repo.EXPECT().ListPortfolioRows(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *domain.Actor, f port.PortfolioFilter) ([]port.PortfolioCandidate, error) {
					Expect(*f.AcademicYear).To(Equal(2026))
					Expect(*f.Semester).To(Equal(1))
					Expect(f.Search).To(Equal("calc"))
					return []port.PortfolioCandidate{{Offering: off, PDFAvailable: true,
						Latest: &domain.Submission{ID: subID, OfferingID: offID, Status: domain.SubmissionGenerated,
							UploadedBy: &domain.UserRef{ID: userID, Name: "A", Email: "a@chula.ac.th"}}}}, nil
				})
			w := serve(httptest.NewRequest(http.MethodGet, "/api/v1/portfolios?academic_year=2026&semester=1&search=calc", nil))
			Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
			row := decode(w)["items"].([]any)[0].(map[string]any)
			for _, k := range []string{"offering_id", "course_code", "course_name", "academic_year", "semester", "curriculum", "department", "can_upload", "can_download", "latest_submission", "pdf_available"} {
				Expect(row).To(HaveKey(k), k)
			}
			Expect(row["can_upload"]).To(BeTrue())
			Expect(row["pdf_available"]).To(BeTrue())
			Expect(row["curriculum"]).To(HaveKeyWithValue("code", "CS"))
			latest := row["latest_submission"].(map[string]any)
			Expect(latest["status"]).To(Equal("generated"))
			Expect(latest["issues"]).To(Equal([]any{}))
		})
		It("answers 400 for malformed filters", func() {
			for _, q := range []string{"academic_year=x", "semester=9", "department_id=zzz", "curriculum_id=zzz"} {
				login(domain.RoleSuperAdmin)
				Expect(serve(httptest.NewRequest(http.MethodGet, "/api/v1/portfolios?"+q, nil)).Code).To(Equal(http.StatusBadRequest), q)
			}
		})
	})

	Describe("templates", func() {
		It("serves the mapping CSV as an attachment with the right content type", func() {
			login(domain.RoleStaff)
			w := serve(httptest.NewRequest(http.MethodGet, "/api/v1/templates/course-instructor-mapping.csv", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/csv; charset=utf-8"))
			Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="course-instructor-mapping.csv"`))
			Expect(w.Body.String()).To(Equal("a,b\r\n"))
		})
		It("serves the instructor package zip to any authenticated user and requires authentication", func() {
			login(domain.RoleInstructor)
			w := serve(httptest.NewRequest(http.MethodGet, "/api/v1/templates/instructor-package.zip", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/zip"))
			Expect(w.Header().Get("Content-Disposition")).To(ContainSubstring("course-template.zip"))
			zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
			Expect(err).NotTo(HaveOccurred())
			Expect(zr.File[0].Name).To(Equal("course-id-course-name/README_FIRST.txt"))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/instructor-package.zip", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
