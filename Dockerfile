FROM alpine:3.7 as alpine
RUN apk add -U --no-cache ca-certificates

FROM scratch
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY burnerkiwi /
EXPOSE 8080
ENTRYPOINT [ "/burnerkiwi" ]