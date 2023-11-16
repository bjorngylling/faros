package main

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRoute_wrong_host(t *testing.T) {
	u, _ := url.Parse("a")
	router := Router{
		table: []*Route{
			{
				backends: []Backend{{url: u}},
				hostnames: []string{"h"},
				matchType: "Exact",
				matchPath: "/",
			},
		},
	}

	got := router.Route("x", "/")
	if got != nil {
		t.Errorf("Route() mismatch, want nil but got %q", got)
	}
}


func TestRouteExact_match(t *testing.T) {
	u, _ := url.Parse("a")
	router := Router{
		table: []*Route{
			{
				backends: []Backend{{url: u}},
				hostnames: []string{"h"},
				matchType: "Exact",
				matchPath: "/",
			},
		},
	}

	got := router.Route("h", "/")
	want, _ := url.Parse("a")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Route() mismatch (-want +got):\n%s", diff)
	}
}

func TestRouteExact_no_match(t *testing.T) {
	u, _ := url.Parse("a")
	router := Router{
		table: []*Route{
			{
				backends: []Backend{{url: u}},
				hostnames: []string{"h"},
				matchType: "Exact",
				matchPath: "/",
			},
		},
	}

	got := router.Route("h", "/foo")
	if got != nil {
		t.Errorf("Route() mismatch, want nil but got %q", got)
	}
}
