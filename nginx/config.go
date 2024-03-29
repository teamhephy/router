package nginx

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/teamhephy/router/model"
)

const (
	confTemplate = `{{ $routerConfig := . }}daemon off;
pid /tmp/nginx.pid;
worker_processes {{ $routerConfig.WorkerProcesses }};

{{ if $routerConfig.LoadModsecurityModule -}}
# Loading the Modsecurity connector nginx dynamic module
load_module modules/ngx_http_modsecurity_module.so;
{{- end }}

events {
	worker_connections {{ $routerConfig.MaxWorkerConnections }};
	# multi_accept on;
}

http {
	# basic settings
	sendfile on;
	tcp_nopush on;
	tcp_nodelay on;

	vhost_traffic_status_zone shared:vhost_traffic_status:{{ $routerConfig.TrafficStatusZoneSize }};

	# The timeout value must be greater than the front facing load balancers timeout value.
	# Default is the deis recommended timeout value for ELB - 1200 seconds + 100s extra.
	keepalive_timeout {{ $routerConfig.DefaultTimeout }};

	types_hash_max_size 2048;
	server_names_hash_max_size {{ $routerConfig.ServerNameHashMaxSize }};
	server_names_hash_bucket_size {{ $routerConfig.ServerNameHashBucketSize }};

	{{ $gzipConfig := $routerConfig.GzipConfig }}{{ if $gzipConfig.Enabled }}gzip on;
	gzip_comp_level {{ $gzipConfig.CompLevel }};
	gzip_disable {{ $gzipConfig.Disable }};
	gzip_http_version {{ $gzipConfig.HTTPVersion }};
	gzip_min_length {{ $gzipConfig.MinLength }};
	gzip_types {{ $gzipConfig.Types }};
	gzip_proxied {{ $gzipConfig.Proxied }};
	gzip_vary {{ $gzipConfig.Vary }};{{ end }}

	client_max_body_size {{ $routerConfig.BodySize }};
	large_client_header_buffers {{ $routerConfig.LargeHeaderBuffersCount }} {{ $routerConfig.LargeHeaderBuffersSize }};

	{{ if $routerConfig.DisableServerTokens -}}
	server_tokens off;
	{{- end}}
	{{ range $realIPCIDR := $routerConfig.ProxyRealIPCIDRs -}}
	set_real_ip_from {{ $realIPCIDR }};
	{{ end -}}
	real_ip_recursive on;
	{{ if $routerConfig.UseProxyProtocol -}}
	real_ip_header proxy_protocol;
	{{- else -}}
	real_ip_header X-Forwarded-For;
	{{- end }}

	log_format upstreaminfo '{{ $routerConfig.LogFormat }}';

	access_log /tmp/logpipe upstreaminfo;
	error_log  /tmp/logpipe {{ $routerConfig.ErrorLogLevel }};

	map $http_upgrade $connection_upgrade {
		default upgrade;
		'' close;
	}

	# The next two maps work together to determine the $access_scheme:
	# 1. Determine if SSL may have been offloaded by the load balancer, in such cases, an HTTP request should be
	# treated as if it were HTTPs.
	map $http_x_forwarded_proto $tmp_access_scheme {
		default $scheme;               # if X-Forwarded-Proto header is empty, $tmp_access_scheme will be the actual protocol used
		"~^(.*, ?)?http$" "http";      # account for the possibility of a comma-delimited X-Forwarded-Proto header value
		"~^(.*, ?)?https$" "https";    # account for the possibility of a comma-delimited X-Forwarded-Proto header value
		"~^(.*, ?)?ws$" "ws";      # account for the possibility of a comma-delimited X-Forwarded-Proto header value
		"~^(.*, ?)?wss$" "wss";    # account for the possibility of a comma-delimited X-Forwarded-Proto header value
	}
	# 2. If the request is an HTTPS/wss request, upgrade $access_scheme to https/wss, regardless of what the X-Forwarded-Proto
	# header might say.
	map $scheme $access_scheme {
		default $tmp_access_scheme;
		"https" "https";
		"wss"	"wss";
	}

	# Determine the forwarded port:
	# 1. First map the unprivileged ports that Nginx (as a non-root user) actually listen on to the
	# familiar, equivalent privileged ports. (These would be the ports the k8s service listens on.)
	map $server_port $standard_server_port {
		default $server_port;
		8080 80;
		6443 443;
	}
	# 2. If the X-Forwarded-Port header has been set already (e.g. by a load balancer), use its
	# value, otherwise, the port we're forwarding for is the $standard_server_port we determined
	# above.
	map $http_x_forwarded_port $forwarded_port {
		default $http_x_forwarded_port;
		'' $standard_server_port;
	}
	# uri_scheme will be the scheme to use when the ssl is enforced.
	map $access_scheme $uri_scheme {
		default "https";
		"ws"	"wss";
	}


	{{ $sslConfig := $routerConfig.SSLConfig }}
	{{ $hstsConfig := $sslConfig.HSTSConfig }}{{ if $hstsConfig.Enabled }}
	# HSTS instructs the browser to replace all HTTP links with HTTPS links for this domain until maxAge seconds from now.
	# The $sts variable is used later in each server block.
	map $access_scheme $sts {
		'https' 'max-age={{ $hstsConfig.MaxAge }}{{ if $hstsConfig.IncludeSubDomains }}; includeSubDomains{{ end }}{{ if $hstsConfig.Preload }}; preload{{ end }}';
	}
	{{ end }}
	{{ if ne $sslConfig.EarlyDataMethods "" }}
	# Only allow early data (TLSv1.3 0-RTT) for select methods
	map $request_method $ssl_block_early_data {
		default $ssl_early_data;
		"~^{{ $sslConfig.EarlyDataMethods }}$" 0;
	}
	{{ end }}

	{{ if $routerConfig.RequestIDs }}
		map $http_x_correlation_id $correlation_id {
			default "$http_x_correlation_id,$request_id";
			'' $request_id;
		}
	{{ end }}

	{{/* Since HSTS headers are not permitted on HTTP requests, 301 redirects to HTTPS resources are also necessary. */}}
	{{/* This means we force HTTPS if HSTS is enabled. */}}
	{{ $enforceSecure := or $sslConfig.Enforce $hstsConfig.Enabled }}

	{{ if $routerConfig.DefaultServiceEnabled }}
	server {
		listen 8080 default_server{{ if $routerConfig.UseProxyProtocol }} proxy_protocol{{ end }};
		server_name _;
		server_name_in_redirect off;
		port_in_redirect off;
		set $app_name "{{ $routerConfig.DefaultAppName }}";
		vhost_traffic_status_filter_by_set_key {{ $routerConfig.DefaultAppName }} application::*;
		location ~ ^/healthz/?$ {
			access_log off;
			default_type 'text/plain';
			return 200;
		}

		location / {
			proxy_buffering {{ if $routerConfig.ProxyBuffersConfig.Enabled }}on{{ else }}off{{ end }};
			proxy_buffer_size {{ $routerConfig.ProxyBuffersConfig.Size }};
			proxy_buffers {{ $routerConfig.ProxyBuffersConfig.Number }} {{ $routerConfig.ProxyBuffersConfig.Size }};
			proxy_busy_buffers_size {{ $routerConfig.ProxyBuffersConfig.BusySize }};
			proxy_set_header Host $host;
			proxy_set_header X-Forwarded-For $remote_addr;
			proxy_set_header X-Forwarded-Proto $access_scheme;
			proxy_set_header X-Forwarded-Port $forwarded_port;
			proxy_redirect off;
			proxy_http_version 1.1;
			proxy_set_header Upgrade $http_upgrade;
			proxy_set_header Connection $connection_upgrade;
			{{ if ne $sslConfig.EarlyDataMethods "" }}proxy_set_header Early-Data $ssl_early_data;{{ end }}
			proxy_pass http://{{$routerConfig.DefaultServiceIP}}:80;
		}
	}
	{{ else }}

	# Default server handles requests for unmapped hostnames, including healthchecks
	server {
		listen 8080 default_server reuseport{{ if $routerConfig.UseProxyProtocol }} proxy_protocol{{ end }};
		listen 6443 default_server ssl {{ if $routerConfig.HTTP2Enabled }}http2{{ end }} {{ if $routerConfig.UseProxyProtocol }}proxy_protocol{{ end }};

		set $app_name "router-default-vhost";
		ssl_protocols {{ $sslConfig.Protocols }};
		{{ if ne $sslConfig.Ciphers "" }}ssl_ciphers {{ $sslConfig.Ciphers }};{{ end }}
		ssl_prefer_server_ciphers on;
		ssl_early_data {{ if ne $sslConfig.EarlyDataMethods "" }}on{{ else }}off{{ end }};
		{{ if $routerConfig.PlatformCertificate }}
		ssl_certificate /opt/router/ssl/platform.crt;
		ssl_certificate_key /opt/router/ssl/platform.key;
		{{ else }}
		ssl_certificate /opt/router/ssl/default/default.crt;
		ssl_certificate_key /opt/router/ssl/default/default.key;
		{{ end }}
		{{ if ne $sslConfig.SessionCache "" }}ssl_session_cache {{ $sslConfig.SessionCache }};
		ssl_session_timeout {{ $sslConfig.SessionTimeout }};{{ end }}
		ssl_session_tickets {{ if $sslConfig.UseSessionTickets }}on{{ else }}off{{ end }};
		ssl_buffer_size {{ $sslConfig.BufferSize }};
		{{ if ne $sslConfig.DHParam "" }}ssl_dhparam /opt/router/ssl/dhparam.pem;{{ end }}
		{{ if ne $routerConfig.ReferrerPolicy "" }}
		add_header Referrer-Policy {{ $routerConfig.ReferrerPolicy }};
		{{ end }}
		server_name _;
		location ~ ^/healthz/?$ {
			access_log off;
			default_type 'text/plain';
			return 200;
		}
		location / {
			return 404;
		}
	}
	{{ end }}

	# Healthcheck on 9090 -- never uses proxy_protocol
	server {
		listen 9090 default_server;
		server_name _;
		set $app_name "router-healthz";
		location ~ ^/healthz/?$ {
			access_log off;
			default_type 'text/plain';
			return 200;
		}
		location ~ ^/stats/?$ {
			vhost_traffic_status_display;
			vhost_traffic_status_display_format json;
			allow 127.0.0.1;
			deny all;
		}
	 	location /nginx_status {
      			stub_status on;
		      	allow 127.0.0.1;
		      	deny all;
		}
		location / {
			return 404;
		}
	}

	{{range $appConfig := $routerConfig.AppConfigs}}{{range $domain := $appConfig.Domains}}server {
		listen 8080{{ if $routerConfig.UseProxyProtocol }} proxy_protocol{{ end }};
		server_name {{ if and $routerConfig.EnableRegexDomains (contains $domain $appConfig.RegexDomain)}}~^{{$domain}}\.(?<domain>.+)$ ~^{{$appConfig.RegexDomain}}\.(?<domain>.+)${{ else if contains "." $domain }}{{ $domain }}{{ else if ne $routerConfig.PlatformDomain "" }}{{ $domain }}.{{ $routerConfig.PlatformDomain }}{{ else }}~^{{ $domain }}\.(?<domain>.+)${{ end }};
		server_name_in_redirect off;
		port_in_redirect off;
		set $app_name "{{ $appConfig.Name }}";

		{{ if $routerConfig.LoadModsecurityModule -}}
		# Turning on modsecurity if modsecurity module loaded
		modsecurity on;
		modsecurity_rules_file /opt/router/conf/modsecurity.conf;
		{{- end }}

		{{ if index $appConfig.Certificates $domain }}
		listen 6443 ssl {{ if $routerConfig.HTTP2Enabled }}http2{{ end }} {{ if $routerConfig.UseProxyProtocol }}proxy_protocol{{ end }};
		ssl_protocols {{ $sslConfig.Protocols }};
		{{ if ne $sslConfig.Ciphers "" }}ssl_ciphers {{ $sslConfig.Ciphers }};{{ end }}
		ssl_prefer_server_ciphers on;
		ssl_early_data {{ if ne $sslConfig.EarlyDataMethods "" }}on{{ else }}off{{ end }};
		ssl_certificate /opt/router/ssl/{{ $domain }}.crt;
		ssl_certificate_key /opt/router/ssl/{{ $domain }}.key;
		{{ if ne $sslConfig.SessionCache "" }}ssl_session_cache {{ $sslConfig.SessionCache }};
		ssl_session_timeout {{ $sslConfig.SessionTimeout }};{{ end }}
		ssl_session_tickets {{ if $sslConfig.UseSessionTickets }}on{{ else }}off{{ end }};
		ssl_buffer_size {{ $sslConfig.BufferSize }};
		{{ if ne $sslConfig.DHParam "" }}ssl_dhparam /opt/router/ssl/dhparam.pem;{{ end }}
		{{ end }}

		{{ if or $routerConfig.EnforceWhitelists (or (ne (len $routerConfig.DefaultWhitelist) 0) (ne (len $appConfig.Whitelist) 0)) }}
		{{ if or (eq (len $appConfig.Whitelist) 0) (eq $routerConfig.WhitelistMode "extend") }}{{ range $whitelistEntry := $routerConfig.DefaultWhitelist }}allow {{ $whitelistEntry }};{{ end }}{{ end }}
		{{ range $whitelistEntry := $appConfig.Whitelist }}allow {{ $whitelistEntry }};{{ end }}
		deny all;
		{{ end }}

		vhost_traffic_status_filter_by_set_key {{ $appConfig.Name }} application::*;

		if ($ssl_block_early_data) {
			return 425;
		}

		{{range $location := $appConfig.Locations}}
			location {{ $location.Path }} {
				{{ if $routerConfig.RequestIDs }}
				add_header X-Request-Id $request_id always;
				add_header X-Correlation-Id $correlation_id always;
				{{end}}

				{{ if (and (ne $appConfig.ReferrerPolicy "")  (ne $appConfig.ReferrerPolicy "none")) }}add_header Referrer-Policy {{ $appConfig.ReferrerPolicy }};
				{{ else if (and (ne $routerConfig.ReferrerPolicy "") (and (ne $appConfig.ReferrerPolicy "none") (ne $routerConfig.ReferrerPolicy "none"))) }}add_header Referrer-Policy {{ $routerConfig.ReferrerPolicy }};{{ end }}

				{{ if $location.App.Maintenance }}return 503;{{ else if $location.App.Available }}
				proxy_buffering {{ if $location.App.Nginx.ProxyBuffersConfig.Enabled }}on{{ else }}off{{ end }};
				proxy_buffer_size {{ $location.App.Nginx.ProxyBuffersConfig.Size }};
				proxy_buffers {{ $location.App.Nginx.ProxyBuffersConfig.Number }} {{ $location.App.Nginx.ProxyBuffersConfig.Size }};
				proxy_busy_buffers_size {{ $location.App.Nginx.ProxyBuffersConfig.BusySize }};
				proxy_set_header Host $host;
				proxy_set_header X-Forwarded-For $remote_addr;
				proxy_set_header X-Forwarded-Proto $access_scheme;
				proxy_set_header X-Forwarded-Port $forwarded_port;
				proxy_redirect off;
				proxy_connect_timeout {{ $location.App.ConnectTimeout }};
				proxy_send_timeout {{ $location.App.TCPTimeout }};
				proxy_read_timeout {{ $location.App.TCPTimeout }};
				proxy_http_version 1.1;
				proxy_set_header Upgrade $http_upgrade;
				proxy_set_header Connection $connection_upgrade;
				{{ if ne $sslConfig.EarlyDataMethods "" }}proxy_set_header Early-Data $ssl_early_data;{{ end }}
				{{ if $routerConfig.RequestIDs }}
				proxy_set_header X-Request-Id $request_id;
				proxy_set_header X-Correlation-Id $correlation_id;
				{{ end }}
				{{ if and $routerConfig.RequestStartHeader (not $appConfig.DisableRequestStartHeader) }}
				proxy_set_header X-Request-Start "t=${msec}";
				{{ end }}

				{{ if or $enforceSecure $location.App.SSLConfig.Enforce }}if ($access_scheme !~* "^https|wss$") {
					return 301 $uri_scheme://$host$request_uri;
				}{{ end }}

				{{ if $hstsConfig.Enabled }}add_header Strict-Transport-Security $sts always;{{ end }}

				proxy_pass http://{{$location.App.ServiceIP}}:80;{{ else }}return 503;{{ end }}
			}
		{{end}}

		{{ if $appConfig.Maintenance }}error_page 503 @maintenance;
			location @maintenance {
					root /;
			    rewrite ^(.*)$ /www/maintenance.html break;
			}
		{{ end }}
	}

	{{end}}{{end}}
}

{{ if $routerConfig.BuilderConfig }}{{ $builderConfig := $routerConfig.BuilderConfig }}stream {
	server {
		listen 2222 {{ if $routerConfig.UseProxyProtocol }}proxy_protocol{{ end }};
		proxy_connect_timeout {{ $builderConfig.ConnectTimeout }};
		proxy_timeout {{ $builderConfig.TCPTimeout }};
		proxy_pass {{$builderConfig.ServiceIP}}:2222;
	}
}{{ end }}
`
)

