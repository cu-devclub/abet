// Package middleware holds the cross-cutting Gin middlewares: actor loading, internal token, logging, metrics.
package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

const (
	// HeaderUserID is set by the gateway forward-auth (APISIX) from the verified session.
	HeaderUserID = "X-User-Id"

	actorKey     = "abet.actor"
	realActorKey = "abet.real_actor"

	// ActAsCookieName holds the UUID of the user a super_admin is acting as (dev only, D28).
	// The path is the public gateway path: the browser reaches main-service as /auth/api/...
	ActAsCookieName = "abet_act_as"
	ActAsCookiePath = "/auth/api"
)

// Authenticate loads the caller from X-User-Id on every request (role and scope are always read from the DB,
// so changes take effect immediately). Missing, malformed or unknown ids answer 401 {"error":"unauthenticated"}.
//
// When actAsEnabled (dev only, D28) and the real actor is a super_admin carrying a valid abet_act_as cookie of an
// existing other user, the request actor becomes that user and the real actor is kept for RealActorFrom.
// An invalid or stale cookie is ignored. With actAsEnabled=false the cookie is never read.
func Authenticate(loader port.ActorLoader, actAsEnabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.GetHeader(HeaderUserID))
		if err != nil {
			unauthenticated(c)
			return
		}
		actor, err := loader.LoadActor(c.Request.Context(), id)
		if err != nil {
			// Unknown user and DB failure both end up here; only the first one is the caller's fault.
			if errors.Is(err, domain.ErrNotFound) {
				unauthenticated(c)
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Set(realActorKey, actor)
		if actAsEnabled {
			if target := actAsTarget(c, loader, actor); target != nil {
				slog.DebugContext(c.Request.Context(), "acting as another user",
					"acting_as", target.UserID.String(), "real_user", actor.UserID.String())
				actor = target
			}
		}
		c.Set(actorKey, actor)
		c.Next()
	}
}

// actAsTarget returns the actor to impersonate, or nil when the cookie is absent, invalid, stale or not allowed.
func actAsTarget(c *gin.Context, loader port.ActorLoader, real *domain.Actor) *domain.Actor {
	if real.Role != domain.RoleSuperAdmin {
		return nil
	}
	raw, err := c.Cookie(ActAsCookieName)
	if err != nil || raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == real.UserID {
		return nil
	}
	target, err := loader.LoadActor(c.Request.Context(), id)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			slog.WarnContext(c.Request.Context(), "act-as target load failed, using real actor", "error", err)
		}
		return nil
	}
	return target
}

// ActorFrom returns the authenticated actor stored by Authenticate (nil outside authenticated routes).
// While acting (dev act-as) this is the impersonated user.
func ActorFrom(c *gin.Context) *domain.Actor {
	v, ok := c.Get(actorKey)
	if !ok {
		return nil
	}
	a, _ := v.(*domain.Actor)
	return a
}

// RealActorFrom returns the really authenticated actor: the super_admin behind an act-as session,
// otherwise the same actor as ActorFrom.
func RealActorFrom(c *gin.Context) *domain.Actor {
	if v, ok := c.Get(realActorKey); ok {
		if a, _ := v.(*domain.Actor); a != nil {
			return a
		}
	}
	return ActorFrom(c)
}

// SetActor stores an actor in the context (used by tests of handlers).
func SetActor(c *gin.Context, a *domain.Actor) { c.Set(actorKey, a) }

func unauthenticated(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
}
