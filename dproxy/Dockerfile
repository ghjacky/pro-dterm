FROM uhub.service.ucloud.cn/sgonline/centos_new:6.8
FROM uhub.service.ucloud.cn/sgdevops/base_image:v0.2
WORKDIR /tmp
RUN yum install -y socat
ENTRYPOINT ["socat", "-d", "-d", "TCP-LISTEN:8001,fork,bind=0.0.0.0", "UNIX:/var/run/docker.sock"]
EXPOSE 8001