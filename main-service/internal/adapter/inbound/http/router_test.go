package http_test

import (
	"context"
	"errors"
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
)

const internalToken = "s3cret-token"

var _ = Describe("router", func() {
	var (
		ctrl    *gomock.Controller
		loader  *mocks.MockActorLoader
		users   *mocks.MockUserRepository
		depts   *mocks.MockDepartmentRepository
		currs   *mocks.MockCurriculumRepository
		storage *mocks.MockStorage
		pinger  *mocks.MockPinger
		router  *gin.Engine

		superID = uuid.New()
		staffID = uuid.New()
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		loader = mocks.NewMockActorLoader(ctrl)
		users = mocks.NewMockUserRepository(ctrl)
		depts = mocks.NewMockDepartmentRepository(ctrl)
		currs = mocks.NewMockCurriculumRepository(ctrl)
		storage = mocks.NewMockStorage(ctrl)
		pinger = mocks.NewMockPinger(ctrl)

		routes := httpAdapter.Routes{
			API: []httpAdapter.Registrar{
				handler.NewMeHandler(queries.NewGetMeHandler(users)),
				handler.NewDepartmentHandler(
					commands.NewCreateDepartmentHandler(depts), commands.NewUpdateDepartmentHandler(depts),
					commands.NewDeleteDepartmentHandler(depts), queries.NewListDepartmentsHandler(depts)),
				handler.NewCurriculumHandler(
					commands.NewCreateCurriculumHandler(currs, depts), commands.NewUpdateCurriculumHandler(currs, depts),
					commands.NewDeleteCurriculumHandler(currs, storage), queries.NewListCurriculaHandler(currs)),
				handler.NewConfigHandler(handler.AppConfig{MaxUploadMB: 400, MaxCoverMB: 20, MaxImportMB: 7, BuildTimeoutMinutes: 45}),
				handler.NewUserHandler(
					commands.NewCreateUserHandler(users, depts, currs, []string{"chula.ac.th"}),
					commands.NewUpdateUserHandler(users, depts, currs),
					commands.NewDeleteUserHandler(users), queries.NewListUsersHandler(users)),
			},
			Internal: []httpAdapter.Registrar{handler.NewInternalAuthHandler(commands.NewLoginSyncHandler(users))},
		}
		router = httpAdapter.SetupRouter(httpAdapter.RouterConfig{InternalAPIToken: internalToken, ActorLoader: loader},
			handler.NewHealthHandler(pinger), routes)
	})
	AfterEach(func() { ctrl.Finish() })

	do := func(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	asSuper := func() map[string]string {
		loader.EXPECT().LoadActor(gomock.Any(), superID).Return(&domain.Actor{UserID: superID, Role: domain.RoleSuperAdmin}, nil)
		return map[string]string{"X-User-Id": superID.String()}
	}
	asStaff := func() map[string]string {
		loader.EXPECT().LoadActor(gomock.Any(), staffID).Return(&domain.Actor{UserID: staffID, Role: domain.RoleStaff}, nil)
		return map[string]string{"X-User-Id": staffID.String()}
	}

	Describe("authentication middleware", func() {
		It("answers 401 without X-User-Id", func() {
			w := do("GET", "/api/v1/me", "", nil)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"unauthenticated"}`))
		})

		It("answers 401 for a malformed id", func() {
			w := do("GET", "/api/v1/me", "", map[string]string{"X-User-Id": "not-a-uuid"})
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("answers 401 for an unknown user", func() {
			id := uuid.New()
			loader.EXPECT().LoadActor(gomock.Any(), id).Return(nil, domain.ErrNotFound)
			w := do("GET", "/api/v1/me", "", map[string]string{"X-User-Id": id.String()})
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"unauthenticated"}`))
		})

		It("answers a generic 500 when loading the actor fails", func() {
			id := uuid.New()
			loader.EXPECT().LoadActor(gomock.Any(), id).Return(nil, errors.New("password=hunter2 db exploded"))
			w := do("GET", "/api/v1/me", "", map[string]string{"X-User-Id": id.String()})
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"internal error"}`))
		})
	})

	Describe("GET /api/v1/me", func() {
		It("returns the user and capabilities", func() {
			headers := asSuper()
			now := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
			users.EXPECT().GetByID(gomock.Any(), superID).Return(&domain.User{
				ID: superID, Email: "root@chula.ac.th", Role: domain.RoleSuperAdmin, CreatedAt: now, UpdatedAt: now,
			}, nil)
			w := do("GET", "/api/v1/me", "", headers)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(`"capabilities":{"manage_org":true,"manage_users":true,"manage_courses":true,"manage_covers":true}`))
			Expect(w.Body.String()).To(ContainSubstring(`"created_at":"2026-10-09T10:00:00Z"`))
			Expect(w.Body.String()).To(ContainSubstring(`"department":null`))
			Expect(w.Body.String()).To(ContainSubstring(`"admin_scope":null`))
			Expect(w.Body.String()).To(ContainSubstring(`"last_login_at":null`))
		})
	})

	Describe("GET /api/v1/config (H3)", func() {
		It("returns the limits from the environment to any authenticated user", func() {
			w := do("GET", "/api/v1/config", "", asStaff())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON(`{"max_upload_mb":400,"max_cover_mb":20,"max_import_mb":7,"build_timeout_minutes":45}`))
		})

		It("requires authentication", func() {
			Expect(do("GET", "/api/v1/config", "", nil).Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("organisation visibility (H1)", func() {
		deptID, otherDept := uuid.New(), uuid.New()
		asMember := func(role domain.Role) map[string]string {
			loader.EXPECT().LoadActor(gomock.Any(), staffID).Return(&domain.Actor{UserID: staffID, Role: role, DepartmentID: &deptID}, nil)
			return map[string]string{"X-User-Id": staffID.String()}
		}

		It("shows instructor and staff only their own department and its curricula", func() {
			for _, role := range []domain.Role{domain.RoleInstructor, domain.RoleStaff} {
				headers := asMember(role)
				depts.EXPECT().List(gomock.Any()).Return([]domain.Department{{ID: deptID, Code: "OWN"}, {ID: otherDept, Code: "OTHER"}}, nil)
				w := do("GET", "/api/v1/departments", "", headers)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"code":"OWN"`))
				Expect(w.Body.String()).NotTo(ContainSubstring("OTHER"))

				headers = asMember(role)
				currs.EXPECT().List(gomock.Any(), &deptID).Return([]domain.Curriculum{{ID: uuid.New(), Code: "MINE", DepartmentID: deptID}}, nil)
				w = do("GET", "/api/v1/curricula", "", headers)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(ContainSubstring(`"code":"MINE"`))
			}
		})

		It("shows admin everything", func() {
			loader.EXPECT().LoadActor(gomock.Any(), staffID).Return(&domain.Actor{UserID: staffID, Role: domain.RoleAdmin}, nil)
			depts.EXPECT().List(gomock.Any()).Return([]domain.Department{{ID: deptID, Code: "OWN"}, {ID: otherDept, Code: "OTHER"}}, nil)
			w := do("GET", "/api/v1/departments", "", map[string]string{"X-User-Id": staffID.String()})
			Expect(w.Body.String()).To(ContainSubstring("OTHER"))
		})
	})

	Describe("departments", func() {
		It("lists as {items: []} even when empty", func() {
			headers := asSuper()
			depts.EXPECT().List(gomock.Any()).Return(nil, nil)
			w := do("GET", "/api/v1/departments", "", headers)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON(`{"items":[]}`))
		})

		It("creates (201) with an upper-cased code", func() {
			headers := asSuper()
			depts.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			w := do("POST", "/api/v1/departments", `{"code":"math-cs","name":"Math"}`, headers)
			Expect(w.Code).To(Equal(http.StatusCreated))
			Expect(w.Body.String()).To(ContainSubstring(`"code":"MATH-CS"`))
			Expect(w.Body.String()).To(ContainSubstring(`"curriculum_count":0`))
		})

		It("answers 403 forbidden for non super admins", func() {
			headers := asStaff()
			w := do("POST", "/api/v1/departments", `{"code":"x","name":"x"}`, headers)
			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"forbidden"}`))
		})

		It("answers 400 for invalid json and for validation failures", func() {
			headers := asSuper()
			w := do("POST", "/api/v1/departments", `{bad`, headers)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			headers = asSuper()
			w = do("POST", "/api/v1/departments", `{"code":"","name":"x"}`, headers)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("validation failed"))
		})

		It("answers 409 with the contract message when deleting a referenced department", func() {
			headers := asSuper()
			id := uuid.New()
			depts.EXPECT().Delete(gomock.Any(), id).Return(domain.ErrConflict)
			w := do("DELETE", "/api/v1/departments/"+id.String(), "", headers)
			Expect(w.Code).To(Equal(http.StatusConflict))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"department is referenced by curricula or users"}`))
		})

		It("answers 204 on delete and 400 for a bad id", func() {
			headers := asSuper()
			id := uuid.New()
			depts.EXPECT().Delete(gomock.Any(), id).Return(nil)
			Expect(do("DELETE", "/api/v1/departments/"+id.String(), "", headers).Code).To(Equal(http.StatusNoContent))
			headers = asSuper()
			Expect(do("DELETE", "/api/v1/departments/xyz", "", headers).Code).To(Equal(http.StatusBadRequest))
		})

		It("never leaks unexpected errors on 500", func() {
			headers := asSuper()
			depts.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("pq: secret connection string"))
			w := do("POST", "/api/v1/departments", `{"code":"x","name":"x"}`, headers)
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"internal error"}`))
		})
	})

	Describe("curricula", func() {
		It("filters by department_id and rejects a malformed one", func() {
			headers := asSuper()
			id := uuid.New()
			currs.EXPECT().List(gomock.Any(), &id).Return([]domain.Curriculum{{ID: uuid.New(), Code: "C", Name: "C"}}, nil)
			w := do("GET", "/api/v1/curricula?department_id="+id.String(), "", headers)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(`"cover":null`))
			headers = asSuper()
			Expect(do("GET", "/api/v1/curricula?department_id=zzz", "", headers).Code).To(Equal(http.StatusBadRequest))
		})

		It("serialises the cover object when present", func() {
			headers := asSuper()
			updated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			currs.EXPECT().List(gomock.Any(), nil).Return([]domain.Curriculum{{
				ID: uuid.New(), Code: "C", Name: "C", Cover: &domain.Cover{Filename: "cover.pdf", SizeBytes: 245760, UpdatedAt: updated},
			}}, nil)
			w := do("GET", "/api/v1/curricula", "", headers)
			Expect(w.Body.String()).To(ContainSubstring(`"cover":{"filename":"cover.pdf","size_bytes":245760,"updated_at":"2026-01-02T03:04:05Z"}`))
		})
	})

	Describe("users", func() {
		It("marks rows editable per caller", func() {
			headers := asSuper()
			other := domain.User{ID: uuid.New(), Email: "o@chula.ac.th", Role: domain.RoleStaff}
			self := domain.User{ID: superID, Email: "root@chula.ac.th", Role: domain.RoleSuperAdmin}
			users.EXPECT().List(gomock.Any(), gomock.Any()).Return([]domain.User{other, self}, nil)
			w := do("GET", "/api/v1/users?search=o&role=staff", "", headers)
			Expect(w.Code).To(Equal(http.StatusOK))
			body := w.Body.String()
			Expect(strings.Count(body, `"editable":true`)).To(Equal(1))
			Expect(strings.Count(body, `"editable":false`)).To(Equal(1))
		})

		It("answers 403 to staff", func() {
			headers := asStaff()
			Expect(do("GET", "/api/v1/users", "", headers).Code).To(Equal(http.StatusForbidden))
		})

		It("answers 400 for an e-mail outside the allowed domains", func() {
			headers := asSuper()
			w := do("POST", "/api/v1/users", `{"email":"x@gmail.com","role":"staff"}`, headers)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("email domain is not allowed"))
		})

		It("answers 409 when deleting yourself", func() {
			headers := asSuper()
			w := do("DELETE", "/api/v1/users/"+superID.String(), "", headers)
			Expect(w.Code).To(Equal(http.StatusConflict))
		})

		It("distinguishes an absent department_id from null on PATCH", func() {
			headers := asSuper()
			target := &domain.User{ID: uuid.New(), Email: "t@chula.ac.th", Role: domain.RoleStaff, DepartmentID: ptrID(uuid.New())}
			users.EXPECT().GetByID(gomock.Any(), target.ID).Return(target, nil).Times(2)
			users.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
				Expect(u.DepartmentID).To(BeNil())
				return nil
			})
			w := do("PATCH", "/api/v1/users/"+target.ID.String(), `{"department_id":null}`, headers)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("internal login-sync", func() {
		body := `{"email":"Alice@Chula.ac.th","name":"Alice","picture_url":"https://x/a.png"}`

		It("rejects a missing or wrong token", func() {
			Expect(do("POST", "/internal/v1/auth/login-sync", body, nil).Code).To(Equal(http.StatusUnauthorized))
			w := do("POST", "/internal/v1/auth/login-sync", body, map[string]string{"X-Internal-Token": "wrong"})
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("syncs a provisioned user", func() {
			id := uuid.New()
			users.EXPECT().TouchLogin(gomock.Any(), "alice@chula.ac.th", "Alice", "https://x/a.png").
				Return(&domain.User{ID: id, Email: "alice@chula.ac.th", Role: domain.RoleInstructor}, nil)
			w := do("POST", "/internal/v1/auth/login-sync", body, map[string]string{"X-Internal-Token": internalToken})
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON(`{"user_id":"` + id.String() + `","email":"alice@chula.ac.th","role":"instructor"}`))
		})

		It("answers 403 not provisioned", func() {
			users.EXPECT().TouchLogin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, domain.ErrNotFound)
			w := do("POST", "/internal/v1/auth/login-sync", body, map[string]string{"X-Internal-Token": internalToken})
			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"not provisioned"}`))
		})

		It("answers 400 for an empty body", func() {
			w := do("POST", "/internal/v1/auth/login-sync", `{}`, map[string]string{"X-Internal-Token": internalToken})
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("is not reachable with a user header only", func() {
			w := do("POST", "/internal/v1/auth/login-sync", body, map[string]string{"X-User-Id": superID.String()})
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("internal token middleware with an empty configured token", func() {
		It("rejects everything", func() {
			r := httpAdapter.SetupRouter(httpAdapter.RouterConfig{InternalAPIToken: "", ActorLoader: loader},
				handler.NewHealthHandler(pinger),
				httpAdapter.Routes{Internal: []httpAdapter.Registrar{handler.NewInternalAuthHandler(commands.NewLoginSyncHandler(users))}})
			req := httptest.NewRequest("POST", "/internal/v1/auth/login-sync", strings.NewReader(`{"email":"a@chula.ac.th"}`))
			req.Header.Set("X-Internal-Token", "")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("probes and misc", func() {
		It("serves /healthz without authentication", func() {
			Expect(do("GET", "/healthz", "", nil).Code).To(Equal(http.StatusOK))
		})

		It("reports /readyz based on the DB ping and leaks no error text", func() {
			pinger.EXPECT().Ping(gomock.Any()).Return(nil)
			Expect(do("GET", "/readyz", "", nil).Code).To(Equal(http.StatusOK))
			pinger.EXPECT().Ping(gomock.Any()).Return(errors.New("dial tcp 10.0.0.5: refused"))
			w := do("GET", "/readyz", "", nil)
			Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
			Expect(w.Body.String()).NotTo(ContainSubstring("10.0.0.5"))
		})

		It("makes /readyz depend on the object store and leaks no error text", func() {
			storagePinger := mocks.NewMockPinger(ctrl)
			r := httpAdapter.SetupRouter(httpAdapter.RouterConfig{ActorLoader: loader},
				handler.NewHealthHandler(pinger).WithStorage(storagePinger), httpAdapter.Routes{})
			serve := func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				r.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
				return w
			}
			pinger.EXPECT().Ping(gomock.Any()).Return(nil)
			storagePinger.EXPECT().Ping(gomock.Any()).Return(nil)
			w := serve()
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(`"storage":"up"`))

			pinger.EXPECT().Ping(gomock.Any()).Return(nil)
			storagePinger.EXPECT().Ping(gomock.Any()).Return(errors.New("dial tcp 10.0.0.9:9000: refused"))
			w = serve()
			Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
			Expect(w.Body.String()).To(ContainSubstring(`"storage":"down"`))
			Expect(w.Body.String()).NotTo(ContainSubstring("10.0.0.9"))
		})

		It("adds the build queue state to /readyz without affecting readiness", func() {
			at := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
			for _, tc := range []struct {
				snap port.QueueSnapshot
				want string
			}{
				{port.QueueSnapshot{}, `"build_queue":{"status":"unknown"}`},
				{port.QueueSnapshot{Known: true, Consumers: 0, Messages: 3, CheckedAt: at},
					`"build_queue":{"checked_at":"2026-10-10T12:00:00Z","consumers":0,"messages":3}`},
			} {
				r := httpAdapter.SetupRouter(httpAdapter.RouterConfig{ActorLoader: loader},
					handler.NewHealthHandler(pinger).WithQueue(fakeQueue{tc.snap}), httpAdapter.Routes{})
				pinger.EXPECT().Ping(gomock.Any()).Return(nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
				Expect(w.Code).To(Equal(http.StatusOK), "no consumers must not make the service unready")
				Expect(w.Body.String()).To(ContainSubstring(tc.want))
			}
		})

		It("exposes /metrics but not pprof or swagger by default", func() {
			Expect(do("GET", "/metrics", "", nil).Code).To(Equal(http.StatusOK))
			Expect(do("GET", "/debug/pprof/", "", nil).Code).To(Equal(http.StatusNotFound))
			Expect(do("GET", "/swagger/index.html", "", nil).Code).To(Equal(http.StatusNotFound))
		})

		It("mounts pprof and swagger only when enabled", func() {
			r := httpAdapter.SetupRouter(httpAdapter.RouterConfig{EnablePprof: true, EnableSwagger: true, ActorLoader: loader},
				handler.NewHealthHandler(pinger), httpAdapter.Routes{})
			for _, p := range []string{"/debug/pprof/", "/swagger/"} {
				req := httptest.NewRequest("GET", p, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				Expect(w.Code).To(Equal(http.StatusOK), p)
			}
		})

		It("answers unknown routes with a JSON 404", func() {
			w := do("GET", "/nope", "", nil)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"not found"}`))
		})
	})
})

func ptrID(id uuid.UUID) *uuid.UUID { return &id }

type fakeQueue struct{ snap port.QueueSnapshot }

func (f fakeQueue) Snapshot() port.QueueSnapshot { return f.snap }