// WriteCerts writes SSL certs to file from router configuration.
func WriteCerts(routerConfig *model.RouterConfig, sslPath string) error {
	// Start by deleting all certs and their corresponding keys. This will ensure certs we no longer
	// need are deleted. Certs that are still needed will simply be re-written.
	allCertsGlob, err := filepath.Glob(filepath.Join(sslPath, "*.crt"))
	if err != nil {
		return err
	}
	allKeysGlob, err := filepath.Glob(filepath.Join(sslPath, "*.key"))
	if err != nil {
		return err
	}
	for _, cert := range allCertsGlob {
		if err := os.Remove(cert); err != nil {
			return err
		}
	}
	for _, key := range allKeysGlob {
		if err := os.Remove(key); err != nil {
			return err
		}
	}
	if routerConfig.PlatformCertificate != nil {
		err = writeCert("platform", routerConfig.PlatformCertificate, sslPath)
		if err != nil {
			return err
		}
	}
	for _, appConfig := range routerConfig.AppConfigs {
		for domain, certificate := range appConfig.Certificates {
			if certificate != nil {
				err = writeCert(domain, certificate, sslPath)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeCert(context string, certificate *model.Certificate, sslPath string) error {
	certPath := filepath.Join(sslPath, fmt.Sprintf("%s.crt", context))
	keyPath := filepath.Join(sslPath, fmt.Sprintf("%s.key", context))
	err := ioutil.WriteFile(certPath, []byte(certificate.Cert), 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(keyPath, []byte(certificate.Key), 0600)
}

// WriteDHParam writes router DHParam to file from router configuration.
func WriteDHParam(routerConfig *model.RouterConfig, sslPath string) error {
	dhParamPath := filepath.Join(sslPath, "dhparam.pem")
	if routerConfig.SSLConfig.DHParam == "" {
		err := os.RemoveAll(dhParamPath)
		if err != nil {
			return err
		}
	} else {
		err := ioutil.WriteFile(dhParamPath, []byte(routerConfig.SSLConfig.DHParam), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteConfig dynamically produces valid nginx configuration by combining a Router configuration
// object with a data-driven template.
func WriteConfig(routerConfig *model.RouterConfig, filePath string) error {
	tmpl, err := template.New("nginx").Funcs(sprig.TxtFuncMap()).Parse(confTemplate)
	if err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = tmpl.Execute(file, routerConfig)
	return err
}
