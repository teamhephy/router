package model

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strings"

	"github.com/teamhephy/router/utils"
	modelerUtility "github.com/teamhephy/router/utils/modeler"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	prefix               string = "router.deis.io"
	modelerFieldTag      string = "key"
	modelerConstraintTag string = "constraint"
)

var (
	namespace   = utils.GetOpt("POD_NAMESPACE", "default")
	modeler     = modelerUtility.NewModeler(prefix, modelerFieldTag, modelerConstraintTag, true)
	listOptions metav1.ListOptions
)

func init() {
	labelMap := labels.Set{fmt.Sprintf("%s/routable", prefix): "true"}
	listOptions = metav1.ListOptions{LabelSelector: labelMap.AsSelector().String(), FieldSelector: fields.Everything().String()}
}

// RouterConfig is the primary type used to encapsulate all router configuration.
type RouterConfig struct {
	WorkerProcesses          string      `key:"workerProcesses" constraint:"^(auto|[1-9]\\d*)$"`
	MaxWorkerConnections     string      `key:"maxWorkerConnections" constraint:"^[1-9]\\d*$"`
	TrafficStatusZoneSize    string      `key:"trafficStatusZoneSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	DefaultTimeout           string      `key:"defaultTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	ServerNameHashMaxSize    string      `key:"serverNameHashMaxSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	ServerNameHashBucketSize string      `key:"serverNameHashBucketSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	GzipConfig               *GzipConfig `key:"gzip"`
	BodySize                 string      `key:"bodySize" constraint:"^[0-9]\\d*[kKmM]?$"`
	LargeHeaderBuffersCount  string      `key:"largeHeaderBuffersCount" constraint:"^[1-9]\\d*$"`
	LargeHeaderBuffersSize   string      `key:"largeHeaderBuffersSize" constraint:"^[0-9]\\d*[kKmM]?$"`
	ProxyRealIPCIDRs         []string    `key:"proxyRealIpCidrs" constraint:"^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\\/([0-9]|[1-2][0-9]|3[0-2]))?(\\s*,\\s*)?)+$"`
	ErrorLogLevel            string      `key:"errorLogLevel" constraint:"^(debug|info|notice|warn|error|crit|alert|emerg)$"`
	PlatformDomain           string      `key:"platformDomain" constraint:"(?i)^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z0-9]+(-*[a-z0-9]+)+$"`
	UseProxyProtocol         bool        `key:"useProxyProtocol" constraint:"(?i)^(true|false)$"`
	DisableServerTokens      bool        `key:"disableServerTokens" constraint:"(?i)^(true|false)$"`
	EnforceWhitelists        bool        `key:"enforceWhitelists" constraint:"(?i)^(true|false)$"`
	DefaultWhitelist         []string    `key:"defaultWhitelist" constraint:"^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\\/([0-9]|[1-2][0-9]|3[0-2]))?(\\s*,\\s*)?)+$"`
	WhitelistMode            string      `key:"whitelistMode" constraint:"^(extend|override)$"`
	EnableRegexDomains       bool        `key:"enableRegexDomains" constraint:"(?i)^(true|false)$"`
	LoadModsecurityModule    bool        `key:"loadModsecurityModule" constraint:"(?i)^(true|false)$"`
	DefaultServiceIP         string      `key:"defaultServiceIP"`
	DefaultAppName           string      `key:"defaultAppName"`
	DefaultServiceEnabled    bool        `key:"defaultServiceEnabled" constraint:"(?i)^(true|false)$"`
	RequestIDs               bool        `key:"requestIDs" constraint:"(?i)^(true|false)$"`
	RequestStartHeader       bool        `key:"requestStartHeader" constraint:"(?i)^(true|false)$"`
	SSLConfig                *SSLConfig  `key:"ssl"`
	AppConfigs               []*AppConfig
	BuilderConfig            *BuilderConfig
	PlatformCertificate      *Certificate
	HTTP2Enabled             bool                `key:"http2Enabled" constraint:"(?i)^(true|false)$"`
	LogFormat                string              `key:"logFormat"`
	ProxyBuffersConfig       *ProxyBuffersConfig `key:"proxyBuffers"`
	ReferrerPolicy           string              `key:"referrerPolicy" constraint:"^(no-referrer|no-referrer-when-downgrade|origin|origin-when-cross-origin|same-origin|strict-origin|strict-origin-when-cross-origin|unsafe-url|none)$"`
}

func newRouterConfig() (*RouterConfig, error) {
	proxyBuffersConfig, err := newProxyBuffersConfig(nil)
	if err != nil {
		return nil, err
	}
	return &RouterConfig{
		WorkerProcesses:          "auto",
		MaxWorkerConnections:     "768",
		TrafficStatusZoneSize:    "1m",
		DefaultTimeout:           "1300s",
		ServerNameHashMaxSize:    "512",
		ServerNameHashBucketSize: "64",
		GzipConfig:               newGzipConfig(),
		BodySize:                 "1m",
		LargeHeaderBuffersCount:  "4",
		LargeHeaderBuffersSize:   "32k",
		ProxyRealIPCIDRs:         []string{"10.0.0.0/8"},
		DisableServerTokens:      false,
		ErrorLogLevel:            "error",
		UseProxyProtocol:         false,
		EnforceWhitelists:        false,
		WhitelistMode:            "extend",
		EnableRegexDomains:       false,
		LoadModsecurityModule:    false,
		RequestIDs:               false,
		RequestStartHeader:       false,
		SSLConfig:                newSSLConfig(),
		DefaultServiceEnabled:    false,
		DefaultAppName:           "",
		DefaultServiceIP:         "",
		HTTP2Enabled:             true,
		LogFormat:                `[$time_iso8601] - $app_name - $remote_addr - $remote_user - $status - "$request" - $bytes_sent - "$http_referer" - "$http_user_agent" - "$server_name" - $upstream_addr - $http_host - $upstream_response_time - $request_time`,
		ProxyBuffersConfig:       proxyBuffersConfig,
		ReferrerPolicy:           "",
	}, nil
}

// GzipConfig encapsulates gzip configuration.
type GzipConfig struct {
	Enabled     bool   `key:"enabled" constraint:"(?i)^(true|false)$"`
	CompLevel   string `key:"compLevel" constraint:"^[1-9]$"`
	Disable     string `key:"disable"`
	HTTPVersion string `key:"httpVersion" constraint:"^(1\\.0|1\\.1)$"`
	MinLength   string `key:"minLength" constraint:"^\\d+$"`
	Proxied     string `key:"proxied" constraint:"^((off|expired|no-cache|no-store|private|no_last_modified|no_etag|auth|any)\\s*)+$"`
	Types       string `key:"types" constraint:"(?i)^([a-z\\d]+/[a-z\\d][a-z\\d+\\-\\.]*[a-z\\d]\\s*)+$"`
	Vary        string `key:"vary" constraint:"^(on|off)$"`
}

func newGzipConfig() *GzipConfig {
	return &GzipConfig{
		Enabled:     true,
		CompLevel:   "5",
		Disable:     "msie6",
		HTTPVersion: "1.1",
		MinLength:   "256",
		Proxied:     "any",
		Types:       "application/atom+xml application/javascript application/json application/rss+xml application/vnd.ms-fontobject application/x-font-ttf application/x-web-app-manifest+json application/xhtml+xml application/xml font/opentype image/svg+xml image/x-icon text/css text/plain text/x-component",
		Vary:        "on",
	}
}

// AppConfig encapsulates the configuration for all routes to a single back end.
type AppConfig struct {
	Name                      string
	Domains                   []string `key:"domains" constraint:"(?i)^((([a-z0-9]+(-*[a-z0-9]+)*)|((\\*\\.)?[a-z0-9]+(-*[a-z0-9]+)*\\.)+[a-z0-9]+(-*[a-z0-9]+)+)(\\s*,\\s*)?)+$"`
	RegexDomain               string   `key:"regexDomain"`
	Whitelist                 []string `key:"whitelist" constraint:"^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(\\/([0-9]|[1-2][0-9]|3[0-2]))?(\\s*,\\s*)?)+$"`
	ConnectTimeout            string   `key:"connectTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	TCPTimeout                string   `key:"tcpTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	ServiceIP                 string
	CertMappings              map[string]string `key:"certificates" constraint:"(?i)^((([a-z0-9]+(-*[a-z0-9]+)*)|((\\*\\.)?[a-z0-9]+(-*[a-z0-9]+)*\\.)+[a-z0-9]+(-*[a-z0-9]+)+):([a-z0-9]+(-*[a-z0-9]+)*)(\\s*,\\s*)?)+$"`
	Certificates              map[string]*Certificate
	Available                 bool
	Maintenance               bool            `key:"maintenance" constraint:"(?i)^(true|false)$"`
	DisableRequestStartHeader bool            `key:"disableRequestStartHeader" constraint:"(?i)^(true|false)$"`
	ReferrerPolicy            string          `key:"referrerPolicy" constraint:"^(no-referrer|no-referrer-when-downgrade|origin|origin-when-cross-origin|same-origin|strict-origin|strict-origin-when-cross-origin|unsafe-url|none)$"`
	SSLConfig                 *SSLConfig      `key:"ssl"`
	Nginx                     *NginxAppConfig `key:"nginx"`
	ProxyLocations            []string        `key:"proxyLocations"`
	ProxyDomain               string          `key:"proxyDomain"`
	Locations                 []*Location
}

