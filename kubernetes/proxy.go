package kubernetes

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const proxyMaxBodyBytes = 16 << 20

func Proxy(clients *clientRegistry) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "proxy",
		Params: ast.Identifiers{"ctx", "target", "path", "options"},
		Func: func(args []any) (any, error) {
			return doProxy(clients, args)
		},
	}
}

type proxyTarget struct {
	Namespace string             `json:"namespace"`
	Service   string             `json:"service"`
	Pod       string             `json:"pod"`
	Port      intstr.IntOrString `json:"port"`
	Scheme    string             `json:"scheme"`
}

type proxyOptions struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func doProxy(clients *clientRegistry, args []any) (any, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("expected ctx, target, path, and options")
	}
	contextName, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("ctx must be a string")
	}
	target, err := decodeArg[proxyTarget](args[1], "target")
	if err != nil {
		return nil, err
	}
	if err := target.validate(); err != nil {
		return nil, err
	}
	rawPath, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}
	path, err := url.Parse(rawPath)
	if err != nil {
		return nil, fmt.Errorf("path: %w", err)
	}
	options, err := decodeArg[proxyOptions](args[3], "options")
	if err != nil {
		return nil, err
	}
	method := http.MethodGet
	if options.Method != "" {
		method = strings.ToUpper(options.Method)
	}

	cl, err := clients.get(contextName)
	if err != nil {
		return nil, err
	}

	u := cl.core.Get().
		Namespace(target.Namespace).
		Resource(target.resource()).
		Name(target.segment()).
		SubResource("proxy").
		URL()
	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + strings.TrimPrefix(path.Path, "/")
	u.RawQuery = path.RawQuery

	var body io.Reader
	if options.Body != "" {
		body = strings.NewReader(options.Body)
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	for k, v := range options.Headers {
		req.Header.Set(k, v)
	}

	resp, err := cl.core.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, proxyMaxBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(respBody) > proxyMaxBodyBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", proxyMaxBodyBytes)
	}

	headers := make(map[string]any, len(resp.Header))
	for k, v := range resp.Header {
		headers[strings.ToLower(k)] = strings.Join(v, ", ")
	}

	return map[string]any{
		"status":     float64(resp.StatusCode),
		"statusText": strings.TrimPrefix(resp.Status, strconv.Itoa(resp.StatusCode)+" "),
		"headers":    headers,
		"body":       string(respBody),
	}, nil
}

func (t proxyTarget) validate() error {
	if t.Namespace == "" {
		return fmt.Errorf("target.namespace is required")
	}
	if (t.Service == "") == (t.Pod == "") {
		return fmt.Errorf("target must set exactly one of service or pod")
	}
	switch t.Scheme {
	case "", "http", "https":
	default:
		return fmt.Errorf("target.scheme must be http or https")
	}
	return nil
}

func (t proxyTarget) resource() string {
	if t.Service != "" {
		return "services"
	}
	return "pods"
}

func (t proxyTarget) segment() string {
	name := t.Service + t.Pod
	port := ""
	if t.Port != (intstr.IntOrString{}) {
		port = t.Port.String()
	}
	switch {
	case t.Scheme == "https":
		return "https:" + name + ":" + port
	case port != "":
		return name + ":" + port
	default:
		return name
	}
}
