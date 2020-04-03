FROM gcr.io/distroless/static
WORKDIR /
COPY burnerkiwi /
EXPOSE 8080 25
ENTRYPOINT [ "/burnerkiwi" ]