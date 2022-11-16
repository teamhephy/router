FROM hephy/base:v0.5.0 as modsecurity

COPY /bin /bin
WORKDIR /tmp/build

ENV DEBIAN_FRONTEND=noninteractive

RUN set -x && \
    apt-get autoremove -y && \
    apt-get clean -y && \
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
    export MOD_SECURITY_VERSION=3.0.5 BUILD_PATH=$PWD PREFIX=/usr/local && \
    get_src 751bf95a7a8d39c440d0c26ec1f73961550ca2eb2ac9e2e7a56dce2dd7b959e9 \
            "https://github.com/SpiderLabs/ModSecurity/releases/download/v$MOD_SECURITY_VERSION/modsecurity-v$MOD_SECURITY_VERSION.tar.gz" && \
    cd "$BUILD_PATH/modsecurity-v$MOD_SECURITY_VERSION" && \
    ldconfig && \
    ./configure \
      --prefix="$PREFIX" \
      --enable-silent-rules \
      --disable-doxygen-doc \
      --disable-examples \
      --disable-dependency-tracking && \
    make -j`nproc` && \
    make install-strip && \
    install -D -m 644 -t "$PREFIX/share/modsecurity" \
        unicode.mapping \
        modsecurity.conf-recommended && \
    # cleanup
    apt-get purge -y --auto-remove $buildDeps && \
    apt-get autoremove -y && \
    apt-get clean -y

FROM hephy/base:v0.5.0 as openssl

COPY /bin /bin
WORKDIR /tmp/build

RUN set -x && \
    apt-get update && \
    ln -fs /usr/share/zoneinfo/Etc/UTC /etc/localtime && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        build-essential fakeroot devscripts equivs

RUN set -x && \
    export OPENSSL_VERSION=1.1.1n OPENSSL_VERSION_DEB=1.1.1n-0+deb11u3 BUILD_PATH=$PWD DEBEMAIL="Team Hephy <team@teamhephy.com>" && \
    get_src_file cbf06993d1dc8c4e59e2f7f0456cc07a005c17274a33eb29135e9fdd7f6b2941 \
                 http://deb.debian.org/debian/pool/main/o/openssl/openssl_$OPENSSL_VERSION_DEB.dsc && \
    get_src_file 40dceb51a4f6a5275bde0e6bf20ef4b91bfc32ed57c0552e2e8e15463372b17a \
                 http://deb.debian.org/debian/pool/main/o/openssl/openssl_$OPENSSL_VERSION.orig.tar.gz && \
    get_src_file e0e89e9467102880ee6f2ee8c1413933eb1268969afb97b9bec61e2190a62fd0 \
                 http://deb.debian.org/debian/pool/main/o/openssl/openssl_$OPENSSL_VERSION.orig.tar.gz.asc && \
    get_src_file 0eed2dc1f8d472291511f386fc0a3c5fe70ec50440b1c252fd99934bf2cddd4a \
                 http://deb.debian.org/debian/pool/main/o/openssl/openssl_$OPENSSL_VERSION_DEB.debian.tar.xz && \
    dpkg-source -x openssl_$OPENSSL_VERSION_DEB.dsc && \
    cd "$BUILD_PATH/openssl-$OPENSSL_VERSION" && \
    sed -i '/^Build-Depends:/ s/ debhelper-compat (= 12)/ debhelper-compat (= 11)/' debian/control && \
    dch -l hephy "Fix build deps for Ubuntu Bionic" && \
    mk-build-deps -i -t "apt-get -y --no-install-recommends" && \
    sed -i '/^CONFARGS\s*=/ s/ enable-unit-test//' debian/rules && \
    dch -l hephy "Disable unit tests" && \
    dch -r " " && \
    DEB_BUILD_OPTIONS=nocheck dpkg-buildpackage -uc -b -rfakeroot && \
    cd .. && \
    dpkg-scanpackages . > Packages && \
    mkdir ../repo && \
    mv Packages *.deb ../repo && \
    # cleanup
    apt-get autoremove -y && \
    apt-get clean -y

FROM hephy/base:v0.5.0

RUN adduser --system \
	--shell /bin/bash \
	--disabled-password \
	--home /opt/router \
	--group \
	router

COPY --from=modsecurity /usr/local /usr/local
COPY --from=openssl /tmp/repo /usr/local/repo

COPY /bin /bin

