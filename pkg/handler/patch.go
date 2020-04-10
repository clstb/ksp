package handler

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/clstb/ksp/pkg/injector"
	"github.com/go-chi/chi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
)

// Patch is a http handler handles the kubernetes secret patch endpoint
func Patch(
	proxyHandler http.Handler,
	client kubernetes.Interface,
	injectors ...injector.Injector,
) http.HandlerFunc {
	patchRequestToOriginSecret := func(
		r *http.Request,
	) (*corev1.Secret, error) {
		name := chi.URLParam(r, "name")
		namespace := chi.URLParam(r, "namespace")
		patch, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, &BodyReadError{Err: err}
		}

		secret, err := client.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, &SecretGetError{Err: err}
		}

		secretBytes, err := json.Marshal(&secret)
		if err != nil {
			return nil, &SecretEncodeError{Err: err}
		}

		orignalSecretBytes, err := strategicpatch.StrategicMergePatch(
			secretBytes,
			patch,
			corev1.Secret{},
		)
		if err != nil {
			return nil, &PatchApplyError{
				Strategy: "strategic-merge",
				Type:     "corev1.Secret",
				Err:      err,
			}
		}

		var originSecret *corev1.Secret
		if err := json.Unmarshal(orignalSecretBytes, &originSecret); err != nil {
			return nil, &SecretDecodeError{Err: err}
		}

		return originSecret, nil
	}

	originSecretToPatchRequest := func(
		r *http.Request,
		originSecret *corev1.Secret,
	) error {
		secret, err := client.CoreV1().Secrets(originSecret.Namespace).Get(originSecret.Name, metav1.GetOptions{})
		if err != nil {
			return &SecretGetError{Err: err}
		}

		secretBytes, err := json.Marshal(&secret)
		if err != nil {
			return &SecretEncodeError{Err: err}
		}

		originSecretBytes, err := json.Marshal(&originSecret)
		if err != nil {
			return &SecretEncodeError{Err: err}
		}

		patch, err := strategicpatch.CreateTwoWayMergePatch(secretBytes, originSecretBytes, corev1.Secret{})
		if err != nil {
			return &PatchCreateError{
				Strategy: "two-way-merge",
				Type:     "corev1.Secret",
				Err:      err,
			}
		}

		body := bytes.NewBuffer(patch)
		r.Body = ioutil.NopCloser(body)
		r.ContentLength = int64(body.Len())

		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		originSecret, err := patchRequestToOriginSecret(r)
		if err != nil {
			Error(w, r, err)
			return
		}

		originSecret, err = runInjectors(originSecret, injectors...)
		if err != nil {
			Error(w, r, err)
			return
		}

		if err := originSecretToPatchRequest(r, originSecret); err != nil {
			Error(w, r, err)
			return
		}

		proxyHandler.ServeHTTP(w, r)
	}
}