// Location represents a location block inside a back end server block.
type Location struct {
	App  *AppConfig
	Path string
}

func newAppConfig(routerConfig *RouterConfig) (*AppConfig, error) {
	nginxConfig, err := newNginxAppConfig(routerConfig)
	if err != nil {
		return nil, err
	}
	return &AppConfig{
		ConnectTimeout: "30s",
		TCPTimeout:     routerConfig.DefaultTimeout,
		Certificates:   make(map[string]*Certificate),
		SSLConfig:      newSSLConfig(),
		Nginx:          nginxConfig,
	}, nil
}

// BuilderConfig encapsulates the configuration of the deis-builder-- if it's in use.
type BuilderConfig struct {
	ConnectTimeout string `key:"connectTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	TCPTimeout     string `key:"tcpTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	ServiceIP      string
}

func newBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		ConnectTimeout: "10s",
		TCPTimeout:     "1200s",
	}
}

// Certificate represents an SSL certificate for use in securing routable applications.
type Certificate struct {
	Cert string
	Key  string
}

func newCertificate(cert string, key string) *Certificate {
	return &Certificate{
		Cert: cert,
		Key:  key,
	}
}

// SSLConfig represents SSL-related configuration options.
type SSLConfig struct {
	Enforce           bool        `key:"enforce" constraint:"(?i)^(true|false)$"`
	Protocols         string      `key:"protocols" constraint:"^((SSLv[2-3]|TLSv1(?:\\.[1-3])?)\\s*)+$"`
	Ciphers           string      `key:"ciphers" constraint:"^((\\b[\\w.!+-]+\\b)+(:?@(STRENGTH|SECLEVEL=[0-5]))?(:([!+-]\\b)?|$))*(((\\b[\\w.+-]+\\b)+|(\\[(\\b[\\w.|+-]+\\b)+\\]))(:|$))*$"`
	SessionCache      string      `key:"sessionCache" constraint:"^(off|none|((builtin(:[1-9]\\d*)?|shared:\\w+:[1-9]\\d*[kKmM]?)\\s*){1,2})$"`
	SessionTimeout    string      `key:"sessionTimeout" constraint:"^[1-9]\\d*(ms|[smhdwMy])?$"`
	UseSessionTickets bool        `key:"useSessionTickets" constraint:"(?i)^(true|false)$"`
	BufferSize        string      `key:"bufferSize" constraint:"^[1-9]\\d*[kKmM]?$"`
	HSTSConfig        *HSTSConfig `key:"hsts"`
	EarlyDataMethods  string      `key:"earlyDataMethods" constraint:"^((GET|HEAD|POST|PUT|DELETE|PATCH|OPTIONS)(\\|\\b|$))*$"`
	DHParam           string
}

