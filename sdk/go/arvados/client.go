// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

// A Client is an HTTP client with an API endpoint and a set of
// Arvados credentials.
//
// It offers methods for accessing individual Arvados APIs, and
// methods that implement common patterns like fetching multiple pages
// of results using List APIs.
type Client struct {
	// HTTP client used to make requests. If nil,
	// DefaultSecureClient or InsecureHTTPClient will be used.
	Client *http.Client `json:"-"`

	// Protocol scheme: "http", "https", or "" (https)
	Scheme string

	// Hostname (or host:port) of Arvados API server.
	APIHost string

	// User authentication token.
	AuthToken string

	// Accept unverified certificates. This works only if the
	// Client field is nil: otherwise, it has no effect.
	Insecure bool

	// Override keep service discovery with a list of base
	// URIs. (Currently there are no Client methods for
	// discovering keep services so this is just a convenience for
	// callers who use a Client to initialize an
	// arvadosclient.ArvadosClient.)
	KeepServiceURIs []string `json:",omitempty"`

	dd *DiscoveryDocument

	ctx context.Context
}

// The default http.Client used by a Client with Insecure==true and
// Client==nil.
var InsecureHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true}},
	Timeout: 5 * time.Minute}

// The default http.Client used by a Client otherwise.
var DefaultSecureClient = &http.Client{
	Timeout: 5 * time.Minute}

// NewClientFromConfig creates a new Client that uses the endpoints in
// the given cluster.
//
// AuthToken is left empty for the caller to populate.
func NewClientFromConfig(cluster *Cluster) (*Client, error) {
	ctrlURL := cluster.Services.Controller.ExternalURL
	if ctrlURL.Host == "" {
		return nil, fmt.Errorf("no host in config Services.Controller.ExternalURL: %v", ctrlURL)
	}
	return &Client{
		Scheme:   ctrlURL.Scheme,
		APIHost:  ctrlURL.Host,
		Insecure: cluster.TLS.Insecure,
	}, nil
}

// NewClientFromEnv creates a new Client that uses the default HTTP
// client with the API endpoint and credentials given by the
// ARVADOS_API_* environment variables.
func NewClientFromEnv() *Client {
	var svcs []string
	for _, s := range strings.Split(os.Getenv("ARVADOS_KEEP_SERVICES"), " ") {
		if s == "" {
			continue
		} else if u, err := url.Parse(s); err != nil {
			log.Printf("ARVADOS_KEEP_SERVICES: %q: %s", s, err)
		} else if !u.IsAbs() {
			log.Printf("ARVADOS_KEEP_SERVICES: %q: not an absolute URI", s)
		} else {
			svcs = append(svcs, s)
		}
	}
	var insecure bool
	if s := strings.ToLower(os.Getenv("ARVADOS_API_HOST_INSECURE")); s == "1" || s == "yes" || s == "true" {
		insecure = true
	}
	return &Client{
		Scheme:          "https",
		APIHost:         os.Getenv("ARVADOS_API_HOST"),
		AuthToken:       os.Getenv("ARVADOS_API_TOKEN"),
		Insecure:        insecure,
		KeepServiceURIs: svcs,
	}
}

var reqIDGen = httpserver.IDGenerator{Prefix: "req-"}

// Do adds Authorization and X-Request-Id headers and then calls
// (*http.Client)Do().
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if auth, _ := req.Context().Value(contextKeyAuthorization{}).(string); auth != "" {
		req.Header.Add("Authorization", auth)
	} else if c.AuthToken != "" {
		req.Header.Add("Authorization", "OAuth2 "+c.AuthToken)
	}

	if req.Header.Get("X-Request-Id") == "" {
		reqid, _ := req.Context().Value(contextKeyRequestID{}).(string)
		if reqid == "" {
			reqid, _ = c.context().Value(contextKeyRequestID{}).(string)
		}
		if reqid == "" {
			reqid = reqIDGen.Next()
		}
		if req.Header == nil {
			req.Header = http.Header{"X-Request-Id": {reqid}}
		} else {
			req.Header.Set("X-Request-Id", reqid)
		}
	}
	return c.httpClient().Do(req)
}

