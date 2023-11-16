package main

import (
	"log/slog"
	"net/url"
	"slices"
	"strings"

	gatev1 "sigs.k8s.io/gateway-api/apis/v1"
)

type Route struct {
	hostnames []string
	url       *url.URL
	matchPath string
	matchType string // Exact or PathPrefix
}

type Router struct {
	table []Route
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
		var err error
		backend, err := url.Parse("http://" + string(rule.BackendRefs[0].Name))
		if err != nil {
			r.log.Error(err.Error())
		}
		for _, m := range rule.Matches {
			rt := Route{url: backend, hostnames: hostnames}
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
			return route.url
		}
		if route.matchType == "PathPrefix" && strings.HasPrefix(path, route.matchPath) {
			return route.url
		}
	}
	return nil
}
