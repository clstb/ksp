package injector

import (
	corev1 "k8s.io/api/core/v1"
)

// Injector is a minimal interface to modify corev1.Secret
type Injector interface {
	Inject(*corev1.Secret) (*corev1.Secret, error)
}