// DoAndDecode performs req and unmarshals the response (which must be
// JSON) into dst. Use this instead of RequestAndDecode if you need
// more control of the http.Request object.
func (c *Client) DoAndDecode(dst interface{}, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newTransactionError(req, resp, buf)
	}
	if dst == nil {
		return nil
	}
	return json.Unmarshal(buf, dst)
}

// Convert an arbitrary struct to url.Values. For example,
//
//     Foo{Bar: []int{1,2,3}, Baz: "waz"}
//
// becomes
//
//     url.Values{`bar`:`{"a":[1,2,3]}`,`Baz`:`waz`}
//
// params itself is returned if it is already an url.Values.
func anythingToValues(params interface{}) (url.Values, error) {
	if v, ok := params.(url.Values); ok {
		return v, nil
	}
	// TODO: Do this more efficiently, possibly using
	// json.Decode/Encode, so the whole thing doesn't have to get
	// encoded, decoded, and re-encoded.
	j, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	var generic map[string]interface{}
	dec := json.NewDecoder(bytes.NewBuffer(j))
	dec.UseNumber()
	err = dec.Decode(&generic)
	if err != nil {
		return nil, err
	}
	urlValues := url.Values{}
	for k, v := range generic {
		if v, ok := v.(string); ok {
			urlValues.Set(k, v)
			continue
		}
		if v, ok := v.(json.Number); ok {
			urlValues.Set(k, v.String())
			continue
		}
		if v, ok := v.(bool); ok {
			if v {
				urlValues.Set(k, "true")
			} else {
				// "foo=false", "foo=0", and "foo="
				// are all taken as true strings, so
				// don't send false values at all --
				// rely on the default being false.
			}
			continue
		}
		j, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(j, []byte("null")) {
			// don't add it to urlValues at all
			continue
		}
		urlValues.Set(k, string(j))
	}
	return urlValues, nil
}

// RequestAndDecode performs an API request and unmarshals the
// response (which must be JSON) into dst. Method and body arguments
// are the same as for http.NewRequest(). The given path is added to
// the server's scheme/host/port to form the request URL. The given
// params are passed via POST form or query string.
//
// path must not contain a query string.
func (c *Client) RequestAndDecode(dst interface{}, method, path string, body io.Reader, params interface{}) error {
	return c.RequestAndDecodeContext(c.context(), dst, method, path, body, params)
}

func (c *Client) RequestAndDecodeContext(ctx context.Context, dst interface{}, method, path string, body io.Reader, params interface{}) error {
	if body, ok := body.(io.Closer); ok {
		// Ensure body is closed even if we error out early
		defer body.Close()
	}
	urlString := c.apiURL(path)
	urlValues, err := anythingToValues(params)
	if err != nil {
		return err
	}
	if urlValues == nil {
		// Nothing to send
	} else if method == "GET" || method == "HEAD" || body != nil {
		// Must send params in query part of URL (FIXME: what
		// if resulting URL is too long?)
		u, err := url.Parse(urlString)
		if err != nil {
			return err
		}
		u.RawQuery = urlValues.Encode()
		urlString = u.String()
	} else {
		body = strings.NewReader(urlValues.Encode())
	}
	req, err := http.NewRequest(method, urlString, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	return c.DoAndDecode(dst, req)
}

type resource interface {
	resourceName() string
}

// UpdateBody returns an io.Reader suitable for use as an http.Request
// Body for a create or update API call.
func (c *Client) UpdateBody(rsc resource) io.Reader {
	j, err := json.Marshal(rsc)
	if err != nil {
		// Return a reader that returns errors.
		r, w := io.Pipe()
		w.CloseWithError(err)
		return r
	}
	v := url.Values{rsc.resourceName(): {string(j)}}
	return bytes.NewBufferString(v.Encode())
}

// WithRequestID returns a new shallow copy of c that sends the given
// X-Request-Id value (instead of a new randomly generated one) with
// each subsequent request that doesn't provide its own via context or
// header.
func (c *Client) WithRequestID(reqid string) *Client {
	cc := *c
	cc.ctx = ContextWithRequestID(cc.context(), reqid)
	return &cc
}

func (c *Client) context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *Client) httpClient() *http.Client {
	switch {
	case c.Client != nil:
		return c.Client
	case c.Insecure:
		return InsecureHTTPClient
	default:
		return DefaultSecureClient
	}
}

