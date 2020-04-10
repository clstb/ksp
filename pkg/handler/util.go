package handler

import (
	"github.com/clstb/ksp/pkg/injector"
	corev1 "k8s.io/api/core/v1"
)

func runInjectors(
	secret *corev1.Secret,
	injectors ...injector.Injector,
) (*corev1.Secret, error) {
	for _, injector := range injectors {
		injectedSecret, err := injector.Inject(secret)
		if err != nil {
			return nil, &InjectorError{Err: err}
		}
		secret = injectedSecret
	}

	return secret, nil
}
