package caddylura

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestParseCaddyFile(t *testing.T) {
	input := `
lura {
	timeout 10s
	cache_ttl 360s
	
	debug_endpoint /api/__debug
	echo_endpoint

    endpoint /users/{user} {
        method GET

        backend {
            to http://mock:8081
			url_pattern /registered/{user}
            allow id name email
            mapping {
                email>personal_email
            }
        }

		backend {
			to http://mock:8081
			url_pattern /users/{user}/permissions
			group permissions
		}
		
		concurrent_calls 2
		timeout 1000s
		cache_ttl 3600s
    }

	endpoint {
		url_pattern /foo/bar
		method POST
		backend http://mock:8082 http://mock:8083 {
			url_pattern /baz
			method PUT
		}
	}
}
`
	d := caddyfile.NewTestDispenser(input)

	l := new(Lura)
	err := l.UnmarshalCaddyfile(d)
	if !assert.NoError(t, err) {
		t.Fatal()
	}

	expected := &Lura{
		Timeout:  caddy.Duration(10 * time.Second),
		CacheTTL: caddy.Duration(360 * time.Second),
		DebugEndpoint: HelperEndpoint{
			URLPattern: "/api/__debug",
			Enabled:    true,
		},
		EchoEndpoint: HelperEndpoint{
			URLPattern: "",
			Enabled:    true,
		},
		Endpoints: []Endpoint{
			{
				Method:     "GET",
				URLPattern: "/users/{user}",
				Backends: []Backend{
					{
						Host:       []string{"http://mock:8081"},
						URLPattern: "/registered/{user}",
						AllowList:  []string{"id", "name", "email"},
						Mapping: map[string]string{
							"email": "personal_email",
						},
					},
					{
						Host:       []string{"http://mock:8081"},
						URLPattern: "/users/{user}/permissions",
						Group:      "permissions",
					},
				},
				ConcurrentCalls: 2,
				Timeout:         caddy.Duration(1000 * time.Second),
				CacheTTL:        caddy.Duration(3600 * time.Second),
			},
			{
				Method:     "POST",
				URLPattern: "/foo/bar",
				Backends: []Backend{
					{
						Host: []string{
							"http://mock:8082",
							"http://mock:8083",
						},
						URLPattern: "/baz",
						Method:     "PUT",
					},
				},
			},
		},
	}

	assert.Equal(t, expected, l)
}
