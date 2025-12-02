package main

import (
	"os"
	"sync"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/providers/dns"
)

var setenvMux sync.Mutex

type providerWrapper struct {
	provider challenge.Provider
	envs     map[string]string
}

func (lp *providerWrapper) Present(domain, token, keyAuth string) error {
	reset := setenvs(lp.envs)
	defer reset()

	if err := lp.provider.Present(domain, token, keyAuth); err != nil {
		dns01.ClearFqdnCache()
		return err
	}

	return nil
}

func (lp *providerWrapper) CleanUp(domain, token, keyAuth string) error {
	reset := setenvs(lp.envs)
	defer reset()

	if err := lp.provider.CleanUp(domain, token, keyAuth); err != nil {
		dns01.ClearFqdnCache()
		return err
	}

	return nil
}

func newProvider(provider string, envs map[string]string) (*providerWrapper, error) {
	reset := setenvs(envs)
	defer reset()

	p, err := dns.NewDNSChallengeProviderByName(provider)
	if err != nil {
		return nil, err
	}

	// Build dns01 options based on envs
	var opts []dns01.ChallengeOption

	// Disable complete propagation requirement
	if val, ok := envs["LEGO_DISABLE_CP"]; ok && (strings.ToLower(val) == "true" || val == "1") {
		opts = append(opts, dns01.DisableCompletePropagationRequirement())
	}

	// Custom DNS resolvers
	if nameservers, ok := envs["LEGO_DNS_RESOLVERS"]; ok {
		resolvers := strings.Split(nameservers, ",")
		opts = append(opts, dns01.AddRecursiveNameservers(resolvers))
	}

	// Wrap the provider with dns01 options if any are set
	var finalProvider challenge.Provider = p
	if len(opts) > 0 {
		finalProvider, err = challenge.NewDNS01Provider(p, opts...)
		if err != nil {
			return nil, err
		}
	}

	return &providerWrapper{finalProvider, envs}, nil
}

func setenvs(envs map[string]string) func() {
	if envs == nil {
		return func() {}
	}

	setenvMux.Lock()

	origEnvs := make(map[string]string, len(envs))
	for name, value := range envs {
		origEnvs[name] = os.Getenv(name)
		os.Setenv(name, value)
	}

	return func() {
		defer setenvMux.Unlock()
		for name, value := range origEnvs {
			if value == "" {
				os.Unsetenv(name)
			} else {
				os.Setenv(name, value)
			}
		}
	}
}
