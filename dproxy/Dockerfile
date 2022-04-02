FROM alpine
WORKDIR /app
RUN apk add socat
CMD ["socat", "-d", "-d", "TCP-LISTEN:8001,fork,bind=0.0.0.0", "UNIX:/var/run/docker.sock"]
EXPOSE 8001