FROM debian:11-slim
WORKDIR /app
COPY fluffy-linux-amd64 ./fluffy
CMD ["./fluffy"]