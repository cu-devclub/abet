package mainclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/auth-service/internal/adapter/outbound/mainclient"
	"abet/auth-service/internal/core/domain"
)

var _ = Describe("HTTPClient.LoginSync", func() {
	var (
		server *httptest.Server
		status int
		body   string
		gotReq map[string]string
		gotTok string
	)

	BeforeEach(func() {
		status, body, gotReq, gotTok = http.StatusOK, "", nil, ""
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodPost))
			Expect(r.URL.Path).To(Equal("/internal/v1/auth/login-sync"))
			gotTok = r.Header.Get("X-Internal-Token")
			_ = json.NewDecoder(r.Body).Decode(&gotReq)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
	})

	AfterEach(func() { server.Close() })

	client := func() *mainclient.HTTPClient {
		return mainclient.NewHTTPClient(server.URL, "secret-token", 2*time.Second)
	}

	It("sends the internal token and body and parses the result", func() {
		uid := uuid.New()
		body = `{"user_id":"` + uid.String() + `","email":"alice@chula.ac.th","role":"instructor"}`

		res, err := client().LoginSync(context.Background(), "alice@chula.ac.th", "Alice", "http://pic")
		Expect(err).NotTo(HaveOccurred())
		Expect(res.UserID).To(Equal(uid))
		Expect(res.Role).To(Equal("instructor"))
		Expect(gotTok).To(Equal("secret-token"))
		Expect(gotReq).To(Equal(map[string]string{"email": "alice@chula.ac.th", "name": "Alice", "picture_url": "http://pic"}))
	})

	It("maps 403 to ErrForbidden", func() {
		status, body = http.StatusForbidden, `{"error":"not provisioned"}`

		_, err := client().LoginSync(context.Background(), "x@chula.ac.th", "X", "")
		Expect(err).To(MatchError(domain.ErrForbidden))
	})

	It("maps 401 and 500 to ErrServiceUnavailable", func() {
		for _, s := range []int{http.StatusUnauthorized, http.StatusInternalServerError} {
			status = s
			_, err := client().LoginSync(context.Background(), "x@chula.ac.th", "X", "")
			Expect(err).To(MatchError(domain.ErrServiceUnavailable))
		}
	})

	It("maps a network failure to ErrServiceUnavailable", func() {
		c := mainclient.NewHTTPClient("http://127.0.0.1:1", "t", time.Second)
		_, err := c.LoginSync(context.Background(), "x@chula.ac.th", "X", "")
		Expect(err).To(MatchError(domain.ErrServiceUnavailable))
	})
})
