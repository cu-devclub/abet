package http_test

import (
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
)

// Dev-only act-as (D28): middleware swap, endpoints and the /me block, with the feature ON and OFF.
var _ = Describe("dev act-as over HTTP", func() {
	var (
		ctrl   *gomock.Controller
		loader *mocks.MockActorLoader
		users  *mocks.MockUserRepository

		superID  = uuid.New()
		adminID  = uuid.New()
		staffID  = uuid.New()
		superAct *domain.Actor
		adminAct *domain.Actor
		staffAct *domain.Actor
	)

	build := func(enabled bool) *gin.Engine {
		api := []httpAdapter.Registrar{handler.NewMeHandler(queries.NewGetMeHandler(users)).
			WithActAs(queries.NewGetActAsStateHandler(users, enabled))}
		if enabled {
			api = append(api, handler.NewActAsHandler(
				queries.NewListActAsUsersHandler(users), commands.NewStartActAsHandler(users), false))
		}
		return httpAdapter.SetupRouter(httpAdapter.RouterConfig{ActorLoader: loader, DevActAsEnabled: enabled},
			handler.NewHealthHandler(mocks.NewMockPinger(ctrl)), httpAdapter.Routes{API: api})
	}
	do := func(r *gin.Engine, method, path, body, userID, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if userID != "" {
			req.Header.Set("X-User-Id", userID)
		}
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "abet_act_as", Value: cookie})
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctrl = gomock.NewController(GinkgoT())
		loader = mocks.NewMockActorLoader(ctrl)
		users = mocks.NewMockUserRepository(ctrl)
		superAct = &domain.Actor{UserID: superID, Email: "root@chula.ac.th", Role: domain.RoleSuperAdmin}
		adminAct = &domain.Actor{UserID: adminID, Email: "adm@chula.ac.th", Role: domain.RoleAdmin}
		staffAct = &domain.Actor{UserID: staffID, Email: "stf@chula.ac.th", Role: domain.RoleStaff}
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("middleware swap", func() {
		meUser := func(id uuid.UUID, role domain.Role, email string) {
			users.EXPECT().GetByID(gomock.Any(), id).Return(&domain.User{ID: id, Role: role, Email: email, Name: "N " + email}, nil)
		}

		It("acts as the cookie user when ON and the real user is super_admin", func() {
			loader.EXPECT().LoadActor(gomock.Any(), superID).Return(superAct, nil)
			loader.EXPECT().LoadActor(gomock.Any(), adminID).Return(adminAct, nil)
			meUser(adminID, domain.RoleAdmin, "adm@chula.ac.th")
			meUser(superID, domain.RoleSuperAdmin, "root@chula.ac.th") // real user for the banner
			w := do(build(true), "GET", "/api/v1/me", "", superID.String(), adminID.String())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(`"email":"adm@chula.ac.th"`))
			Expect(w.Body.String()).To(ContainSubstring(`"act_as":{"available":true,"real_user":{"id":"` + superID.String() + `","email":"root@chula.ac.th"`))
		})

		It("reports available=true and real_user=null for a super_admin who is not acting", func() {
			loader.EXPECT().LoadActor(gomock.Any(), superID).Return(superAct, nil)
			meUser(superID, domain.RoleSuperAdmin, "root@chula.ac.th")
			w := do(build(true), "GET", "/api/v1/me", "", superID.String(), "")
			Expect(w.Body.String()).To(ContainSubstring(`"act_as":{"available":true,"real_user":null}`))
		})

		It("ignores the cookie entirely when OFF (never loads the target)", func() {
			loader.EXPECT().LoadActor(gomock.Any(), superID).Return(superAct, nil)
			meUser(superID, domain.RoleSuperAdmin, "root@chula.ac.th")
			w := do(build(false), "GET", "/api/v1/me", "", superID.String(), adminID.String())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(`"email":"root@chula.ac.th"`))
			Expect(w.Body.String()).To(ContainSubstring(`"act_as":{"available":false,"real_user":null}`))
		})

		It("ignores the cookie of a non-super_admin real user", func() {
			loader.EXPECT().LoadActor(gomock.Any(), staffID).Return(staffAct, nil)
			meUser(staffID, domain.RoleStaff, "stf@chula.ac.th")
			w := do(build(true), "GET", "/api/v1/me", "", staffID.String(), adminID.String())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(ContainSubstring(`"email":"stf@chula.ac.th"`))
			Expect(w.Body.String()).To(ContainSubstring(`"act_as":{"available":false,"real_user":null}`))
		})

		It("ignores an invalid, stale or self cookie without failing the request", func() {
			ghost := uuid.New()
			for _, cookie := range []string{"not-a-uuid", ghost.String(), superID.String()} {
				loader.EXPECT().LoadActor(gomock.Any(), superID).Return(superAct, nil)
				if cookie == ghost.String() {
					loader.EXPECT().LoadActor(gomock.Any(), ghost).Return(nil, domain.ErrNotFound)
				}
				meUser(superID, domain.RoleSuperAdmin, "root@chula.ac.th")
				w := do(build(true), "GET", "/api/v1/me", "", superID.String(), cookie)
				Expect(w.Code).To(Equal(http.StatusOK), cookie)
				Expect(w.Body.String()).To(ContainSubstring(`"email":"root@chula.ac.th"`))
			}
		})
	})

	Describe("endpoints", func() {
		asSuper := func() { loader.EXPECT().LoadActor(gomock.Any(), superID).Return(superAct, nil) }

		It("are 404 when OFF", func() {
			r := build(false)
			for _, tc := range [][2]string{{"GET", "/api/v1/dev/act-as/users"}, {"POST", "/api/v1/dev/act-as"}, {"DELETE", "/api/v1/dev/act-as"}} {
				w := do(r, tc[0], tc[1], `{}`, superID.String(), "") // unmatched route: auth middleware never runs
				Expect(w.Code).To(Equal(http.StatusNotFound), tc[0])
				Expect(w.Body.String()).To(MatchJSON(`{"error":"not found"}`))
			}
		})

		It("are 403 for a non-super_admin real user, even with a super_admin cookie", func() {
			r := build(true)
			for _, tc := range [][2]string{{"GET", "/api/v1/dev/act-as/users"}, {"POST", "/api/v1/dev/act-as"}, {"DELETE", "/api/v1/dev/act-as"}} {
				loader.EXPECT().LoadActor(gomock.Any(), staffID).Return(staffAct, nil)
				w := do(r, tc[0], tc[1], `{"user_id":"`+adminID.String()+`"}`, staffID.String(), superID.String())
				Expect(w.Code).To(Equal(http.StatusForbidden), tc[0])
				Expect(w.Body.String()).To(MatchJSON(`{"error":"forbidden"}`))
			}
		})

		It("lists users except the real one, sorted by role then email, with the filters passed through", func() {
			asSuper()
			dept := uuid.New()
			users.EXPECT().List(gomock.Any(), port.UserFilter{Search: "chula"}).Return([]domain.User{
				{ID: staffID, Email: "b@chula.ac.th", Role: domain.RoleStaff},
				{ID: superID, Email: "root@chula.ac.th", Role: domain.RoleSuperAdmin},
				{ID: uuid.New(), Email: "z@chula.ac.th", Role: domain.RoleAdmin},
				{ID: uuid.New(), Email: "a@chula.ac.th", Role: domain.RoleAdmin, DepartmentID: &dept},
			}, nil)
			w := do(build(true), "GET", "/api/v1/dev/act-as/users?q=chula", "", superID.String(), "")
			Expect(w.Code).To(Equal(http.StatusOK))
			body := w.Body.String()
			Expect(body).NotTo(ContainSubstring("root@chula.ac.th"))
			iA, iZ, iB := strings.Index(body, "a@chula"), strings.Index(body, "z@chula"), strings.Index(body, "b@chula")
			Expect(iA).To(BeNumerically(">=", 0))
			Expect(iA).To(BeNumerically("<", iZ))
			Expect(iZ).To(BeNumerically("<", iB))
			Expect(body).To(ContainSubstring(`"department_id":"` + dept.String() + `"`))
		})

		It("filters the list by role", func() {
			asSuper()
			role := domain.RoleInstructor
			users.EXPECT().List(gomock.Any(), port.UserFilter{Role: &role}).Return(nil, nil)
			w := do(build(true), "GET", "/api/v1/dev/act-as/users?role=instructor", "", superID.String(), "")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON(`{"users":[]}`))
		})

		It("POST sets the HttpOnly Lax session cookie on /auth/api", func() {
			asSuper()
			users.EXPECT().GetByID(gomock.Any(), adminID).Return(&domain.User{ID: adminID}, nil)
			w := do(build(true), "POST", "/api/v1/dev/act-as", `{"user_id":"`+adminID.String()+`"}`, superID.String(), "")
			Expect(w.Code).To(Equal(http.StatusNoContent))
			c := w.Header().Get("Set-Cookie")
			Expect(c).To(ContainSubstring("abet_act_as=" + adminID.String()))
			Expect(c).To(ContainSubstring("Path=/auth/api"))
			Expect(c).To(ContainSubstring("HttpOnly"))
			Expect(c).To(ContainSubstring("SameSite=Lax"))
			Expect(c).NotTo(ContainSubstring("Max-Age"))
			Expect(c).NotTo(ContainSubstring("Secure"))
		})

		It("POST rejects a bad body, an unknown user and yourself", func() {
			r := build(true)
			asSuper()
			Expect(do(r, "POST", "/api/v1/dev/act-as", `{"user_id":"nope"}`, superID.String(), "").Code).To(Equal(http.StatusBadRequest))
			asSuper()
			Expect(do(r, "POST", "/api/v1/dev/act-as", `{}`, superID.String(), "").Code).To(Equal(http.StatusBadRequest))
			asSuper()
			ghost := uuid.New()
			users.EXPECT().GetByID(gomock.Any(), ghost).Return(nil, domain.ErrNotFound)
			w := do(r, "POST", "/api/v1/dev/act-as", `{"user_id":"`+ghost.String()+`"}`, superID.String(), "")
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(w.Body.String()).To(MatchJSON(`{"error":"not found"}`))
			asSuper()
			Expect(do(r, "POST", "/api/v1/dev/act-as", `{"user_id":"`+superID.String()+`"}`, superID.String(), "").Code).To(Equal(http.StatusBadRequest))
		})

		It("DELETE clears the cookie and works while acting (real actor decides)", func() {
			loader.EXPECT().LoadActor(gomock.Any(), superID).Return(superAct, nil)
			loader.EXPECT().LoadActor(gomock.Any(), staffID).Return(staffAct, nil)
			w := do(build(true), "DELETE", "/api/v1/dev/act-as", "", superID.String(), staffID.String())
			Expect(w.Code).To(Equal(http.StatusNoContent))
			c := w.Header().Get("Set-Cookie")
			Expect(c).To(ContainSubstring("abet_act_as="))
			Expect(c).To(ContainSubstring("Max-Age=0"))
			Expect(c).To(ContainSubstring("Path=/auth/api"))
		})
	})
})