func (c *Client) apiURL(path string) string {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + c.APIHost + "/" + path
}

// DiscoveryDocument is the Arvados server's description of itself.
type DiscoveryDocument struct {
	BasePath                     string              `json:"basePath"`
	DefaultCollectionReplication int                 `json:"defaultCollectionReplication"`
	BlobSignatureTTL             int64               `json:"blobSignatureTtl"`
	GitURL                       string              `json:"gitUrl"`
	Schemas                      map[string]Schema   `json:"schemas"`
	Resources                    map[string]Resource `json:"resources"`
}

type Resource struct {
	Methods map[string]ResourceMethod `json:"methods"`
}

type ResourceMethod struct {
	HTTPMethod string         `json:"httpMethod"`
	Path       string         `json:"path"`
	Response   MethodResponse `json:"response"`
}

type MethodResponse struct {
	Ref string `json:"$ref"`
}

type Schema struct {
	UUIDPrefix string `json:"uuidPrefix"`
}

// DiscoveryDocument returns a *DiscoveryDocument. The returned object
// should not be modified: the same object may be returned by
// subsequent calls.
func (c *Client) DiscoveryDocument() (*DiscoveryDocument, error) {
	if c.dd != nil {
		return c.dd, nil
	}
	var dd DiscoveryDocument
	err := c.RequestAndDecode(&dd, "GET", "discovery/v1/apis/arvados/v1/rest", nil, nil)
	if err != nil {
		return nil, err
	}
	c.dd = &dd
	return c.dd, nil
}

var pdhRegexp = regexp.MustCompile(`^[0-9a-f]{32}\+\d+$`)

func (c *Client) modelForUUID(dd *DiscoveryDocument, uuid string) (string, error) {
	if pdhRegexp.MatchString(uuid) {
		return "Collection", nil
	}
	if len(uuid) != 27 {
		return "", fmt.Errorf("invalid UUID: %q", uuid)
	}
	infix := uuid[6:11]
	var model string
	for m, s := range dd.Schemas {
		if s.UUIDPrefix == infix {
			model = m
			break
		}
	}
	if model == "" {
		return "", fmt.Errorf("unrecognized type portion %q in UUID %q", infix, uuid)
	}
	return model, nil
}

func (c *Client) KindForUUID(uuid string) (string, error) {
	dd, err := c.DiscoveryDocument()
	if err != nil {
		return "", err
	}
	model, err := c.modelForUUID(dd, uuid)
	if err != nil {
		return "", err
	}
	return "arvados#" + strings.ToLower(model[:1]) + model[1:], nil
}

func (c *Client) PathForUUID(method, uuid string) (string, error) {
	dd, err := c.DiscoveryDocument()
	if err != nil {
		return "", err
	}
	model, err := c.modelForUUID(dd, uuid)
	if err != nil {
		return "", err
	}
	var resource string
	for r, rsc := range dd.Resources {
		if rsc.Methods["get"].Response.Ref == model {
			resource = r
			break
		}
	}
	if resource == "" {
		return "", fmt.Errorf("no resource for model: %q", model)
	}
	m, ok := dd.Resources[resource].Methods[method]
	if !ok {
		return "", fmt.Errorf("no method %q for resource %q", method, resource)
	}
	path := dd.BasePath + strings.Replace(m.Path, "{uuid}", uuid, -1)
	if path[0] == '/' {
		path = path[1:]
	}
	return path, nil
}
