FROM golang

WORKDIR /go/src/hkjn.me/dashboard
COPY *.go ./
COPY cmd/ ./cmd/
RUN go get hkjn.me/dashboard/cmd/gomon

WORKDIR /go/bin
COPY *.yaml ./
COPY tmpl/ ./tmpl/

ENTRYPOINT ["gomon"]
CMD ["-alsologtostderr"]