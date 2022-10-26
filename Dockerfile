FROM golang:1.18

ENV GOPROXY https://goproxy.cn,direct


WORKDIR $GOPATH/src/faucet-node
COPY . $GOPATH/src/faucet-node

RUN go get -u github.com/gobuffalo/packr/packr \
    && make install \
    && faucet init

EXPOSE 8080

CMD ["faucet","start"]