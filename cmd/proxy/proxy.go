package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/clstb/ksp/pkg/handler"
	"github.com/clstb/ksp/pkg/injector"
	mw "github.com/clstb/ksp/pkg/middleware"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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

	for k, v := range kubeConfig.Clusters {
		cluster := &api.Cluster{
			Server:                fmt.Sprintf("https://localhost:%d/%s", port, k),
			InsecureSkipTLSVerify: true,
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
		log.Printf("http: listening for requests...\n")
		if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "server.crt", "server.key", r); err != nil {
			log.Fatal(err)
		}
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
