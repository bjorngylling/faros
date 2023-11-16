package main

import (
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"

	gatev1 "sigs.k8s.io/gateway-api/apis/v1"
)

type Backend struct {
	url *url.URL
}

type Route struct {
	hostnames []string
	backends  []Backend
	cur       uint32 // The most recently used backend
	matchPath string
	matchType string // Exact or PathPrefix
}

func (r *Route) NextBackend() Backend {
	return r.backends[int(atomic.AddUint32(&r.cur, 1)%uint32(len(r.backends)))]
}

type Router struct {
	table []*Route
	log   slog.Logger
}

func (r *Router) Add(route *gatev1.HTTPRoute) {
	var hostnames []string
	for _, h := range route.Spec.Hostnames {
		hostnames = append(hostnames, string(h))
	}
	for _, rule := range route.Spec.Rules {
		if len(rule.BackendRefs) <= 0 {
			continue // Rules without a backendref are ignored
		}
		var backends []Backend
		for _, backend := range rule.BackendRefs {
			b, err := url.Parse("http://" + string(backend.Name))
			if err != nil {
				r.log.Error(err.Error())
			}
			backends = append(backends, Backend{url: b})
		}
		for _, m := range rule.Matches {
			rt := &Route{backends: backends, hostnames: hostnames}
			rt.matchType = string(*m.Path.Type)
			rt.matchPath = *m.Path.Value
			r.table = append(r.table, rt)
		}
	}
}

func (r *Router) Route(host string, path string) *url.URL {
	for _, route := range r.table {
		if !slices.Contains(route.hostnames, host) {
			continue
		}
		if route.matchType == "Exact" && route.matchPath == path {
			return route.NextBackend().url
		}
		if route.matchType == "PathPrefix" && strings.HasPrefix(path, route.matchPath) {
			return route.NextBackend().url
		}
	}
	return nil
}
