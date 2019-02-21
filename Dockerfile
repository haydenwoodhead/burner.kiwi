FROM gcr.io/distroless/static
WORKDIR /
COPY burnerkiwi /
EXPOSE 8080
ENTRYPOINT [ "/burnerkiwi" ]