func newSSLConfig() *SSLConfig {
	return &SSLConfig{
		Enforce:   false,
		Protocols: "TLSv1.2 TLSv1.3",
		// Source: https://ssl-config.mozilla.org/#server=nginx&version=1.22.1&config=intermediate&openssl=1.1.1n&hsts=false&ocsp=false&guideline=5.6
		Ciphers:           "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384",
		SessionTimeout:    "10m",
		UseSessionTickets: true,
		BufferSize:        "4k",
		HSTSConfig:        newHSTSConfig(),
		EarlyDataMethods:  "GET|HEAD|OPTIONS",
	}
}

// HSTSConfig represents configuration options having to do with HTTP Strict Transport Security.
type HSTSConfig struct {
	Enabled           bool `key:"enabled" constraint:"(?i)^(true|false)$"`
	MaxAge            int  `key:"maxAge" constraint:"^[1-9]\\d*$"`
	IncludeSubDomains bool `key:"includeSubDomains" constraint:"(?i)^(true|false)$"`
	Preload           bool `key:"preload" constraint:"(?i)^(true|false)$"`
}

func newHSTSConfig() *HSTSConfig {
	return &HSTSConfig{
		Enabled:           false,
		MaxAge:            15552000, // 180 days
		IncludeSubDomains: false,
		Preload:           false,
	}
}

