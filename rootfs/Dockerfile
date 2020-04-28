FROM hephy/base:v0.3.6 as modsecurity

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
    export MOD_SECURITY_VERSION=3.0.4 BUILD_PATH=$PWD PREFIX=/usr/local && \
    get_src b4231177dd80b4e076b228e57d498670113b69d445bab86db25f65346c24db22 \
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

FROM hephy/base:v0.3.6 as openssl

COPY /bin /bin
WORKDIR /tmp/build

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        g++ make patch perl

RUN set -x && \
    export OPENSSL_VERSION=1.1.1g OPENSSL_SIGNING_KEY=0E604491 BUILD_PATH=$PWD PREFIX=/usr/local && \
    get_src_gpg $OPENSSL_SIGNING_KEY "https://www.openssl.org/source/openssl-$OPENSSL_VERSION.tar.gz" && \
    # ChaCha20-Poly1305 Draft Support for older Android versions
    get_src_file 5e082d46544915b0a580fe71a5e53cb22f535c7dc67a35221d292316701dc085 \
                 https://raw.githubusercontent.com/hakasenyang/openssl-patch/3ea9038/openssl-1.1.1f-chacha_draft.patch && \
    # Allow setting equal preference and TLSv1.3 ciphers in NGINX
    get_src_file 04f682c36405a908247c27e317fb0f5f5bb19cbac9699f5afa21fd81511e6be2 \
                 https://raw.githubusercontent.com/hakasenyang/openssl-patch/e3bd4a8/openssl-equal-1.1.1e-dev_ciphers.patch && \
    cd "$BUILD_PATH/openssl-$OPENSSL_VERSION" && \
    patch -p1 -i "$BUILD_PATH/openssl-1.1.1f-chacha_draft.patch" && \
    patch -p1 -i "$BUILD_PATH/openssl-equal-1.1.1e-dev_ciphers.patch" && \
    ./config --prefix=/usr/local \
      --openssldir=/etc/ssl \
      shared enable-weak-ssl-ciphers && \
    make -j`nproc` && \
    make install_sw

FROM hephy/base:v0.3.6

RUN adduser --system \
	--shell /bin/bash \
	--disabled-password \
	--home /opt/router \
	--group \
	router

COPY --from=modsecurity /usr/local /usr/local
COPY --from=openssl /usr/local /usr/local

COPY /bin /bin

RUN set -x && \
    buildDeps='gcc make patch libgeoip-dev libmaxminddb-dev libpcre3-dev zlib1g-dev' \
    runtimeDeps='ca-certificates libcurl3 libxml2 libpcre3 libgeoip1 libmaxminddb0 libfuzzy2' && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        $buildDeps \
        $runtimeDeps && \
    # Remove openssl binaries from base image, will be provided by our custom build
    apt-get remove -y openssl && \
    export NGINX_VERSION=1.18.0 SIGNING_KEY=A1C052F8 \
           VTS_VERSION=0.1.18 GEOIP2_VERSION=3.3 \
           MOD_SECURITY_NGINX_VERSION=e50e43ee4cc87565922ed98b1b6c72751019c326 \
           OWASP_MOD_SECURITY_CRS_VERSION=cf57fd53de06b87b90d2cc5d61d602df81b2dd70 \
           BUILD_PATH=/tmp/build PREFIX=/opt/router && \
    ldconfig && \
    # Provide custom openssl binaries in standard /usr/bin location
    update-alternatives --verbose \
      --install /usr/bin/openssl openssl /usr/local/bin/openssl 1 \
      --slave /usr/bin/c_rehash c_rehash /usr/local/bin/c_rehash && \
    rm -rf "$PREFIX" && \
    mkdir "$PREFIX" && \
    mkdir "$BUILD_PATH" && \
    cd "$BUILD_PATH" && \
    get_src_gpg $SIGNING_KEY "http://nginx.org/download/nginx-$NGINX_VERSION.tar.gz" && \
    # Always enable PrioritizeChaCha in NGINX (prefer ChaCha-Poly for mobile clients w/o AES support)
    get_src_file 95e772ca01c7201ccc2a1b188181b06a497ba89a9076a9b28b32b5814c9e1654 \
            "https://raw.githubusercontent.com/kn007/patch/c59592b/nginx_auto_using_PRIORITIZE_CHACHA.patch" && \
    get_src 17ea41d4083f6d1ab1ab83dad9160eeca66867abe16c5a0421f85a39d7c84b65 \
            "https://github.com/vozlt/nginx-module-vts/archive/v$VTS_VERSION.tar.gz" && \
    get_src 41378438c833e313a18869d0c4a72704b4835c30acaf7fd68013ab6732ff78a7 \
            "https://github.com/leev/ngx_http_geoip2_module/archive/$GEOIP2_VERSION.tar.gz" && \
    get_src a2e5a6950616ae68ba960f83d45830de9349ea06b5bb9fcf9dda0af453ec747b \
            "https://github.com/SpiderLabs/ModSecurity-nginx/archive/$MOD_SECURITY_NGINX_VERSION.tar.gz" && \
    cd "$BUILD_PATH/nginx-$NGINX_VERSION" && \
    patch -p1 -i "$BUILD_PATH/nginx_auto_using_PRIORITIZE_CHACHA.patch" && \
    ./configure \
      --prefix="$PREFIX" \
      --pid-path=/tmp/nginx.pid \
      --with-debug \
      --with-pcre-jit \
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
    get_src ab86c1dede7873f7bdc3c882f60b5b06bd0798ab555d7275b178ac6e104c909e \
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
