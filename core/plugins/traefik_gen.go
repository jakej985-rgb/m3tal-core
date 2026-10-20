package plugin

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// TraefikDynamicConfig represents the file provider format for Traefik.
type TraefikDynamicConfig struct {
	HTTP TraefikHTTPConfig `yaml:"http"`
}

type TraefikHTTPConfig struct {
	Routers     map[string]*TraefikRouter     `yaml:"routers,omitempty"`
	Services    map[string]*TraefikService    `yaml:"services,omitempty"`
	Middlewares map[string]*TraefikMiddleware `yaml:"middlewares,omitempty"`
}

type TraefikRouter struct {
	Rule        string      `yaml:"rule"`
	Service     string      `yaml:"service"`
	EntryPoints []string    `yaml:"entryPoints,omitempty"`
	Middlewares []string    `yaml:"middlewares,omitempty"`
	TLS         *TraefikTLS `yaml:"tls,omitempty"`
}

type TraefikTLS struct {
	CertResolver string `yaml:"certResolver,omitempty"`
}

type TraefikService struct {
	LoadBalancer *TraefikLoadBalancer `yaml:"loadBalancer"`
}

type TraefikLoadBalancer struct {
	Servers []TraefikServer `yaml:"servers"`
}

type TraefikServer struct {
	URL string `yaml:"url"`
}

// TraefikMiddleware represents various Traefik middleware types in YAML.
type TraefikMiddleware struct {
	BasicAuth   *TraefikBasicAuth      `yaml:"basicAuth,omitempty"`
	RateLimit   *TraefikRateLimit      `yaml:"rateLimit,omitempty"`
	StripPrefix *TraefikStripPrefix    `yaml:"stripPrefix,omitempty"`
	Headers     map[string]interface{} `yaml:"headers,omitempty"`
}

type TraefikBasicAuth struct {
	Users     []string `yaml:"users,omitempty"`
	UsersFile string   `yaml:"usersFile,omitempty"`
}

type TraefikRateLimit struct {
	Average int64 `yaml:"average,omitempty"`
	Burst   int64 `yaml:"burst,omitempty"`
}

type TraefikStripPrefix struct {
	Prefixes []string `yaml:"prefixes,omitempty"`
}

// GenerateTraefikConfig compiles all enabled Route and Middleware plugins in the registry.
func (r *Registry) GenerateTraefikConfig() ([]byte, error) {
	cfg := TraefikDynamicConfig{
		HTTP: TraefikHTTPConfig{
			Routers:     make(map[string]*TraefikRouter),
			Services:    make(map[string]*TraefikService),
			Middlewares: make(map[string]*TraefikMiddleware),
		},
	}

	// 1. Process Route Plugins
	for _, rp := range r.Routes {
		if !rp.Enabled {
			continue
		}
		name := rp.Metadata.Name

		entrypoints := []string{"web"}
		if rp.SSL {
			entrypoints = []string{"web", "websecure"}
		}
		if rp.Entrypoints != "" {
			entrypoints = strings.Split(rp.Entrypoints, ",")
			for i := range entrypoints {
				entrypoints[i] = strings.TrimSpace(entrypoints[i])
			}
		}

		domainRule := fmt.Sprintf("Host(`%s`)", rp.Domain)
		if strings.HasPrefix(rp.Domain, "Host(`") {
			domainRule = rp.Domain
		}

		var routerTLS *TraefikTLS
		if rp.SSL {
			routerTLS = &TraefikTLS{CertResolver: "letsencrypt"}
		}

		cfg.HTTP.Routers[name] = &TraefikRouter{
			Rule:        domainRule,
			Service:     name,
			EntryPoints: entrypoints,
			Middlewares: rp.Middlewares,
			TLS:         routerTLS,
		}

		// Use the plugin's service name or metadata name for local container name resolution
		targetService := rp.Service
		if targetService == "" {
			targetService = name
		}

		cfg.HTTP.Services[name] = &TraefikService{
			LoadBalancer: &TraefikLoadBalancer{
				Servers: []TraefikServer{
					{URL: fmt.Sprintf("http://%s:%d", targetService, rp.Port)},
				},
			},
		}
	}

	// 2. Process Middleware Plugins
	for _, mp := range r.Middlewares {
		if !mp.Enabled {
			continue
		}
		name := mp.Name
		if name == "" {
			name = mp.Metadata.Name
		}

		tm := &TraefikMiddleware{}
		switch strings.ToLower(mp.Type) {
		case "basicauth":
			auth := &TraefikBasicAuth{}
			if u, ok := mp.Config["users"]; ok {
				users := strings.Split(u, ",")
				for i := range users {
					users[i] = strings.TrimSpace(users[i])
				}
				auth.Users = users
			}
			if uf, ok := mp.Config["usersFile"]; ok {
				auth.UsersFile = uf
			} else if uf, ok := mp.Config["usersfile"]; ok {
				auth.UsersFile = uf
			}
			tm.BasicAuth = auth

		case "ratelimit":
			rl := &TraefikRateLimit{}
			if avg, ok := mp.Config["average"]; ok {
				if val, err := strconv.ParseInt(avg, 10, 64); err == nil {
					rl.Average = val
				}
			}
			if burst, ok := mp.Config["burst"]; ok {
				if val, err := strconv.ParseInt(burst, 10, 64); err == nil {
					rl.Burst = val
				}
			}
			tm.RateLimit = rl

		case "stripprefix":
			sp := &TraefikStripPrefix{}
			if pref, ok := mp.Config["prefixes"]; ok {
				prefixes := strings.Split(pref, ",")
				for i := range prefixes {
					prefixes[i] = strings.TrimSpace(prefixes[i])
				}
				sp.Prefixes = prefixes
			}
			tm.StripPrefix = sp

		case "headers":
			headers := make(map[string]interface{})
			for k, v := range mp.Config {
				if strings.ToLower(v) == "true" {
					headers[k] = true
				} else if strings.ToLower(v) == "false" {
					headers[k] = false
				} else if val, err := strconv.ParseInt(v, 10, 64); err == nil {
					headers[k] = val
				} else {
					headers[k] = v
				}
			}
			tm.Headers = headers
		}

		cfg.HTTP.Middlewares[name] = tm
	}

	if len(cfg.HTTP.Routers) == 0 && len(cfg.HTTP.Services) == 0 && len(cfg.HTTP.Middlewares) == 0 {
		return []byte("# No active plugin routes or middlewares\n"), nil
	}

	return yaml.Marshal(cfg)
}
