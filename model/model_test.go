package model

import (
	"reflect"
	"testing"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	routerName       = "deis-router"
	builderName      = "deis-builder"
	deisNamespace    = "deis"
	dhParamName      = "deis-router-dhparam"
	platformCertName = "deis-router-platform-cert"
)

func TestBuildRouterConfig(t *testing.T) {
	// Ensure a valid Router Deployment, Platform Cert, and DHParam result in the expected RouterConfig.
	replicas := int32(1)
	routerDeployment := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routerName,
			Namespace: deisNamespace,
			Annotations: map[string]string{
				"router.deis.io/nginx.defaultTimeout":             "1500s",
				"router.deis.io/nginx.ssl.bufferSize":             "6k",
				"router.deis.io/nginx.ssl.hsts.maxAge":            "1234",
				"router.deis.io/nginx.ssl.hsts.includeSubDomains": "true",
			},
			Labels: map[string]string{
				"heritage": "deis",
			},
		},
		Spec: appv1.DeploymentSpec{
			Strategy: appv1.DeploymentStrategy{
				Type:          appv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appv1.RollingUpdateDeployment{},
			},
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": routerName}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": routerName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "deis/router",
						},
					},
				},
			},
		},
	}

	platformCertSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      platformCertName,
			Namespace: deisNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tls.crt": []byte("foo"),
			"tls.key": []byte("bar"),
		},
	}

	dhParamSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dhParamName,
			Namespace: deisNamespace,
			Labels: map[string]string{
				"heritage": "deis",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"dhparam": []byte("bizbaz"),
		},
	}

	expectedConfig, err := newRouterConfig()
	if err != nil {
		t.Error(err)
	}
	sslConfig := newSSLConfig()
	hstsConfig := newHSTSConfig()
	platformCert := newCertificate("foo", "bar")

	// A value not set in the deployment annotations (should be default value).
	expectedConfig.MaxWorkerConnections = "768"

	// A simple string value.
	expectedConfig.DefaultTimeout = "1500s"

	// A nested value.
	sslConfig.BufferSize = "6k"
	sslConfig.DHParam = "bizbaz"

	// A nested+nested value.
	hstsConfig.MaxAge = 1234
	hstsConfig.IncludeSubDomains = true

	sslConfig.HSTSConfig = hstsConfig
	expectedConfig.SSLConfig = sslConfig

	expectedConfig.PlatformCertificate = platformCert

	actualConfig, err := buildRouterConfig(&routerDeployment, &platformCertSecret, &dhParamSecret)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(expectedConfig, actualConfig) {
		t.Errorf("Expected routerConfig does not match actual.")

		t.Errorf("Expected:\n")
		t.Errorf("%+v\n", expectedConfig)
		t.Errorf("Actual:\n")
		t.Errorf("%+v\n", actualConfig)
	}
}

func TestBuildBuilderConfig(t *testing.T) {
	// Ensure a Builder Service with annotations returns the expected BuilderConfig.
	builderService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builderName,
			Namespace: deisNamespace,
			Labels: map[string]string{
				"heritage": "deis",
			},
			Annotations: map[string]string{
				"router.deis.io/nginx.connectTimeout": "20s",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "ssh",
					Port: int32(2222),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 2223,
					},
				},
			},
			Selector: map[string]string{
				"app": builderName,
			},
			ClusterIP: "1.2.3.4",
		},
	}

	expectedConfig := BuilderConfig{
		// A value  set in the service annotations.
		ConnectTimeout: "20s",
		// A value not set in the service annotations (should be default value).
		TCPTimeout: "1200s",
		// A value determined by the service.spec.ClusterIP
		ServiceIP: "1.2.3.4",
	}

	actualConfig, err := buildBuilderConfig(&builderService)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(&expectedConfig, actualConfig) {
		t.Errorf("Expected builderConfig does not match actual.")

		t.Errorf("Expected:\n")
		t.Errorf("%+v\n", &expectedConfig)
		t.Errorf("Actual:\n")
		t.Errorf("%+v\n", actualConfig)
	}
}

func TestBuildCertificate(t *testing.T) {
	// Ensure a valid Cert Secret returns the expected certificate.
	validCertSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      platformCertName,
			Namespace: deisNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tls.crt": []byte("foo"),
			"tls.key": []byte("bar"),
		},
	}
	expectedCert := newCertificate("foo", "bar")
	actualCert, err := buildCertificate(&validCertSecret, "test-valid")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expectedCert, actualCert) {
		t.Errorf("Expected certificate does not match actual.")

		t.Errorf("Expected:\n")
		t.Errorf("%+v\n", expectedCert)
		t.Errorf("Actual:\n")
		t.Errorf("%+v\n", actualCert)
	}

	// Ensure an invalid Cert Secret returns nil.
	invalidCertSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      platformCertName,
			Namespace: deisNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"a": []byte("foo"),
			"b": []byte("bar"),
		},
	}

	invalidCert, err := buildCertificate(&invalidCertSecret, "test-invalid")
	if err != nil {
		t.Error(err)
	}
	if invalidCert != nil {
		t.Errorf("Expected invalid cert secret to return nil.")
	}
}

func TestBuildDHParam(t *testing.T) {
	// Ensure a valid DHParam Secret returns the expected DHParam string.
	expectedDHParam := "bizbaz"
	dhParamSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dhParamName,
			Namespace: deisNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"dhparam": []byte(expectedDHParam),
		},
	}

	actualDHParam, err := buildDHParam(&dhParamSecret)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expectedDHParam, actualDHParam) {
		t.Errorf("Expected DHParam %s does not match actual %s.", expectedDHParam, actualDHParam)
	}

	// Ensure an invalid DHParam Secret returns an empty string.
	invalidDHParamSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dhParamName,
			Namespace: deisNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"foo": []byte("bar"),
		},
	}
	actualInvalid, err := buildDHParam(&invalidDHParamSecret)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual("", actualInvalid) {
		t.Errorf("Invalid DHParam Secret should have returned empty string.")
	}
}
