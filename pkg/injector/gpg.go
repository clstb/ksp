package injector

import (
	"context"

	cli "github.com/gopasspw/gopass/pkg/backend/crypto/gpg/cli"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// GPG is an injector that does gpg decryption on secret data
// It uses the local gpg cli
type GPG struct {
	ctx context.Context
	cli *cli.GPG
}

// NewGPG creates a new gpg injector
func NewGPG(ctx context.Context) (*GPG, error) {
	cli, err := cli.New(ctx, cli.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "wrapping gpg cli failed")
	}

	return &GPG{
		ctx: ctx,
		cli: cli,
	}, nil
}

// Inject satisfies the injector interface
func (i *GPG) Inject(secret *corev1.Secret) (*corev1.Secret, error) {
	t, ok := secret.Annotations["ksp/inject"]
	if !ok || t != "gpg" {
		return secret, nil
	}

	for k, v := range secret.Data {
		v, err := i.Decrypt(v)
		if err != nil {
			return secret, errors.Wrap(err, "decrypting failed")
		}
		secret.Data[k] = v
	}

	return secret, nil
}

// Encrypt encrypts byte data with the local gpg cli
func (i *GPG) Encrypt(
	keys []string,
	b []byte,
) ([]byte, error) {
	return i.cli.Encrypt(i.ctx, b, keys)
}

// Decrypt decrypts byte data with the local gpg cli
func (i *GPG) Decrypt(
	b []byte,
) ([]byte, error) {
	return i.cli.Decrypt(i.ctx, b)
}
