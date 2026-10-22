package http_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"

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

// Handler-level specs for offerings, access, import and covers (real use cases, mocked repositories, real local storage).
var _ = Describe("courses module over HTTP", func() {
	const maxCover = 1 << 20
	var (
		ctrl      *gomock.Controller
		loader    *mocks.MockActorLoader
		offerings *mocks.MockOfferingRepository
		access    *mocks.MockAccessRepository
		imports   *mocks.MockImportRepository
		currs     *mocks.MockCurriculumRepository
		covers    *mocks.MockCoverRepository
		users     *mocks.MockUserRepository
		files     *memstore.Store
		router    *gin.Engine

		curID   = uuid.New()
		adminID = uuid.New()
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		loader = mocks.NewMockActorLoader(ctrl)
		offerings = mocks.NewMockOfferingRepository(ctrl)
		access = mocks.NewMockAccessRepository(ctrl)
		imports = mocks.NewMockImportRepository(ctrl)
		currs = mocks.NewMockCurriculumRepository(ctrl)
		covers = mocks.NewMockCoverRepository(ctrl)
		users = mocks.NewMockUserRepository(ctrl)
		files = memstore.New()

		allow := []string{"chula.ac.th"}
		routes := httpAdapter.Routes{API: []httpAdapter.Registrar{
			handler.NewOfferingHandler(
				queries.NewListOfferingsHandler(offerings), queries.NewListOfferingYearsHandler(offerings),
				commands.NewCreateOfferingHandler(offerings, currs), commands.NewUpdateOfferingHandler(offerings, currs),
				commands.NewDeleteOfferingsHandler(offerings, files), queries.NewListAccessHandler(offerings, access),
				commands.NewSetAccessHandler(offerings, access, users), commands.NewRemoveAccessHandler(offerings, access)),
			handler.NewImportHandler(commands.NewPreviewImportHandler(imports, currs, allow, port.DefaultMaxImportBytes), commands.NewApplyImportHandler(imports, currs, allow, port.DefaultMaxImportBytes), port.DefaultMaxImportBytes),
			handler.NewCoverHandler(commands.NewPutCoverHandler(currs, covers, files, maxCover),
				commands.NewGetCoverHandler(currs, files), commands.NewDeleteCoverHandler(currs, covers, files), maxCover),
		}}
		router = httpAdapter.SetupRouter(httpAdapter.RouterConfig{ActorLoader: loader}, handler.NewHealthHandler(mocks.NewMockPinger(ctrl)), routes)
	})
	AfterEach(func() { ctrl.Finish() })

	asAdmin := func() map[string]string {
		loader.EXPECT().LoadActor(gomock.Any(), adminID).Return(&domain.Actor{
			UserID: adminID, Role: domain.RoleAdmin, ScopeCurriculumIDs: []uuid.UUID{curID},
		}, nil)
		return map[string]string{"X-User-Id": adminID.String()}
	}
	asStaff := func() map[string]string {
		id := uuid.New()
		loader.EXPECT().LoadActor(gomock.Any(), id).Return(&domain.Actor{UserID: id, Role: domain.RoleStaff}, nil)
		return map[string]string{"X-User-Id": id.String()}
	}
	do := func(method, path, contentType string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	multipartBody := func(fields map[string]string, filename string, content []byte) ([]byte, string) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		for k, v := range fields {
			Expect(mw.WriteField(k, v)).To(Succeed())
		}
		if filename != "" {
			fw, err := mw.CreateFormFile("file", filename)
			Expect(err).NotTo(HaveOccurred())
			_, _ = fw.Write(content)
		}
		Expect(mw.Close()).To(Succeed())
		return buf.Bytes(), mw.FormDataContentType()
	}

	Describe("offerings", func() {
		It("requires academic_year (400) and routes /years before /:id", func() {
			w := do("GET", "/api/v1/offerings", "", nil, asAdmin())
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"academic_year is required"}`))

			offerings.EXPECT().Years(gomock.Any()).Return([]int{2025}, nil)
			w = do("GET", "/api/v1/offerings/years", "", nil, asStaff())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(`"items":[`))
			Expect(w.Body.String()).To(ContainSubstring(`"current":`))
		})

		It("lists offerings with the contract JSON shape", func() {
			offerings.EXPECT().List(gomock.Any(), gomock.Any()).Return([]domain.Offering{{
				ID: uuid.New(), CourseID: uuid.New(), CourseCode: "100", CourseName: "Calc", AcademicYear: 2026, Semester: 1,
				Curriculum: domain.Ref{ID: curID, Code: "CS", Name: "CS"}, Department: domain.Ref{ID: uuid.New(), Code: "D", Name: "D"},
				InstructorCount: 1, StaffCount: 2, HasSubmissions: true,
			}}, nil)
			w := do("GET", "/api/v1/offerings?academic_year=2026&semester=1&search=calc", "", nil, asAdmin())
			Expect(w.Code).To(Equal(http.StatusOK))
			for _, key := range []string{`"course_id"`, `"course_code":"100"`, `"academic_year":2026`, `"semester":1`, `"curriculum":{`, `"department":{`,
				`"instructor_count":1`, `"staff_count":2`, `"has_submissions":true`, `"created_at"`, `"updated_at"`} {
				Expect(w.Body.String()).To(ContainSubstring(key))
			}
		})

		It("answers 403 to staff on the list and 400 to a malformed body on create", func() {
			Expect(do("GET", "/api/v1/offerings?academic_year=2026", "", nil, asStaff()).Code).To(Equal(http.StatusForbidden))
			w := do("POST", "/api/v1/offerings", "application/json", []byte(`{"course_code":"1"}`), asAdmin())
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("bulk-deletes and reports the count; rejects an empty list", func() {
			a := uuid.New()
			offerings.EXPECT().ListByIDs(gomock.Any(), []uuid.UUID{a}).Return([]domain.Offering{{ID: a, Curriculum: domain.Ref{ID: curID}}}, nil)
			offerings.EXPECT().Delete(gomock.Any(), []uuid.UUID{a}).Return(nil, nil)
			w := do("POST", "/api/v1/offerings/bulk-delete", "application/json", []byte(`{"ids":["`+a.String()+`"]}`), asAdmin())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON(`{"deleted":1}`))

			w = do("POST", "/api/v1/offerings/bulk-delete", "application/json", []byte(`{"ids":[]}`), asAdmin())
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("serialises access entries including the default reason", func() {
			off := &domain.Offering{ID: uuid.New(), Curriculum: domain.Ref{ID: curID}, Department: domain.Ref{ID: uuid.New(), Code: "MATH-CS"}}
			offerings.EXPECT().GetByID(gomock.Any(), off.ID).Return(off, nil)
			access.EXPECT().List(gomock.Any(), off.ID).Return(nil, nil)
			access.EXPECT().ListDefaultStaff(gomock.Any(), off.ID, off.Department.ID).Return(nil, nil)
			w := do("GET", "/api/v1/offerings/"+off.ID.String()+"/access", "", nil, asAdmin())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON(`{"items":[]}`))
		})

		It("rejects a malformed id with 400", func() {
			Expect(do("PATCH", "/api/v1/offerings/xyz", "application/json", []byte(`{}`), asAdmin()).Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("mapping CSV import", func() {
		csvBody := func(csv string) ([]byte, string) {
			return multipartBody(map[string]string{"curriculum_id": curID.String(), "academic_year": "2026"}, "map.csv", []byte(csv))
		}

		It("previews with 200 and a contract-shaped plan", func() {
			currs.EXPECT().GetByID(gomock.Any(), curID).Return(&domain.Curriculum{ID: curID}, nil)
			imports.EXPECT().LoadState(gomock.Any(), curID, 2026, gomock.Any()).Return(&domain.ImportState{}, nil)
			body, ct := csvBody("course id,course name,semester\n100,Calc,1\n")
			w := do("POST", "/api/v1/offerings/import/preview", ct, body, asAdmin())
			Expect(w.Code).To(Equal(http.StatusOK))
			for _, key := range []string{`"applied":false`, `"courses_create":1`, `"courses_kept_with_uploads":0`, `"access":[]`, `"users":[]`, `"errors":[]`, `"action":"create"`} {
				Expect(w.Body.String()).To(ContainSubstring(key))
			}
		})

		It("answers 422 with the plan when apply finds row errors", func() {
			currs.EXPECT().GetByID(gomock.Any(), curID).Return(&domain.Curriculum{ID: curID}, nil)
			imports.EXPECT().Apply(gomock.Any(), gomock.Any()).Return(&domain.ImportPlan{
				Errors: []domain.ImportError{{Row: 3, Message: "semester must be 1, 2 or 3"}},
			}, domain.ErrUnprocessable)
			body, ct := csvBody("course id,course name,semester\n100,Calc,9\n")
			w := do("POST", "/api/v1/offerings/import/apply", ct, body, asAdmin())
			Expect(w.Code).To(Equal(http.StatusUnprocessableEntity))
			Expect(w.Body.String()).To(ContainSubstring(`"row":3`))
			Expect(w.Body.String()).To(ContainSubstring(`"applied":false`))
		})

		It("answers 403 to an out-of-scope curriculum, 400 to bad fields and 413 to an oversize file", func() {
			other := uuid.New()
			body, ct := multipartBody(map[string]string{"curriculum_id": other.String(), "academic_year": "2026"}, "m.csv", []byte("x"))
			Expect(do("POST", "/api/v1/offerings/import/preview", ct, body, asAdmin()).Code).To(Equal(http.StatusForbidden))

			body, ct = multipartBody(map[string]string{"curriculum_id": "nope", "academic_year": "2026"}, "m.csv", []byte("x"))
			Expect(do("POST", "/api/v1/offerings/import/preview", ct, body, asAdmin()).Code).To(Equal(http.StatusBadRequest))

			body, ct = multipartBody(map[string]string{"curriculum_id": curID.String(), "academic_year": "2026"}, "", nil)
			Expect(do("POST", "/api/v1/offerings/import/preview", ct, body, asAdmin()).Code).To(Equal(http.StatusBadRequest))

			body, ct = multipartBody(map[string]string{"curriculum_id": curID.String(), "academic_year": "2026"}, "m.csv", bytes.Repeat([]byte("a"), 6<<20+10))
			Expect(do("POST", "/api/v1/offerings/import/preview", ct, body, asAdmin()).Code).To(Equal(http.StatusRequestEntityTooLarge))
		})
	})

	Describe("covers", func() {
		url := func() string { return "/api/v1/curricula/" + curID.String() + "/cover" }
		pdf := []byte("%PDF-1.7\nhello cover")

		It("uploads a PDF, then serves it with the iframe-safe headers and download mode", func() {
			currs.EXPECT().GetByID(gomock.Any(), curID).Return(&domain.Curriculum{ID: curID, Department: domain.Ref{ID: uuid.New()}}, nil).Times(2)
			covers.EXPECT().Set(gomock.Any(), curID, "cover.pdf", int64(len(pdf)), gomock.Any()).Return(nil)
			body, ct := multipartBody(nil, "cover.pdf", pdf)
			w := do("PUT", url(), ct, body, asAdmin())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(files.Has("covers/" + curID.String() + ".pdf")).To(BeTrue())

			cover := &domain.Cover{Filename: "cover.pdf", SizeBytes: int64(len(pdf))}
			currs.EXPECT().GetByID(gomock.Any(), curID).Return(&domain.Curriculum{ID: curID, Cover: cover}, nil).Times(2)
			w = do("GET", url()+"?v=12345", "", nil, asAdmin())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.Bytes()).To(Equal(pdf))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/pdf"))
			Expect(w.Header().Get("Content-Disposition")).To(Equal(`inline; filename="cover.pdf"`))
			Expect(w.Header().Get("Cache-Control")).To(Equal("no-cache"))
			Expect(w.Header().Get("X-Frame-Options")).To(Equal("SAMEORIGIN"))

			w = do("GET", url()+"?download=1&v=1", "", nil, asAdmin())
			Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="cover.pdf"`))
			Expect(w.Header().Get("X-Frame-Options")).To(Equal("SAMEORIGIN"))
		})

		It("rejects a non-PDF with 400 and stores nothing", func() {
			currs.EXPECT().GetByID(gomock.Any(), curID).Return(&domain.Curriculum{ID: curID}, nil)
			body, ct := multipartBody(nil, "cover.pdf", []byte("MZ this is an exe"))
			w := do("PUT", url(), ct, body, asAdmin())
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(files.Has("covers/" + curID.String() + ".pdf")).To(BeFalse())
		})

		It("rejects a file over MAX_COVER_MB with 413 (declared size and body limit)", func() {
			big := append([]byte("%PDF-"), bytes.Repeat([]byte("a"), maxCover)...)
			body, ct := multipartBody(nil, "cover.pdf", big)
			w := do("PUT", url(), ct, body, asAdmin())
			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(files.Has("covers/" + curID.String() + ".pdf")).To(BeFalse())

			huge := bytes.Repeat([]byte("a"), 3*maxCover)
			body, ct = multipartBody(nil, "cover.pdf", append([]byte("%PDF-"), huge...))
			w = do("PUT", url(), ct, body, asAdmin())
			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
		})

		It("answers 400 without a file, 403 for staff, 404 when there is no cover, 204 on delete", func() {
			body, ct := multipartBody(map[string]string{"x": "y"}, "", nil)
			Expect(do("PUT", url(), ct, body, asAdmin()).Code).To(Equal(http.StatusBadRequest))
			Expect(do("GET", url(), "", nil, asStaff()).Code).To(Equal(http.StatusForbidden))

			currs.EXPECT().GetByID(gomock.Any(), curID).Return(&domain.Curriculum{ID: curID}, nil)
			w := do("GET", url(), "", nil, asAdmin())
			Expect(w.Code).To(Equal(http.StatusNotFound))

			currs.EXPECT().GetByID(gomock.Any(), curID).Return(&domain.Curriculum{ID: curID}, nil)
			covers.EXPECT().Clear(gomock.Any(), curID, gomock.Any()).Return(nil)
			Expect(do("DELETE", url(), "", nil, asAdmin()).Code).To(Equal(http.StatusNoContent))
		})

		It("keeps the multipart helper honest", func() {
			body, ct := multipartBody(map[string]string{"a": "b"}, "f.pdf", pdf)
			Expect(ct).To(HavePrefix("multipart/form-data"))
			Expect(strings.Contains(string(body), "f.pdf")).To(BeTrue())
		})
	})
})
