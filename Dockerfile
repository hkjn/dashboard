FROM hkjn/golang:1.18.1

RUN mkdir -p /home/go/src/hkjn.me/dashboard/gen

WORKDIR /home/go/src/hkjn.me/dashboard

COPY ["*.go", "./"]
COPY ["*.sh", "./"]
COPY ["*.yaml", "./"]
COPY ["cmd/", "./cmd/"]
COPY ["tmpl/", "./tmpl/"]
COPY ["Makefile", "./"]
COPY ["go.*", "./"]
COPY ["VERSION", "./"]

RUN make

WORKDIR /home/go/bin
COPY *.yaml ./
COPY tmpl/ ./tmpl/
RUN mv -iv /home/go/src/hkjn.me/dashboard/gomon .

ENTRYPOINT ["gomon"]
CMD ["-alsologtostderr"]
