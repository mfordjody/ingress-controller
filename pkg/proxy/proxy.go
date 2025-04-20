package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/demdxx/gocast"
	"github.com/rs/zerolog/log"
	"ingress-controller/pkg/server"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	llog "log"
	"net/http"
	"strings"
)

func Render(client *kubernetes.Clientset) {
	// Retrieve all Ingress resources across all namespaces
	ingressList, err := client.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list ingresses")
		return
	}

	// Iterate through each Ingress resource
	serverMap := make(map[string]*server.Server)
	for _, ing := range ingressList.Items {
		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				host := rule.Host + path.Path
				if _, ok := serverMap[path.Path]; !ok {
					var srv *server.Server
					srv = &server.Server{
						Name:   path.Backend.Service.Name,
						Port:   gocast.ToString(path.Backend.Service.Port.Number),
						Host:   strings.ReplaceAll(host, "*", ".*"),
						Client: client,
					}
					http.Handle(path.Path, srv)
					serverMap[path.Path] = srv
				} else {
					srv := serverMap[path.Path]
					srv.Name = path.Backend.Service.Name
					srv.Port = gocast.ToString(path.Backend.Service.Port.Number)
					srv.Host = strings.ReplaceAll(host, "*", ".*")
				}
			}
		}

		var tlsCertMap = make(map[string]tls.Certificate)
		for _, sr := range ing.Spec.TLS {
			if sr.SecretName != "" {
				secrets, err := client.CoreV1().Secrets(ing.Namespace).Get(context.TODO(), sr.SecretName, metav1.GetOptions{})
				if err != nil {
					log.Error().Err(err).Str("namespace", ing.Namespace).Str("name", sr.SecretName).Msg("failed to get TLS secret")
					continue
				}
				cert, err := tls.X509KeyPair(secrets.Data["tls.crt"], secrets.Data["tls.key"])
				if err != nil {
					log.Error().Err(err).Str("namespace", ing.Namespace).Str("name", sr.SecretName).Msg("invalid tls certificate")
					continue
				}
				tlsCertMap[sr.SecretName] = cert
			}
		}
	}
}

func Run(client *kubernetes.Clientset, ingressClass string) error {
	// Save the mapping of hostname -> TLS cert (for SNI)
	tlsCertMap := map[string]tls.Certificate{}

	// Retrieve all Ingress resources across all namespaces
	ingresses, err := client.NetworkingV1().Ingresses("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list ingresses: %w", err)
	}

	// Iterate through each Ingress resource
	for _, ingress := range ingresses.Items {
		for _, secrets := range ingress.Spec.TLS {
			secretName := secrets.SecretName
			if secretName == "" {
				continue
			}
			secret, err := client.CoreV1().Secrets(ingress.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Str("namespace", ingress.Namespace).Str("secret", secretName).Msg("Failed to load TLS secret")
				continue
			}
			cert, err := tls.X509KeyPair(secret.Data["tls.crt"], secret.Data["tls.key"])
			if err != nil {
				log.Error().Err(err).Str("namespace", ingress.Namespace).Str("secret", secretName).Msg("Invalid TLS cert")
				continue
			}
			for _, host := range secrets.Hosts {
				tlsCertMap[host] = cert
				log.Info().Str("host", host).Msg("Loaded certificate for host")
			}
		}

		// Check IngressClass
		if ingress.Spec.IngressClassName != nil && *ingress.Spec.IngressClassName == ingressClass {
			log.Info().Str("ingress", ingress.Name).Msg("Found matching ingress class, starting proxy")
		} else if *ingress.Spec.IngressClassName != ingressClass {
			continue
		}
	}

	go func() {
		log.Info().Str("addr", "80").Msg("starting insecure HTTP server")
		if err := http.ListenAndServe(":80", nil); err != nil {
			log.Fatal().Err(err).Msg("Failed to start HTTP")
		}
	}()

	// HTTPS Server with SNI
	httpsServer := &http.Server{
		Addr:     ":443",
		Handler:  nil,
		ErrorLog: llog.New(io.Discard, "", 0), // Shield error log
		TLSConfig: &tls.Config{
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if cert, ok := tlsCertMap[clientHello.ServerName]; ok {
					return &cert, nil
				}
				log.Warn().Str("host", clientHello.ServerName).Msg("No matching certificate found")
				return nil, fmt.Errorf("no certificate for host %s", clientHello.ServerName)
			},
		},
	}

	log.Info().Str("addr", "443").Msg("starting secure HTTP server")
	return httpsServer.ListenAndServeTLS("", "")
}
