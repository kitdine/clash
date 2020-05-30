package route

import (
	"context"
	"net/http"

	"github.com/Dreamacro/clash/adapters/provider"
	"github.com/Dreamacro/clash/tunnel"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

func proxyProviderRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getProxyProviders)

	r.Route("/{name}", func(r chi.Router) {
		r.Use(parseProviderName, findProxyProviderByName)
		r.Get("/", getProxyProvider)
		r.Put("/", updateProxyProvider)
		r.Get("/healthcheck", healthCheckProvider)
	})
	return r
}

func getProxyProviders(w http.ResponseWriter, r *http.Request) {
	providers := tunnel.ProxyProviders()
	render.JSON(w, r, render.M{
		"providers": providers,
	})
}

func getProxyProvider(w http.ResponseWriter, r *http.Request) {
	provider := r.Context().Value(CtxKeyProvider).(provider.ProxyProvider)
	render.JSON(w, r, provider)
}

func updateProxyProvider(w http.ResponseWriter, r *http.Request) {
	provider := r.Context().Value(CtxKeyProvider).(provider.ProxyProvider)
	if err := provider.Update(); err != nil {
		render.Status(r, http.StatusServiceUnavailable)
		render.JSON(w, r, newError(err.Error()))
		return
	}
	render.NoContent(w, r)
}

func healthCheckProvider(w http.ResponseWriter, r *http.Request) {
	provider := r.Context().Value(CtxKeyProvider).(provider.ProxyProvider)
	provider.HealthCheck()
	render.NoContent(w, r)
}

func findProxyProviderByName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Context().Value(CtxKeyProviderName).(string)
		providers := tunnel.ProxyProviders()
		provider, exist := providers[name]
		if !exist {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), CtxKeyProvider, provider)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func findRuleProviderByName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Context().Value(CtxKeyProviderName).(string)
		providers := tunnel.RuleProviders()
		provider, exist := providers[name]
		if !exist {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), CtxKeyProvider, provider)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ruleProviderRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/", getRuleProviders)

	r.Route("/{name}", func(r chi.Router) {
		r.Use(parseProviderName, findRuleProviderByName)
		r.Get("/", getRuleProvider)
		r.Put("/", updateRuleProvider)
	})
	return r
}

func getRuleProviders(w http.ResponseWriter, r *http.Request) {
	providers := tunnel.RuleProviders()
	render.JSON(w, r, render.M{
		"providers": providers,
	})
}

func getRuleProvider(w http.ResponseWriter, r *http.Request) {
	provider := r.Context().Value(CtxKeyProvider).(provider.RuleProvider)
	render.JSON(w, r, provider)
}

func updateRuleProvider(w http.ResponseWriter, r *http.Request) {
	provider := r.Context().Value(CtxKeyProvider).(provider.RuleProvider)
	if err := provider.Update(); err != nil {
		render.Status(r, http.StatusServiceUnavailable)
		render.JSON(w, r, newError(err.Error()))
		return
	}
	render.NoContent(w, r)
}

func parseProviderName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := getEscapeParam(r, "name")
		ctx := context.WithValue(r.Context(), CtxKeyProviderName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}