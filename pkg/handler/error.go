package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

// BodyReadError occurs when the read of a http request body failed
type BodyReadError struct {
	Err error
}

func (e *BodyReadError) Error() string {
	return "reading body failed: " + e.Err.Error()
}

func (e *BodyReadError) Unwrap() error {
	return e.Err
}

// SecretDecodeError occurs when decoding json to corev1.Secret failed
type SecretDecodeError struct {
	Err error
}

func (e *SecretDecodeError) Error() string {
	return "decoding secret failed: " + e.Err.Error()
}

func (e *SecretDecodeError) Unwrap() error {
	return e.Err
}

// SecretEncodeError occurs when encoding a corev1.Secret to json failed
type SecretEncodeError struct {
	Err error
}

func (e *SecretEncodeError) Error() string {
	return "encoding secret failed: " + e.Err.Error()
}

func (e *SecretEncodeError) Unwrap() error {
	return e.Err
}

// SecretGetError occurs when retrieving a corev1.Secret from a remote cluster failed
type SecretGetError struct {
	Err error
}

func (e *SecretGetError) Error() string {
	return "getting secret failed: " + e.Err.Error()
}

func (e *SecretGetError) Unwrap() error {
	return e.Err
}

// PatchApplyError occurs when applying a patch to a kubernetes resources failed
type PatchApplyError struct {
	Strategy string
	Type     string
	Err      error
}

func (e *PatchApplyError) Error() string {
	s := fmt.Sprintf("applying %s patch of %s failed: ", e.Strategy, e.Type)
	return s + e.Err.Error()
}

func (e *PatchApplyError) Unwrap() error {
	return e.Err
}

// PatchCreateError occurs when creating a patch from two kubernetes resources failed
type PatchCreateError struct {
	Strategy string
	Type     string
	Err      error
}

func (e *PatchCreateError) Error() string {
	s := fmt.Sprintf("creating %s patch of %s failed: ", e.Strategy, e.Type)
	return s + e.Err.Error()
}

func (e *PatchCreateError) Unwrap() error {
	return e.Err
}

// InjectorError occurs when injecting a corev1.Secret failed
type InjectorError struct {
	Err error
}

func (e *InjectorError) Error() string {
	return "injecting secret failed: " + e.Err.Error()
}

func (e *InjectorError) Unwrap() error {
	return e.Err
}

// Error is a http handler that is called when errors in other handlers occur
func Error(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	log.Printf("error: http: %v\n", err)
	switch {
	case errors.Is(err, &BodyReadError{}):
		http.Error(w, "ksp: internal server error: reading request body failed", http.StatusInternalServerError)
	case errors.Is(err, &SecretDecodeError{}):
		http.Error(w, "ksp: bad request: invalid secret", http.StatusBadRequest)
	case errors.Is(err, &SecretEncodeError{}):
		http.Error(w, "ksp: internal server error: encoding secret failed", http.StatusInternalServerError)
	case errors.Is(err, &SecretGetError{}):
		http.Error(w, "ksp: failed dependency: getting secret failed", http.StatusFailedDependency)
	case errors.Is(err, &PatchApplyError{}):
		http.Error(w, "ksp: internal server error: applying patch failed", http.StatusInternalServerError)
	case errors.Is(err, &PatchCreateError{}):
		http.Error(w, "ksp: internal server error: creating patch failed", http.StatusInternalServerError)
	case errors.Is(err, &InjectorError{}):
		http.Error(w, "ksp: failed dependency: secret injector failed", http.StatusFailedDependency)
	default:
		http.Error(w, "ksp: unknown internal server error", http.StatusInternalServerError)
	}
}
