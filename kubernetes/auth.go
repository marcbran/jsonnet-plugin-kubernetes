package kubernetes

import (
	"golang.org/x/oauth2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

type Auth struct {
	tokenSource transport.ResettableTokenSource
}

type AuthByContext func(contextName string) (Auth, error)

func KubeconfigAuth() Auth {
	return Auth{}
}

func TokenAuth(ts oauth2.TokenSource) Auth {
	return Auth{tokenSource: transport.NewCachedTokenSource(ts)}
}

func (a Auth) apply(cfg *rest.Config) {
	if a.tokenSource == nil {
		return
	}
	cfg.ExecProvider = nil
	cfg.AuthProvider = nil
	cfg.BearerToken = ""
	cfg.BearerTokenFile = ""
	cfg.Username = ""
	cfg.Password = ""
	cfg.Wrap(transport.ResettableTokenSourceWrapTransport(a.tokenSource))
}
