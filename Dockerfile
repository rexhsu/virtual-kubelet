FROM golang:alpine as builder

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apk add --no-cache \
	ca-certificates

COPY . /go/src/github.com/virtual-kubelet/virtual-kubelet

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		git \
		gcc \
		libc-dev \
		libgcc \
        make \
	&& cd /go/src/github.com/virtual-kubelet/virtual-kubelet \
	&& make build \ 
	&& apk del .build-deps \
    && cp bin/virtual-kubelet /usr/bin/virtual-kubelet \
	&& rm -rf /go \
	&& echo "Build complete."

FROM scratch

COPY --from=builder /usr/bin/virtual-kubelet /usr/bin/virtual-kubelet
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENV OS_PROJECT_DOMAIN_ID=default OS_REGION_NAME=RegionOne OS_USER_DOMAIN_ID=default OS_PROJECT_NAME=admin OS_IDENTITY_API_VERSION=3 OS_PASSWORD=password OS_AUTH_TYPE=password OS_AUTH_URL=http://10.169.36.100/identity OS_USERNAME=admin OS_TENANT_NAME=admin OS_VOLUME_API_VERSION=2 OS_DOMAIN_ID=default

ENTRYPOINT [ "/usr/bin/virtual-kubelet" ]
CMD [ "--help" ]