// NginxAppConfig is a wrapper for all Nginx-specific app configurations. These
// options shouldn't be expected to be universally supported by alternative
// router implementations.
type NginxAppConfig struct {
	ProxyBuffersConfig *ProxyBuffersConfig `key:"proxyBuffers"`
}

func newNginxAppConfig(routerConfig *RouterConfig) (*NginxAppConfig, error) {
	proxyBuffersConfig, err := newProxyBuffersConfig(routerConfig.ProxyBuffersConfig)
	if err != nil {
		return nil, err
	}
	return &NginxAppConfig{
		ProxyBuffersConfig: proxyBuffersConfig,
	}, nil
}

// ProxyBuffersConfig represents configuration options having to do with Nginx
// proxy buffers.
type ProxyBuffersConfig struct {
	Enabled  bool   `key:"enabled" constraint:"(?i)^(true|false)$"`
	Number   int    `key:"number" constraint:"^[1-9]\\d*$"`
	Size     string `key:"size" constraint:"^[1-9]\\d*[kKmM]?$"`
	BusySize string `key:"busySize" constraint:"^[1-9]\\d*[kKmM]?$"`
}

func newProxyBuffersConfig(proxyBuffersConfig *ProxyBuffersConfig) (*ProxyBuffersConfig, error) {
	if proxyBuffersConfig != nil {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		dec := gob.NewDecoder(&buf)
		err := enc.Encode(proxyBuffersConfig)
		if err != nil {
			return nil, err
		}
		var copy *ProxyBuffersConfig
		err = dec.Decode(&copy)
		if err != nil {
			return nil, err
		}
		return copy, nil
	}
	return &ProxyBuffersConfig{
		Number:   8,
		Size:     "4k",
		BusySize: "8k",
	}, nil
}

// Build creates a RouterConfig configuration object by querying the k8s API for
// relevant metadata concerning itself and all routable services.
func Build(kubeClient *kubernetes.Clientset) (*RouterConfig, error) {
	// Get all relevant information from k8s:
	//   deis-router deployment
	//   All services with label "routable=true"
	//   deis-builder service, if it exists
	// These are used to construct a model...
	routerDeployment, err := getDeployment(kubeClient)
	if err != nil {
		return nil, err
	}
	appServices, err := getAppServices(kubeClient)
	if err != nil {
		return nil, err
	}
	// builderService might be nil if it's not found and that's ok.
	builderService, err := getBuilderService(kubeClient)
	if err != nil {
		return nil, err
	}
	platformCertSecret, err := getSecret(kubeClient, "deis-router-platform-cert", namespace)
	if err != nil {
		return nil, err
	}
	dhParamSecret, err := getSecret(kubeClient, "deis-router-dhparam", namespace)
	if err != nil {
		return nil, err
	}
	// Build the model...
	routerConfig, err := build(kubeClient, routerDeployment, platformCertSecret, dhParamSecret, appServices, builderService)
	if err != nil {
		return nil, err
	}
	return routerConfig, nil
}

