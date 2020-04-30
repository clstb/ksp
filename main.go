package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/clstb/ksp/cmd/gpg/encrypt"
	"github.com/clstb/ksp/cmd/proxy"
	"github.com/urfave/cli"
)

func main() {
	app := &cli.App{
		Name: "ksp - Kubernetes Secret Proxy",
		Authors: []cli.Author{
			{
				Name:  "Claas Stoertenbecker",
				Email: "claas.stoertenbecker@gmail.com",
			},
		},
		Commands: []cli.Command{
			{
				Name:    "proxy",
				Aliases: []string{"p"},
				Usage:   "run the ksp proxy server",
				Action:  proxy.Run,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:   "port",
						Usage:  "port to listen on",
						Value:  8000,
						EnvVar: "KSP_PROXY_PORT",
					},
					&cli.StringFlag{
						Name:   "config",
						Usage:  "path to kubeconfig file",
						Value:  filepath.Join(os.Getenv("HOME"), ".kube/config"),
						EnvVar: "KSP_PROXY_CONFIG",
					},
					&cli.BoolFlag{
						Name:   "verbose",
						Usage:  "enable verbose logging",
						EnvVar: "KSP_PROXY_VERBOSE",
					},
					&cli.BoolFlag{
						Name:   "injector-gpg",
						Usage:  "enable gpg injector",
						EnvVar: "KSP_PROXY_INJECTOR_GPG",
					},
				},
			},
			{
				Name:    "gpg",
				Aliases: []string{"g"},
				Subcommands: []cli.Command{
					{
						Name:    "encrypt",
						Aliases: []string{"e"},
						Usage:   "encrypt secrets in provided file",
						Action:  encrypt.Run,
						Flags: []cli.Flag{
							&cli.StringSliceFlag{
								Name:     "keys",
								Usage:    "public keys to encrypt with",
								Required: true,
								EnvVar:   "KSP_GPG_KEYS",
							},
							&cli.StringFlag{
								Name:     "file",
								Usage:    "file to decrypt",
								Required: true,
							},
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
