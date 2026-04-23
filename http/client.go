package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/arcgolabs/clientx"
	"github.com/samber/oops"
	"resty.dev/v3"
)

// DefaultClient is the default HTTP client implementation.
type DefaultClient struct {
	raw      *resty.Client
	baseURL  string
	hooks    []clientx.Hook
	policies []clientx.Policy
}

// New creates a Client from cfg and applies opts.
func New(cfg Config, opts ...Option) (Client, error) {
	normalized, err := cfg.NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{}
	if normalized.TLS.Enabled || normalized.TLS.InsecureSkipVerify || normalized.TLS.ServerName != "" {
		transport.TLSClientConfig = &tls.Config{
			//nolint:gosec // This client must support explicitly configured insecure TLS for development and controlled environments.
			InsecureSkipVerify: normalized.TLS.InsecureSkipVerify,
			ServerName:         normalized.TLS.ServerName,
		}
	}

	c := resty.New().
		SetBaseURL(normalized.BaseURL).
		SetTimeout(normalized.Timeout).
		SetTransport(transport)

	if normalized.UserAgent != "" {
		c.SetHeader("User-Agent", normalized.UserAgent)
	}
	if normalized.Headers != nil {
		normalized.Headers.Range(func(key, value string) bool {
			c.SetHeader(key, value)
			return true
		})
	}

	client := &DefaultClient{raw: c, baseURL: normalized.BaseURL}
	if normalized.Retry.Enabled {
		appendPolicies(clientx.NewRetryPolicy(clientx.RetryPolicyConfig{
			MaxAttempts: max(1, normalized.Retry.MaxRetries+1),
			BaseDelay:   normalized.Retry.WaitMin,
			MaxDelay:    normalized.Retry.WaitMax,
		}))(client)
	}

	clientx.Apply(client, opts...)
	return client, nil
}

// Close releases idle HTTP connections held by the underlying transport.
func (c *DefaultClient) Close() error {
	if c == nil || c.raw == nil {
		return nil
	}
	if raw := c.raw.Client(); raw != nil {
		raw.CloseIdleConnections()
	}
	return nil
}

// Raw returns the underlying resty client.
func (c *DefaultClient) Raw() *resty.Client {
	return c.raw
}

// R creates a new resty request from the underlying client.
func (c *DefaultClient) R() *resty.Request {
	return c.raw.R()
}

// Execute runs an HTTP request through the configured policy chain.
func (c *DefaultClient) Execute(ctx context.Context, req *resty.Request, method, endpoint string) (*resty.Response, error) {
	op := strings.ToLower(strings.TrimSpace(method))
	if op == "" {
		op = "request"
	}
	addr := c.resolveAddr(endpoint)
	operation := clientx.Operation{
		Protocol: clientx.ProtocolHTTP,
		Kind:     clientx.OperationKindRequest,
		Op:       op,
		Network:  "http",
		Addr:     addr,
	}

	resp, err := invokeWithPolicies(ctx, operation, func(execCtx context.Context) (*resty.Response, error) {
		workingReq := req
		if workingReq == nil {
			workingReq = c.R()
		}
		workingReq.SetContext(execCtx)

		start := time.Now()
		resp, err := workingReq.Execute(method, endpoint)
		if err != nil {
			wrappedErr := wrapClientError(op, addr, err)
			clientx.EmitIO(c.hooks, clientx.IOEvent{
				Protocol: clientx.ProtocolHTTP,
				Op:       op,
				Addr:     addr,
				Duration: time.Since(start),
				Err:      wrappedErr,
			})
			return nil, wrappedErr
		}
		clientx.EmitIO(c.hooks, clientx.IOEvent{
			Protocol: clientx.ProtocolHTTP,
			Op:       op,
			Addr:     addr,
			Bytes:    max(0, int(resp.Size())),
			Duration: time.Since(start),
		})
		return resp, nil
	}, c.policies...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *DefaultClient) resolveAddr(endpoint string) string {
	addr := strings.TrimSpace(endpoint)
	if addr == "" {
		return c.baseURL
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") || c.baseURL == "" {
		return addr
	}
	base := strings.TrimRight(c.baseURL, "/")
	if strings.HasPrefix(addr, "/") {
		return base + addr
	}
	return base + "/" + addr
}

func invokeWithPolicies(
	ctx context.Context,
	operation clientx.Operation,
	fn func(context.Context) (*resty.Response, error),
	policies ...clientx.Policy,
) (*resty.Response, error) {
	resp, err := clientx.InvokeWithPolicies(ctx, operation, fn, policies...)
	if err != nil {
		return nil, oops.In("clientx/http").
			With("op", operation.Op, "addr", operation.Addr, "network", operation.Network, "protocol", operation.Protocol, "operation_kind", operation.Kind).
			Wrapf(err, "execute http operation")
	}
	return resp, nil
}

func wrapClientError(op, addr string, err error) error {
	return oops.In("clientx/http").
		With("op", op, "addr", addr, "protocol", clientx.ProtocolHTTP).
		Wrapf(clientx.WrapError(clientx.ProtocolHTTP, op, addr, err), "http %s %s", op, addr)
}
