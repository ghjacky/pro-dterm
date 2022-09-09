FROM golang:1.18-alpine AS build
WORKDIR /code
COPY . ./
RUN go build -o dterm *.go

FROM alpine
RUN apk add --no-cache tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=build /code/dterm ./
COPY --from=build /code/configs ./configs
RUN echo -e '[main] \n\
listen = ":8080"\n\
data_dir = "./"\n\
\n\
[log]\n\
path = ""\n\
level = "trace"\n\
max_size = 100\n\
max_age = 2\n\
max_backups = 10\n\
\n\
[easy]\n\
schema = "https"\n\
domain = "easyops.songguo7.com"\n\
api_check_token = "/api_check_token"\n\
\n\
[mysql]\n\
host = "songguo-sre-prepub-apm-mysqlredis-uhost.uc-bj2b.songguo7.com"\n\
port = 3306\n\
database = "altsubv3"\n\
user = "root"\n\
password = "Db_123#@!"' > ./configs/config.toml
ENTRYPOINT ["./dterm"]
EXPOSE 8080

