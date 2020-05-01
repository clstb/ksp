package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clstb/ksp/pkg/handler"
	"github.com/clstb/ksp/pkg/injector"
	mw "github.com/clstb/ksp/pkg/middleware"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Run executes the proxy command
func Run(c *cli.Context) error {
	ctx := context.Background()

	port := c.Int("port")
	config := c.String("config")
	verbose := c.Bool("verbose")

	injectorGPG := c.Bool("injector-gpg")

	kubeConfig, err := clientcmd.LoadFromFile(config)
	if err != nil {
		return err
	}
	proxyConfig := kubeConfig.DeepCopy()

	r := chi.NewRouter()
	if verbose {
		r.Use(middleware.Logger)
	}

	var injectors []injector.Injector
	{
		if injectorGPG {
			gpg, err := injector.NewGPG(ctx)
			if err != nil {
				return err
			}
			injectors = append(injectors, gpg)
		}
	}

	cert, key, err := keyPair()
	if err != nil {
		return err
	}

	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}

	for k, v := range kubeConfig.Clusters {
		cluster := &api.Cluster{
			Server:                   fmt.Sprintf("https://localhost:%d/%s", port, k),
			CertificateAuthorityData: cert,
		}
		proxyConfig.Clusters[k] = cluster

		url, err := url.Parse(v.Server)
		if err != nil {
			return err
		}

		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		rootCAs.AppendCertsFromPEM(v.CertificateAuthorityData)

		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: rootCAs,
			},
		}

		config, err := clientcmd.NewDefaultClientConfig(*kubeConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return err
		}

		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}

		log.Printf("http: redirecting %s to %s\n", cluster.Server, v.Server)
		r.Route("/"+k, func(r chi.Router) {
			r.Use(mw.TrimPrefix("/" + k))
			r.Handle("/*", proxy)
			r.Post("/api/v1/namespaces/{namespace}/secrets", handler.Create(proxy, injectors...))
			r.Patch("/api/v1/namespaces/{namespace}/secrets/{name}", handler.Patch(proxy, client, injectors...))
		})
	}

	go func() {
		c := &tls.Config{Certificates: []tls.Certificate{tlsCert}}
		s := &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      r,
			TLSConfig:    c,
			ReadTimeout:  time.Minute,
			WriteTimeout: time.Minute,
		}

		log.Printf("http: listening for requests...\n")
		log.Fatal(s.ListenAndServeTLS("", ""))
	}()

	if err := clientcmd.WriteToFile(*kubeConfig, config+".ksp.bak"); err != nil {
		return err
	}

	if err := clientcmd.WriteToFile(*proxyConfig, config); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	if err := clientcmd.WriteToFile(*kubeConfig, config); err != nil {
		return err
	}

	return nil
}

func keyPair() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating rsa key failed")
	}

	tpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(2, 0, 0),
		BasicConstraintsValid: true,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageCertSign,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generating cert failed")
	}

	var certBuf bytes.Buffer
	if err := pem.Encode(&certBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}); err != nil {
		return nil, nil, errors.Wrap(err, "pem encoding cert failed")
	}
	pemCert := certBuf.Bytes()

	var keyBuf bytes.Buffer
	if err := pem.Encode(&keyBuf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return nil, nil, errors.Wrap(err, "pem encoding rsa key failed")
	}
	pemKey := keyBuf.Bytes()

	return pemCert, pemKey, nil
}
