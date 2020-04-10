package handler

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/clstb/ksp/pkg/injector"
	corev1 "k8s.io/api/core/v1"
)

// Create is a http handler that handles the kubernetes secret post endpoint
func Create(
	proxyHandler http.Handler,
	injectors ...injector.Injector,
) http.HandlerFunc {
	secretFromRequest := func(r *http.Request) (*corev1.Secret, error) {
		var secret *corev1.Secret
		if err := json.NewDecoder(r.Body).Decode(&secret); err != nil {
			return secret, &SecretDecodeError{Err: err}
		}

		return secret, nil
	}
	secretToRequest := func(
		r *http.Request,
		secret *corev1.Secret,
	) error {
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(&secret); err != nil {
			return &SecretEncodeError{Err: err}
		}

		r.Body = ioutil.NopCloser(&body)
		r.ContentLength = int64(body.Len())

		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		secret, err := secretFromRequest(r)
		if err != nil {
			Error(w, r, err)
			return
		}

		secret, err = runInjectors(secret, injectors...)
		if err != nil {
			Error(w, r, err)
			return
		}

		if err := secretToRequest(r, secret); err != nil {
			Error(w, r, err)
			return
		}

		proxyHandler.ServeHTTP(w, r)
	}
}
