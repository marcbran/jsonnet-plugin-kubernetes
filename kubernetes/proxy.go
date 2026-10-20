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

type ProxyInput struct {
	Context string
	Target  ProxyTarget
	Path    string
	Options ProxyOptions
}

type ProxyTarget struct {
	Namespace string             `json:"namespace"`
	Service   string             `json:"service"`
	Pod       string             `json:"pod"`
	Port      intstr.IntOrString `json:"port"`
	Scheme    string             `json:"scheme"`
}

type ProxyOptions struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type ProxyOutput struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

func Proxy(clients *clientRegistry) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "proxy",
		Params: ast.Identifiers{"ctx", "target", "path", "options"},
		Func: func(args []any) (any, error) {
			pi, err := parseProxyInput(args)
			if err != nil {
				return nil, err
			}
			out, err := runProxy(clients, pi)
			if err != nil {
				return nil, err
			}
			return encodeOutput(out)
		},
	}
}

func parseProxyInput(args []any) (ProxyInput, error) {
	if len(args) != 4 {
		return ProxyInput{}, fmt.Errorf("expected ctx, target, path, and options")
	}
	contextName, ok := args[0].(string)
	if !ok {
		return ProxyInput{}, fmt.Errorf("ctx must be a string")
	}
	target, err := decodeArg[ProxyTarget](args[1], "target")
	if err != nil {
		return ProxyInput{}, err
	}
	if target.Namespace == "" {
		return ProxyInput{}, fmt.Errorf("target.namespace is required")
	}
	if (target.Service == "") == (target.Pod == "") {
		return ProxyInput{}, fmt.Errorf("target must set exactly one of service or pod")
	}
	switch target.Scheme {
	case "", "http", "https":
	default:
		return ProxyInput{}, fmt.Errorf("target.scheme must be http or https")
	}
	path, ok := args[2].(string)
	if !ok {
		return ProxyInput{}, fmt.Errorf("path must be a string")
	}
	if _, err := url.Parse(path); err != nil {
		return ProxyInput{}, fmt.Errorf("path: %w", err)
	}
	options, err := decodeArg[ProxyOptions](args[3], "options")
	if err != nil {
		return ProxyInput{}, err
	}
	if options.Method == "" {
		options.Method = http.MethodGet
	}
	options.Method = strings.ToUpper(options.Method)
	return ProxyInput{Context: contextName, Target: target, Path: path, Options: options}, nil
}

func runProxy(clients *clientRegistry, pi ProxyInput) (ProxyOutput, error) {
	path, err := url.Parse(pi.Path)
	if err != nil {
		return ProxyOutput{}, fmt.Errorf("path: %w", err)
	}

	cl, err := clients.get(pi.Context)
	if err != nil {
		return ProxyOutput{}, err
	}

	u := cl.core.Get().
		Namespace(pi.Target.Namespace).
		Resource(pi.Target.resource()).
		Name(pi.Target.segment()).
		SubResource("proxy").
		URL()
	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + strings.TrimPrefix(path.Path, "/")
	u.RawQuery = path.RawQuery

	var body io.Reader
	if pi.Options.Body != "" {
		body = strings.NewReader(pi.Options.Body)
	}
	req, err := http.NewRequest(pi.Options.Method, u.String(), body)
	if err != nil {
		return ProxyOutput{}, err
	}
	for k, v := range pi.Options.Headers {
		req.Header.Set(k, v)
	}

	resp, err := cl.core.Client.Do(req)
	if err != nil {
		return ProxyOutput{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, proxyMaxBodyBytes+1))
	if err != nil {
		return ProxyOutput{}, err
	}
	if len(respBody) > proxyMaxBodyBytes {
		return ProxyOutput{}, fmt.Errorf("response body exceeds %d bytes", proxyMaxBodyBytes)
	}

	headers := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		headers[strings.ToLower(k)] = strings.Join(v, ", ")
	}

	return ProxyOutput{
		Status:     resp.StatusCode,
		StatusText: strings.TrimPrefix(resp.Status, strconv.Itoa(resp.StatusCode)+" "),
		Headers:    headers,
		Body:       string(respBody),
	}, nil
}

func (t ProxyTarget) resource() string {
	if t.Service != "" {
		return "services"
	}
	return "pods"
}

func (t ProxyTarget) segment() string {
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
