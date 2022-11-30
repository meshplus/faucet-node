FROM golang:1.18.2 as builder

ENV PATH=$PATH:/go/bin
WORKDIR /go/src/github.com/meshplus/faucet-node
COPY . /go/src/github.com/meshplus/faucet-node

RUN go env -w GOPROXY=https://goproxy.cn,direct \
    && go env -w GOFLAGS="--mod=mod" \
    && wget https://nexus.hyperchain.cn/repository/infra/bin/tini_amd64 -O /usr/local/bin/tini  \
    && chmod +x /usr/local/bin/tini \
    && go install github.com/gobuffalo/packr/v2/packr2@v2.8.3 \
    && make install \
    && echo $(go env) \
    && faucet init

FROM frolvlad/alpine-glibc:glibc-2.32

COPY --from=0 /go/bin/faucet /usr/local/bin/faucet
COPY --from=0 /root/.faucet /root/.faucet
COPY --from=0 /usr/local/bin/tini /usr/local/bin/tini

EXPOSE 8080
ENTRYPOINT ["tini","--"]
CMD ["faucet","start"]