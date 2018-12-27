FROM quay.io/deis/base:v0.3.6

RUN adduser --system \
	--shell /bin/bash \
	--disabled-password \
	--home /opt/router \
	--group \
	router

COPY /bin /bin

RUN buildDeps='gcc make apt-utils libgeoip-dev libssl-dev libpcre3-dev'; \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        $buildDeps \
        libgeoip1 && \
    export NGINX_VERSION=1.13.7 SIGNING_KEY=A1C052F8 VTS_VERSION=0.1.10 BUILD_PATH=/tmp/build PREFIX=/opt/router && \
    rm -rf "$PREFIX" && \
    mkdir "$PREFIX" && \
    mkdir "$BUILD_PATH" && \
    cd "$BUILD_PATH" && \
    get_src_gpg $SIGNING_KEY "http://nginx.org/download/nginx-$NGINX_VERSION.tar.gz" && \
    get_src c6f3733e9ff84bfcdc6bfb07e1baf59e72c4e272f06964dd0ed3a1bdc93fa0ca "https://github.com/vozlt/nginx-module-vts/archive/v$VTS_VERSION.tar.gz" && \
    cd "$BUILD_PATH/nginx-$NGINX_VERSION" && \
    ./configure \
      --prefix="$PREFIX" \
      --pid-path=/tmp/nginx.pid \
      --with-debug \
      --with-pcre-jit \
      --with-ipv6 \
      --with-http_ssl_module \
      --with-http_stub_status_module \
      --with-http_realip_module \
      --with-http_auth_request_module \
      --with-http_addition_module \
      --with-http_dav_module \
      --with-http_geoip_module \
      --with-http_gzip_static_module \
      --with-http_sub_module \
      --with-http_v2_module \
      --with-mail \
      --with-mail_ssl_module \
      --with-stream \
      --add-module="$BUILD_PATH/nginx-module-vts-$VTS_VERSION" && \
    make && \
    make install && \
    rm -rf "$BUILD_PATH" && \
    # include tcell dynamic nginx module
    mkdir "$PREFIX/modules" && \
    cd "$PREFIX/modules" && \
    get_src 8f30a4d5f4a65e1a94c367b98cfec33e727453a7e7ffc7e85094a0e7a561f72d "https://s3.amazonaws.com/hephy-artifacts/hephy-router/nginx_tcellagent-1.1.0-agentonly-zuora-linux-x86_64.tar.gz" && \
    mv "$PREFIX/modules/nginx_tcellagent-1.1.0-agentonly-zuora-linux-x86_64/ubuntu/xenial/nginx-1.13.7-custom_flags_ssl_1.0.2g/ngx_http_tcell_agent_module.so" . && \
    rm -rf "$PREFIX/modules/nginx_tcellagent-1.1.0-agentonly-zuora-linux-x86_64" && \
    # include libmodsecurity3 and modsecurity connector dynamic module
    modsecurityDeps='apt-utils git libcurl4-openssl-dev libyajl-dev libxml2 libxml2-dev' && \
    apt-get install -y --no-install-recommends \
        $modsecurityDeps && \
    cd "$PREFIX/modules" && \
    get_src 136e0faf4b313817abd07365935ebd9174e8754700fe8a06281dbcbbe6d0ad50 "https://s3.amazonaws.com/hephy-artifacts/hephy-router/modsecurity-v3.0.3-ubuntu-16-04.tar.gz" && \
    mv usr/local/modsecurity /usr/local/modsecurity && \
    rm -rf usr && \
    get_src_file c9fd4ddb69ba1ce0a3118e529c43f87c3ab216e20900e25863e58537399d2d19 "https://s3.amazonaws.com/hephy-artifacts/hephy-router/ngx_http_modsecurity_module.so" && \
    # setup the modsecurity config and OWASP rules
    cd "$PREFIX/conf" && \
    get_src_file 5614fd0f68fc7707c0dc008d45b92de586b6e14937a41b93e80165aec454eecd "https://s3.amazonaws.com/hephy-artifacts/hephy-router/modsecurity.conf" && \
    curl -sSL https://github.com/SpiderLabs/ModSecurity/raw/v3/master/unicode.mapping -o unicode.mapping && \
    git clone https://github.com/SpiderLabs/owasp-modsecurity-crs.git && \
    cp -R owasp-modsecurity-crs/rules/ $PREFIX/conf/ && \
    cp $PREFIX/conf/owasp-modsecurity-crs/crs-setup.conf.example $PREFIX/conf/crs-setup.conf && \
    rm -rf owasp-modsecurity-crs && \
    # cleanup
    apt-get purge -y --auto-remove $buildDeps && \
    apt-get autoremove -y && \
    apt-get clean -y && \
    # package up license files if any by appending to existing tar
    COPYRIGHT_TAR='/usr/share/copyrights.tar'; \
    gunzip -f $COPYRIGHT_TAR.gz; tar -rf $COPYRIGHT_TAR /usr/share/doc/*/copyright; gzip $COPYRIGHT_TAR && \
    rm -rf \
        /usr/share/doc \
        /usr/share/man \
        /usr/share/info \
        /usr/share/locale \
        /var/lib/apt/lists/* \
        /var/log/* \
        /var/cache/debconf/* \
        /etc/systemd \
        /lib/lsb \
        /lib/udev \
        /usr/lib/x86_64-linux-gnu/gconv/IBM* \
        /usr/lib/x86_64-linux-gnu/gconv/EBC* && \
    bash -c "mkdir -p /usr/share/man/man{1..8}"

COPY . /

# Fix some permissions since we'll be running as a non-root user
RUN chown -R router:router /opt/router /var/log

USER router

CMD ["/opt/router/sbin/boot"]
EXPOSE 2222 8080 6443 9090
