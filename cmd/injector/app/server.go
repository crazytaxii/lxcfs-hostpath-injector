package app

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/crazytaxii/lxcfs-sidecar-injector/pkg/webhook"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

const (
	defaultPort     = 443                           // default port listening
	defaultCertFile = "/etc/webhook/certs/cert.pem" // default path 4 certficate pem
	defaultKeyFile  = "/etc/webhook/certs/key.pem"  // default path 4 key pem
)

// ServerRunOptions runs this webhook server.
type ServerRunOptions struct {
	Port       int
	CertFile   string
	KeyFile    string
	ConfigFile string
}

// Create a *cobra.Command object with default parameters.
func NewMutateWebhookServer() *cobra.Command {
	opt := &ServerRunOptions{}
	cmd := &cobra.Command{
		Use:   "webhook-server",
		Short: "Kubernetest mutate webhook server for lxcfs hostpath injecting.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opt)
		},
	}

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	klogFlags.Set("logtostderr", "true")
	fs := cmd.Flags()
	opt.addFlags(fs)
	fs.AddGoFlagSet(klogFlags)

	return cmd
}

func (opt *ServerRunOptions) addFlags(fs *pflag.FlagSet) {
	fs.IntVar(&opt.Port, "port", defaultPort, "Webhook server port")
	fs.StringVar(&opt.CertFile, "tls-cert-file", defaultCertFile, "File containing the x509 Certificate for HTTPS")
	fs.StringVar(&opt.KeyFile, "tls-key-file", defaultKeyFile, "File containing the x509 private key to --tls-cert-file")
	fs.StringVar(&opt.ConfigFile, "sidecar-config-file", webhook.DefaultConfigFile, "File containing sidecar configuration")
}

// Running the specified API Server.
// This should never exit.
func run(opt *ServerRunOptions) error {
	// config for webhook server
	cfg, err := webhook.LoadWebhookServerConfig(opt.ConfigFile)
	if err != nil {
		klog.Errorf("Failed to laod sidecar config: %v", err)
		return err
	}

	// TLS
	pair, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile)
	if err != nil {
		klog.Errorf("Failed to load key pair: %v", err)
		return err
	}

	whsvr := &webhook.WebhookServer{
		Config: cfg,
		Server: &http.Server{
			Addr: fmt.Sprintf(":%v", opt.Port),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{pair},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.Mutate)
	whsvr.Server.Handler = mux

	go func() {
		if err := whsvr.Server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	klog.Infof("Got OS shutdown signal, shutting down wenhook server.")
	return whsvr.Server.Shutdown(context.Background())
}
