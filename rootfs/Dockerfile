FROM quay.io/deis/base:v0.3.6 as modsecurity

COPY /bin /bin
WORKDIR /tmp/build

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        g++ make \
        libcurl4-openssl-dev \
        libyajl-dev \
        liblmdb-dev \
        libxml2-dev \
        libpcre3-dev \
        libmaxminddb-dev \
        libfuzzy-dev && \
    export MOD_SECURITY_VERSION=3.0.3 BUILD_PATH=$PWD PREFIX=/usr/local && \
    get_src 8aa1300105d8cc23315a5e54421192bc617a66246ad004bd89e67c232208d0f4 \
            "https://github.com/SpiderLabs/ModSecurity/releases/download/v$MOD_SECURITY_VERSION/modsecurity-v$MOD_SECURITY_VERSION.tar.gz" && \
    cd "$BUILD_PATH/modsecurity-v$MOD_SECURITY_VERSION" && \
    ./configure \
      --prefix="$PREFIX" \
      --enable-silent-rules \
      --enable-static=no \
      --disable-doxygen-doc \
      --disable-examples \
      --disable-dependency-tracking && \
    make -j`nproc` && \
    make install-strip && \
    install -D -m 644 -t "$PREFIX/share/modsecurity" \
        unicode.mapping \
        modsecurity.conf-recommended


FROM quay.io/deis/base:v0.3.6

RUN adduser --system \
	--shell /bin/bash \
	--disabled-password \
	--home /opt/router \
	--group \
	router

COPY --from=modsecurity /usr/local /usr/local

COPY /bin /bin

RUN set -x && \
    buildDeps='gcc make apt-utils libgeoip-dev libmaxminddb-dev libssl-dev libpcre3-dev' \
    runtimeDeps='libcurl3 libxml2 libpcre3 libgeoip1 libmaxminddb0 libfuzzy2 openssl' && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        $buildDeps \
        $runtimeDeps && \
    export NGINX_VERSION=1.14.2 SIGNING_KEY=A1C052F8 \
           VTS_VERSION=0.1.18 GEOIP2_VERSION=3.2 \
           MOD_SECURITY_NGINX_VERSION=d7101e13685efd7e7c9f808871b202656a969f4b \
           OWASP_MOD_SECURITY_CRS_VERSION=46171c0ef335f92b26787ce269e397c480286155 \
           BUILD_PATH=/tmp/build PREFIX=/opt/router && \
    rm -rf "$PREFIX" && \
    mkdir "$PREFIX" && \
    mkdir "$BUILD_PATH" && \
    cd "$BUILD_PATH" && \
    get_src_gpg $SIGNING_KEY "http://nginx.org/download/nginx-$NGINX_VERSION.tar.gz" && \
    get_src 17ea41d4083f6d1ab1ab83dad9160eeca66867abe16c5a0421f85a39d7c84b65 \
            "https://github.com/vozlt/nginx-module-vts/archive/v$VTS_VERSION.tar.gz" && \
    get_src 15bd1005228cf2c869a6f09e8c41a6aaa6846e4936c473106786ae8ac860fab7 \
            "https://github.com/leev/ngx_http_geoip2_module/archive/$GEOIP2_VERSION.tar.gz" && \
    get_src 5c8d25e68fb852f61489b669aebb7bd8ca8c88ebb5e5f969212fcceff3ee2d0b \
            "https://github.com/SpiderLabs/ModSecurity-nginx/archive/$MOD_SECURITY_NGINX_VERSION.tar.gz" && \
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
      --with-http_gunzip_module \
      --with-http_sub_module \
      --with-http_v2_module \
      --with-mail \
      --with-mail_ssl_module \
      --with-stream \
      --add-module="$BUILD_PATH/nginx-module-vts-$VTS_VERSION" \
      --add-dynamic-module="$BUILD_PATH/ngx_http_geoip2_module-$GEOIP2_VERSION" \
      --add-dynamic-module="$BUILD_PATH/ModSecurity-nginx-$MOD_SECURITY_NGINX_VERSION" && \
    make -j`nproc` && \
    make install && \
    strip -s "$PREFIX/sbin/nginx" "$PREFIX/modules/"*.so && \
    cd "$BUILD_PATH" && \
    # setup the modsecurity config and OWASP rules
    get_src c0e5d496db41b9b5e201fd8138e2507d22b22cf945b7b06bf3c9fad31b0bba95 \
            "https://github.com/SpiderLabs/owasp-modsecurity-crs/archive/$OWASP_MOD_SECURITY_CRS_VERSION.tar.gz" && \
    cp -R owasp-modsecurity-crs-$OWASP_MOD_SECURITY_CRS_VERSION/rules $PREFIX/conf/ && \
    cp owasp-modsecurity-crs-$OWASP_MOD_SECURITY_CRS_VERSION/crs-setup.conf.example $PREFIX/conf/crs-setup.conf && \
    cp /usr/local/share/modsecurity/unicode.mapping "$PREFIX/conf/" && \
    sed -e 's/^SecRuleEngine DetectionOnly/SecRuleEngine On/' \
        -e '$ a # Load OWASP Core Rule Set' \
        -e '$ a Include crs-setup.conf' \
        -e '$ a Include rules/*.conf' \
        /usr/local/share/modsecurity/modsecurity.conf-recommended > "$PREFIX/conf/modsecurity.conf" && \
    cd / && \
    rm -rf "$BUILD_PATH" && \
    rm -rf /usr/local/include/* && \
    # cleanup
    apt-get purge -y --auto-remove $buildDeps && \
    apt-get autoremove -y && \
    apt-get clean -y && \
    # package up license files if any by appending to existing tar
    COPYRIGHT_TAR='/usr/share/copyrights.tar' && \
    gunzip -f $COPYRIGHT_TAR.gz && tar -rf $COPYRIGHT_TAR /usr/share/doc/*/copyright && gzip $COPYRIGHT_TAR && \
    rm -rf \
        /usr/share/doc \
        /usr/share/man/man?/* \
        /usr/share/info \
        /usr/share/locale \
        /var/lib/apt/lists/* \
        /var/log/* \
        /var/cache/debconf/* \
        /etc/systemd \
        /lib/lsb \
        /lib/udev \
        /usr/lib/x86_64-linux-gnu/gconv/IBM* \
        /usr/lib/x86_64-linux-gnu/gconv/EBC*

COPY . /

# Fix some permissions since we'll be running as a non-root user
RUN chown -R router:router /opt/router /var/log

USER router

CMD ["/opt/router/sbin/boot"]
EXPOSE 2222 8080 6443 9090
