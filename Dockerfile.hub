# dockerfile for building on docker hub. Get's latest binary from s3
FROM gcr.io/distroless/static
WORKDIR /
ADD https://s3.amazonaws.com/burner-kiwi-us-east-1/burnerkiwi /
EXPOSE 8080
ENTRYPOINT [ "/burnerkiwi" ]