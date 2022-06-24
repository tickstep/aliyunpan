# alpine:3.15.0

# 参数
ARG DOCKER_IMAGE_HASH

FROM alpine@sha256:$DOCKER_IMAGE_HASH
LABEL author="tickstep"
LABEL email="tickstep@outlook.com"
LABEL version="1.0"
LABEL description="sync & backup service for aliyun cloud drive"

# 时区
ENV TZ=Asia/Shanghai
# 手动下载tzdata安装包，注意要下载对应架构的： https://dl-cdn.alpinelinux.org/alpine/v3.15/community/
RUN apk add -U tzdata
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建运行目录
RUN mkdir -p /home/app
VOLUME /home/app
WORKDIR /home/app
RUN cd /home/app

# 创建配置文件目录
RUN mkdir -p /home/app/config

# 创建数据文件夹
RUN mkdir -p /home/app/data
RUN chmod 777 /home/app/data

# 复制文件
COPY ./docker/sync/app.sh /home/app/app.sh
RUN chmod +x /home/app/app.sh
COPY ./docker/sync/health_check.sh /home/app/health_check.sh
RUN chmod +x /home/app/health_check.sh

COPY ./out/binary_files/aliyunpan /home/app
RUN mkdir -p /home/app/config/plugin
COPY ./out/binary_files/plugin /home/app/config/plugin
RUN mkdir -p /home/app/config/sync_drive
COPY ./out/binary_files/sync_drive /home/app/config/sync_drive
#RUN chmod +x /home/app/aliyunpan

# 健康检查
HEALTHCHECK --start-period=5s --interval=10s --timeout=5s --retries=3 CMD /bin/sh /home/app/health_check.sh

# 环境变量
ENV ALIYUNPAN_DOCKER=1
ENV ALIYUNPAN_CONFIG_DIR=/home/app/config

ENV ALIYUNPAN_REFRESH_TOKEN=""
ENV ALIYUNPAN_TRANSFER_URL_TYPE=0

ENV ALIYUNPAN_DOWNLOAD_PARALLEL=2
ENV ALIYUNPAN_UPLOAD_PARALLEL=2
ENV ALIYUNPAN_DOWNLOAD_BLOCK_SIZE=1024
ENV ALIYUNPAN_UPLOAD_BLOCK_SIZE=10240
ENV ALIYUNPAN_LOCAL_DIR=/home/app/data
ENV ALIYUNPAN_PAN_DIR=/sync_drive
ENV ALIYUNPAN_SYNC_MODE=upload

# 运行
ENTRYPOINT ./app.sh