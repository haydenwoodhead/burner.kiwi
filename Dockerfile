FROM alpine:3
ARG TARGETARCH
COPY burnerkiwi.${TARGETARCH} burnerkiwi
EXPOSE 8080 25
ENTRYPOINT [ "/burnerkiwi" ]