RUN set -x && \
    buildDeps='gcc make patch libgeoip-dev libmaxminddb-dev libpcre3-dev libssl-dev' \
    runtimeDeps='ca-certificates libcurl4 libxml2 libpcre3 libgeoip1 libmaxminddb0 libfuzzy2 libssl1.1 openssl libyajl2' && \
    echo 'deb [trusted=yes] file:/usr/local/repo ./' >> /etc/apt/sources.list && \
    apt-get update && \
    ln -fs /usr/share/zoneinfo/Etc/UTC /etc/localtime && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        $buildDeps \
        $runtimeDeps && \
    export NGINX_VERSION=1.22.1 SIGNING_KEY=13C82A63B603576156E30A4EA0EA981B66B0D967 \
           CLOUDFLARE_ZLIB_VERSION=372bcd151c901418c2721232bf09dc9cdbebafb5 \
           VTS_VERSION=0.1.18 GEOIP2_VERSION=3.3 \
           MOD_SECURITY_NGINX_VERSION=1.0.3 \
           OWASP_MOD_SECURITY_CRS_VERSION=3.3.2 \
           BUILD_PATH=/tmp/build PREFIX=/opt/router && \
    ldconfig && \
    rm -rf "$PREFIX" && \
    mkdir "$PREFIX" && \
    mkdir "$BUILD_PATH" && \
    cd "$BUILD_PATH" && \
    get_src_gpg $SIGNING_KEY "http://nginx.org/download/nginx-$NGINX_VERSION.tar.gz" && \
    # Always enable PrioritizeChaCha in NGINX (prefer ChaCha-Poly for mobile clients w/o AES support)
    get_src_file 95e772ca01c7201ccc2a1b188181b06a497ba89a9076a9b28b32b5814c9e1654 \
            "https://raw.githubusercontent.com/kn007/patch/c59592b/nginx_auto_using_PRIORITIZE_CHACHA.patch" && \
    get_src 6cd7f933230f657a4a68a1c258f6718d4ed7937725afe2c83a94b71f7a6b5d0f \
            "https://github.com/cloudflare/zlib/archive/$CLOUDFLARE_ZLIB_VERSION.tar.gz" && \
    get_src 17ea41d4083f6d1ab1ab83dad9160eeca66867abe16c5a0421f85a39d7c84b65 \
            "https://github.com/vozlt/nginx-module-vts/archive/v$VTS_VERSION.tar.gz" && \
    get_src 41378438c833e313a18869d0c4a72704b4835c30acaf7fd68013ab6732ff78a7 \
            "https://github.com/leev/ngx_http_geoip2_module/archive/$GEOIP2_VERSION.tar.gz" && \
    get_src 32a42256616cc674dca24c8654397390adff15b888b77eb74e0687f023c8751b \
            "https://github.com/SpiderLabs/ModSecurity-nginx/archive/v$MOD_SECURITY_NGINX_VERSION.tar.gz" && \
    cd "$BUILD_PATH/zlib-$CLOUDFLARE_ZLIB_VERSION" && \
    make -f Makefile.in distclean && \
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
      --with-zlib="$BUILD_PATH/zlib-$CLOUDFLARE_ZLIB_VERSION" \
      --add-module="$BUILD_PATH/nginx-module-vts-$VTS_VERSION" \
      --add-dynamic-module="$BUILD_PATH/ngx_http_geoip2_module-$GEOIP2_VERSION" \
      --add-dynamic-module="$BUILD_PATH/ModSecurity-nginx-$MOD_SECURITY_NGINX_VERSION" && \
    make -j`nproc` && \
    make install && \
    strip -s "$PREFIX/sbin/nginx" "$PREFIX/modules/"*.so && \
    cd "$BUILD_PATH" && \
    # setup the modsecurity config and OWASP rules
    get_src 56ddb33a5d0413f43157e1ea22415adf75d8b12219c43156861fd11708c8033e \
            "https://github.com/coreruleset/coreruleset/archive/refs/tags/v$OWASP_MOD_SECURITY_CRS_VERSION.tar.gz" && \
    cp -R coreruleset-$OWASP_MOD_SECURITY_CRS_VERSION/rules $PREFIX/conf/ && \
    cp coreruleset-$OWASP_MOD_SECURITY_CRS_VERSION/crs-setup.conf.example $PREFIX/conf/crs-setup.conf && \
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
        /usr/lib/x86_64-linux-gnu/gconv/EBC* && \
    # Fix some permissions since we'll be running as a non-root user
    chown -R router:router /opt/router /var/log

COPY bin/. /bin/.
COPY --chown=router:router opt/. /opt/.
COPY www/. /www/.

USER router

CMD ["/opt/router/sbin/boot"]
EXPOSE 2222 8080 6443 9090
