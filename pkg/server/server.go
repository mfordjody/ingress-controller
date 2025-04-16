package server

import (
	"context"
	"crypto/tls"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Server struct {
	Client      *kubernetes.Clientset
	Host        string
	ServiceName string
	ServicePort string
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Retrieve all Ingress resources across all namespaces
	ingressList, err := s.Client.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		http.Error(w, "Failed to list Ingresses", http.StatusInternalServerError)
		return
	}

	// Check if the request is using TLS (i.e., HTTPS)
	if r.TLS == nil {
		// If not, redirect the HTTP request to HTTPS
		redirectURL := "https://" + r.Host + r.URL.Path
		http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
		return
	}

	// Iterate through each Ingress resource
	for _, ingress := range ingressList.Items {
		ingressNamespace := ingress.Namespace
		ingressName := ingress.Name

		// Get detailed information for the current Ingress
		ingresses, err := s.Client.NetworkingV1().Ingresses(ingressNamespace).Get(context.TODO(), ingressName, metav1.GetOptions{})
		if err != nil {
			http.Error(w, "Ingress not found", http.StatusInternalServerError)
			return
		}

		// Get the associated Service resource in the same namespace
		svc, err := s.Client.CoreV1().Services(ingresses.Namespace).Get(context.TODO(), s.ServiceName, metav1.GetOptions{})
		if err != nil {
			http.Error(w, "Service not found", http.StatusInternalServerError)
			return
		}

		// Determine the protocol based on TLS in Ingress
		var scheme string
		if len(ingress.Spec.TLS) > 0 {
			scheme = "https://"
		} else {
			scheme = "http://"
		}

		// Build the backend URL from the service ClusterIP and port
		serviceScheme := "http://"
		serviceIP := svc.Spec.ClusterIP
		backendURL, err := url.Parse(serviceScheme + serviceIP + ":" + s.ServicePort)
		if err != nil {
			http.Error(w, "BackendURL error", http.StatusInternalServerError)
			return
		}

		log.Info().
			Str("host", r.Host).
			Str("path", r.URL.Path).
			Str("backend", scheme+serviceIP+":"+s.ServicePort).
			Msg("proxying request")

		// Create a reverse proxy to forward the request to the backend service
		proxy := httputil.NewSingleHostReverseProxy(backendURL)

		// TLS verification
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		}

		// Serve the request via the reverse proxy
		proxy.ServeHTTP(w, r)
	}
}
