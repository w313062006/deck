package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/kong/deck/konnect"
	"github.com/kong/go-kong/kong"
	"github.com/kong/go-kong/kong/custom"
	"github.com/pkg/errors"
)

// KongRawState contains all of Kong Data
type KongRawState struct {
	Services []*kong.Service
	Routes   []*kong.Route

	Plugins []*kong.Plugin

	Upstreams []*kong.Upstream
	Targets   []*kong.Target

	Certificates   []*kong.Certificate
	SNIs           []*kong.SNI
	CACertificates []*kong.CACertificate

	Consumers      []*kong.Consumer
	CustomEntities []*custom.Entity

	KeyAuths    []*kong.KeyAuth
	HMACAuths   []*kong.HMACAuth
	JWTAuths    []*kong.JWTAuth
	BasicAuths  []*kong.BasicAuth
	ACLGroups   []*kong.ACLGroup
	Oauth2Creds []*kong.Oauth2Credential
	MTLSAuths   []*kong.MTLSAuth

	RBACRoles               []*kong.RBACRole
	RBACEndpointPermissions []*kong.RBACEndpointPermission
}

// KonnectRawState contains all of Konnect resources.
type KonnectRawState struct {
	ServicePackages []*konnect.ServicePackage
}

// ErrArray holds an array of errors.
type ErrArray struct {
	Errors []error
}

// Error returns a pretty string of errors present.
func (e ErrArray) Error() string {
	if len(e.Errors) == 0 {
		return "nil"
	}
	var res string

	res = strconv.Itoa(len(e.Errors)) + " errors occurred:\n"
	for _, err := range e.Errors {
		res += fmt.Sprintf("\t%v\n", err)
	}
	return res
}

// KongClientConfig holds config details to use to talk to a Kong server.
type KongClientConfig struct {
	Address   string
	Workspace string

	TLSServerName string

	TLSCACert string

	TLSSkipVerify bool
	Debug         bool

	SkipWorkspaceCrud bool

	Headers []string

	HTTPClient *http.Client
}

type KonnectConfig struct {
	Email    string
	Password string
	Debug    bool
}

// ForWorkspace returns a copy of KongClientConfig that produces a KongClient for the workspace specified by argument.
func (kc *KongClientConfig) ForWorkspace(name string) KongClientConfig {
	result := *kc
	result.Workspace = name
	return result
}

// GetKongClient returns a Kong client
func GetKongClient(opt KongClientConfig) (*kong.Client, error) {

	var tlsConfig tls.Config
	if opt.TLSSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}
	if opt.TLSServerName != "" {
		tlsConfig.ServerName = opt.TLSServerName
	}

	if opt.TLSCACert != "" {
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM([]byte(opt.TLSCACert))
		if !ok {
			return nil, errors.New("failed to load TLSCACert")
		}
		tlsConfig.RootCAs = certPool
	}

	c := opt.HTTPClient
	if c == nil {
		c = &http.Client{}
	}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	defaultTransport.TLSClientConfig = &tlsConfig
	c.Transport = defaultTransport
	if len(opt.Headers) > 0 {

		headers := http.Header{}
		for _, keyValue := range opt.Headers {
			split := strings.SplitN(keyValue, ":", 2)
			if len(split) >= 2 {
				headers[split[0]] = []string{split[1]}
			}
		}
		*c = kong.HTTPClientWithHeaders(c, headers)
	}
	address := CleanAddress(opt.Address)

	url, err := url.ParseRequestURI(address)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse kong address")
	}
	if opt.Workspace != "" {
		url.Path = path.Join(url.Path, opt.Workspace)
	}

	kongClient, err := kong.NewClient(kong.String(url.String()), c)
	if err != nil {
		return nil, errors.Wrap(err, "creating client for Kong's Admin API")
	}
	if opt.Debug {
		kongClient.SetDebugMode(true)
		kongClient.SetLogger(os.Stderr)
	}
	return kongClient, nil
}

func GetKonnectClient(httpClient *http.Client, debug bool) (*konnect.Client,
	error) {
	client, err := konnect.NewClient(httpClient)
	if err != nil {
		return nil, err
	}
	if debug {
		client.SetDebugMode(true)
		client.SetLogger(os.Stderr)
	}
	return client, nil
}

// CleanAddress removes trailling / from a URL.
func CleanAddress(address string) string {
	re := regexp.MustCompile("[/]+$")
	return re.ReplaceAllString(address, "")
}