func getDeployment(kubeClient *kubernetes.Clientset) (*appv1.Deployment, error) {
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get("deis-router", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func getAppServices(kubeClient *kubernetes.Clientset) (*corev1.ServiceList, error) {
	serviceClient := kubeClient.CoreV1().Services(metav1.NamespaceAll)
	services, err := serviceClient.List(listOptions)
	if err != nil {
		return nil, err
	}
	return services, nil
}

// getBuilderService will return the service named "deis-builder" from the same namespace as
// the router, but will return nil (without error) if no such service exists.
func getBuilderService(kubeClient *kubernetes.Clientset) (*corev1.Service, error) {
	serviceClient := kubeClient.CoreV1().Services(namespace)
	service, err := serviceClient.Get("deis-builder", metav1.GetOptions{})
	if err != nil {
		statusErr, ok := err.(*errors.StatusError)
		// If the issue is just that no deis-builder was found, that's ok.
		if ok && statusErr.Status().Code == 404 {
			// We'll just return nil instead of a found *metav1.Service.
			return nil, nil
		}
		return nil, err
	}
	return service, nil
}

func getSecret(kubeClient *kubernetes.Clientset, name string, ns string) (*corev1.Secret, error) {
	secretClient := kubeClient.CoreV1().Secrets(ns)
	secret, err := secretClient.Get(name, metav1.GetOptions{})
	if err != nil {
		statusErr, ok := err.(*errors.StatusError)
		// If the issue is just that no such secret was found, that's ok.
		if ok && statusErr.Status().Code == 404 {
			// We'll just return nil instead of a found *metav1.Secret
			return nil, nil
		}
		return nil, err
	}
	return secret, nil
}

func build(kubeClient *kubernetes.Clientset, routerDeployment *appv1.Deployment, platformCertSecret *corev1.Secret, dhParamSecret *corev1.Secret, appServices *corev1.ServiceList, builderService *corev1.Service) (*RouterConfig, error) {
	routerConfig, err := buildRouterConfig(routerDeployment, platformCertSecret, dhParamSecret)
	if err != nil {
		return nil, err
	}
	for _, appService := range appServices.Items {
		appConfig, err := buildAppConfig(kubeClient, appService, routerConfig)
		if err != nil {
			return nil, err
		}
		if appConfig != nil {
			routerConfig.AppConfigs = append(routerConfig.AppConfigs, appConfig)
		}
	}
	err = linkLocations(routerConfig.AppConfigs)
	if err != nil {
		return nil, err
	}
	addRootLocations(routerConfig.AppConfigs)
	if builderService != nil {
		builderConfig, err := buildBuilderConfig(builderService)
		if err != nil {
			return nil, err
		}
		if builderConfig != nil {
			routerConfig.BuilderConfig = builderConfig
		}
	}
	return routerConfig, nil
}

func appByDomain(appConfigs []*AppConfig, domain string) *AppConfig {
	for _, app := range appConfigs {
		for _, appDomain := range app.Domains {
			if domain == appDomain {
				return app
			}
		}
	}
	return nil
}

func linkLocations(appConfigs []*AppConfig) error {
	for _, app := range appConfigs {
		if app.ProxyDomain != "" && len(app.ProxyLocations) > 0 {
			targetApp := appByDomain(appConfigs, app.ProxyDomain)
			if targetApp == nil {
				return fmt.Errorf("Can't find ProxyDomain '%s' in any application", app.ProxyDomain)
			}

			for _, loc := range app.ProxyLocations {
				location := &Location{App: app, Path: loc}
				targetApp.Locations = append(targetApp.Locations, location)
			}
		}
	}
	return nil
}

func addRootLocations(appConfigs []*AppConfig) {
	for _, app := range appConfigs {
		rootLocation := &Location{App: app, Path: "/"}
		app.Locations = append(app.Locations, rootLocation)
	}
}

func buildRouterConfig(routerDeployment *appv1.Deployment, platformCertSecret *corev1.Secret, dhParamSecret *corev1.Secret) (*RouterConfig, error) {
	routerConfig, err := newRouterConfig()
	if err != nil {
		return nil, err
	}
	err = modeler.MapToModel(routerDeployment.Annotations, "nginx", routerConfig)
	if err != nil {
		return nil, err
	}
	if platformCertSecret != nil {
		platformCertificate, err := buildCertificate(platformCertSecret, "platform")
		if err != nil {
			return nil, err
		}
		routerConfig.PlatformCertificate = platformCertificate
	}
	if dhParamSecret != nil {
		dhParam, err := buildDHParam(dhParamSecret)
		if err != nil {
			return nil, err
		}
		routerConfig.SSLConfig.DHParam = dhParam
	}
	return routerConfig, nil
}

func buildAppConfig(kubeClient *kubernetes.Clientset, service corev1.Service, routerConfig *RouterConfig) (*AppConfig, error) {
	appConfig, err := newAppConfig(routerConfig)
	if err != nil {
		return nil, err
	}
	appConfig.Name = service.Labels["app"]
	// If we didn't get the app name from the app label, fall back to inferring the app name from
	// the service's own name.
	if appConfig.Name == "" {
		appConfig.Name = service.Name
	}
	// if app name and Namespace are not same then combine the two as it
	// makes deis services (as an example) clearer, such as deis/controller
	if appConfig.Name != service.Namespace {
		appConfig.Name = service.Namespace + "/" + appConfig.Name
	}
	err = modeler.MapToModel(service.Annotations, "", appConfig)
	if err != nil {
		return nil, err
	}

	// If no domains are found, we don't have the information we need to build routes
	// to this application.  Abort.
	if len(appConfig.Domains) == 0 {
		return nil, nil
	}
	// Step through the domains, and decide which cert, if any, will be used for securing each.
	// For each that is a FQDN, we'll look to see if a corresponding cert-bearing secret also
	// exists.  If so, that will be used.  If a domain isn't an FQDN we will use the default cert--
	// even if that is nil.
	for _, domain := range appConfig.Domains {
		if strings.Contains(domain, ".") {
			// Look for a cert-bearing secret for this domain.
			if certMapping, ok := appConfig.CertMappings[domain]; ok {
				secretName := fmt.Sprintf("%s-cert", certMapping)
				certSecret, err := getSecret(kubeClient, secretName, service.Namespace)
				if err != nil {
					return nil, err
				}
				if certSecret != nil {
					certificate, err := buildCertificate(certSecret, domain)
					if err != nil {
						return nil, err
					}
					appConfig.Certificates[domain] = certificate
				}
			}
		} else {
			appConfig.Certificates[domain] = routerConfig.PlatformCertificate
		}
	}
	appConfig.ServiceIP = service.Spec.ClusterIP
	endpointsClient := kubeClient.CoreV1().Endpoints(service.Namespace)
	endpoints, err := endpointsClient.Get(service.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	appConfig.Available = len(endpoints.Subsets) > 0 && len(endpoints.Subsets[0].Addresses) > 0
	return appConfig, nil
}

func buildBuilderConfig(service *corev1.Service) (*BuilderConfig, error) {
	builderConfig := newBuilderConfig()
	builderConfig.ServiceIP = service.Spec.ClusterIP
	err := modeler.MapToModel(service.Annotations, "nginx", builderConfig)
	if err != nil {
		return nil, err
	}
	return builderConfig, nil
}

func buildCertificate(certSecret *corev1.Secret, context string) (*Certificate, error) {
	cert, ok := certSecret.Data["tls.crt"]
	// If no cert is found in the secret, warn and return nil
	if !ok {
		log.Printf("WARN: The k8s secret intended to convey the %s certificate contained no entry \"tls.crt\".\n", context)
		return nil, nil
	}
	key, ok := certSecret.Data["tls.key"]
	// If no key is found in the secret, warn and return nil
	if !ok {
		log.Printf("WARN: The k8s secret intended to convey the %s certificate key contained no entry \"tls.key\".\n", context)
		return nil, nil
	}
	certStr := string(cert[:])
	keyStr := string(key[:])
	return newCertificate(certStr, keyStr), nil
}

func buildDHParam(dhParamSecret *corev1.Secret) (string, error) {
	dhParam, ok := dhParamSecret.Data["dhparam"]
	// If no dhparam is found in the secret, warn and return ""
	if !ok {
		log.Println("WARN: The k8s secret intended to convey the dhparam contained no entry \"dhparam\".")
		return "", nil
	}
	return string(dhParam), nil
}
