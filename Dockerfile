FROM golang:alpine as builder
WORKDIR /build
ADD . .
RUN CGO_ENABLED=0 go build -ldflags='-w -s -extldflags "-static"' -a -o /build/gossm .

FROM scratch 
COPY --from=builder /build/ /go/bin/
CMD [ "/go/bin/gossm" ]