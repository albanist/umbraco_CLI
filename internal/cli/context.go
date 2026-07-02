package cli

import (
	"crypto/tls"
	"net/http"
	"time"

	"umbraco-cli/internal/api"
	"umbraco-cli/internal/auth"
	"umbraco-cli/internal/config"
)

type Runtime struct {
	Config     config.Config
	Client     *api.Client
	HTTPClient *http.Client
}

// NewRuntime resolves config and wires the API client. A config resolution
// failure does not fail runtime construction — informational commands
// (--help, --version, schema, generate-skills) must keep working on a
// broken setup. The error is carried inside the client instead, so any
// command that actually reaches for the API reports the real cause.
func NewRuntime() *Runtime {
	// Clone the default transport so proxy env, dial timeouts, and HTTP/2
	// support are inherited; only the TLS floor is tightened.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	httpClient := &http.Client{Timeout: 60 * time.Second, Transport: transport}

	cfg, err := config.Load()
	if err != nil {
		return &Runtime{HTTPClient: httpClient, Client: api.NewUnavailableClient(err)}
	}

	tokenProvider := auth.New(cfg, httpClient)
	client := api.NewClient(cfg, httpClient, tokenProvider)
	return &Runtime{Config: cfg, Client: client, HTTPClient: httpClient}
}

func (r *Runtime) Reload(opts config.LoadOptions) error {
	cfg, err := config.LoadWithOptions(opts)
	if err != nil {
		if r.Client == nil {
			r.Client = api.NewUnavailableClient(err)
		} else {
			r.Client.ReplaceWith(api.NewUnavailableClient(err))
		}
		return err
	}

	tokenProvider := auth.New(cfg, r.HTTPClient)
	client := api.NewClient(cfg, r.HTTPClient, tokenProvider)
	if r.Client == nil {
		r.Client = client
	} else {
		r.Client.ReplaceWith(client)
	}
	r.Config = cfg
	return nil
}
