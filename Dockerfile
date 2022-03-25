FROM ubuntu:18.04
ARG TARGETARCH
COPY burnerkiwi.${TARGETARCH} burnerkiwi
EXPOSE 8080 25
ENTRYPOINT [ "/burnerkiwi" ]