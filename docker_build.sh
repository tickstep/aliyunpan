#!/bin/sh

# how to use
# for macOS & linux, run this command in shell
# ./docker_build.sh v0.0.1

name="aliyunpan"
version=$1
docker_image_name=tickstep/aliyunpan-webdav

if [ "$1" = "" ]; then
  version=v1.0.0
fi

# build amd64 docker
echo "building amd64 docker image"
ARCH=amd64
ZIP_FILE_NAME=aliyunpan-$version-linux-$ARCH
# use alpine:3.15.0 as base image
# browse the url to view more information: https://hub.docker.com/layers/alpine/library/alpine/3.15.0/images/sha256-c74f1b1166784193ea6c8f9440263b9be6cae07dfe35e32a5df7a31358ac2060?context=explore
DOCKER_IMAGE_HASH=e7d88de73db3d3fd9b2d63aa7f447a10fd0220b7cbf39803c803f2af9ba256b3
unzip -d ./out ./out/$ZIP_FILE_NAME.zip
mv ./out/$ZIP_FILE_NAME ./out/binary_files

docker build \
-f ./docker/Dockerfile \
-t $docker_image_name:$version-$ARCH \
--build-arg DOCKER_IMAGE_HASH=$DOCKER_IMAGE_HASH \
--no-cache .

rm -rf out/binary_files

# build i386 docker
echo "building i386 docker image"
ARCH=386
ZIP_FILE_NAME=aliyunpan-$version-linux-$ARCH
DOCKER_IMAGE_HASH=2689e157117d2da668ad4699549e55eba1ceb79cb7862368b30919f0488213f4
unzip -d ./out ./out/$ZIP_FILE_NAME.zip
mv ./out/$ZIP_FILE_NAME ./out/binary_files

docker build \
-f ./docker/Dockerfile \
-t $docker_image_name:$version-$ARCH \
--build-arg DOCKER_IMAGE_HASH=$DOCKER_IMAGE_HASH \
--no-cache .

rm -rf out/binary_files

# build arm64 docker
echo "building arm64 docker image"
ARCH=arm64
ZIP_FILE_NAME=aliyunpan-$version-linux-$ARCH
DOCKER_IMAGE_HASH=c74f1b1166784193ea6c8f9440263b9be6cae07dfe35e32a5df7a31358ac2060
unzip -d ./out ./out/$ZIP_FILE_NAME.zip
mv ./out/$ZIP_FILE_NAME ./out/binary_files

docker build \
-f ./docker/Dockerfile \
-t $docker_image_name:$version-$ARCH \
--build-arg DOCKER_IMAGE_HASH=$DOCKER_IMAGE_HASH \
--no-cache .

rm -rf out/binary_files

# build armv7 docker
echo "building armv7 docker image"
ARCH=armv7
ZIP_FILE_NAME=aliyunpan-$version-linux-$ARCH
DOCKER_IMAGE_HASH=8483ecd016885d8dba70426fda133c30466f661bb041490d525658f1aac73822
unzip -d ./out ./out/$ZIP_FILE_NAME.zip
mv ./out/$ZIP_FILE_NAME ./out/binary_files

docker build \
-f ./docker/Dockerfile \
-t $docker_image_name:$version-$ARCH \
--build-arg DOCKER_IMAGE_HASH=$DOCKER_IMAGE_HASH \
--no-cache .

rm -rf out/binary_files

# build armv5 docker
echo "building armv5 docker image"
ARCH=armv5
ZIP_FILE_NAME=aliyunpan-$version-linux-$ARCH
DOCKER_IMAGE_HASH=e047bc2af17934d38c5a7fa9f46d443f1de3a7675546402592ef805cfa929f9d
unzip -d ./out ./out/$ZIP_FILE_NAME.zip
mv ./out/$ZIP_FILE_NAME ./out/binary_files

docker build \
-f ./docker/Dockerfile \
-t $docker_image_name:$version-$ARCH \
--build-arg DOCKER_IMAGE_HASH=$DOCKER_IMAGE_HASH \
--no-cache .

rm -rf out/binary_files

echo "push docker images"
docker push $docker_image_name:$version-amd64
docker push $docker_image_name:$version-386
docker push $docker_image_name:$version-arm64
docker push $docker_image_name:$version-armv7
docker push $docker_image_name:$version-armv5

echo "create docker manifest"
docker manifest create $docker_image_name:$version \
$docker_image_name:$version-amd64 \
$docker_image_name:$version-386 \
$docker_image_name:$version-arm64 \
$docker_image_name:$version-armv7 \
$docker_image_name:$version-armv5 \
--amend
docker manifest create $docker_image_name:latest \
$docker_image_name:$version-amd64 \
$docker_image_name:$version-386 \
$docker_image_name:$version-arm64 \
$docker_image_name:$version-armv7 \
$docker_image_name:$version-armv5 \
--amend


echo "annotate manifest for amd64 image"
docker manifest annotate \
--arch=amd64 \
--os=linux \
$docker_image_name:$version \
$docker_image_name:$version-amd64
docker manifest annotate \
--arch=amd64 \
--os=linux \
$docker_image_name:latest \
$docker_image_name:$version-amd64

echo "annotate manifest for 386 image"
docker manifest annotate \
--arch=386 \
--os=linux \
$docker_image_name:$version \
$docker_image_name:$version-386
docker manifest annotate \
--arch=386 \
--os=linux \
$docker_image_name:latest \
$docker_image_name:$version-386

echo "annotate manifest for arm64 image"
docker manifest annotate \
--arch=arm64 \
--os=linux  \
--variant=v8 \
$docker_image_name:$version \
$docker_image_name:$version-arm64
docker manifest annotate \
--arch=arm64 \
--os=linux  \
--variant=v8 \
$docker_image_name:latest \
$docker_image_name:$version-arm64


echo "annotate manifest for armv7 image"
docker manifest annotate \
--arch=arm \
--os=linux \
--variant=v7 \
$docker_image_name:$version \
$docker_image_name:$version-armv7
docker manifest annotate \
--arch=arm \
--os=linux \
--variant=v7 \
$docker_image_name:latest \
$docker_image_name:$version-armv7

echo "annotate manifest for armv5 image"
docker manifest annotate \
--arch=arm \
--os=linux \
--variant=v6 \
$docker_image_name:$version \
$docker_image_name:$version-armv5
docker manifest annotate \
--arch=arm \
--os=linux \
--variant=v6 \
$docker_image_name:latest \
$docker_image_name:$version-armv5

echo "push manifest to docker hub"
docker manifest push $docker_image_name:$version
docker manifest push $docker_image_name:latest

echo "clear local docker image"
docker rmi $docker_image_name:$version-amd64
docker rmi $docker_image_name:$version-386
docker rmi $docker_image_name:$version-arm64
docker rmi $docker_image_name:$version-armv7
docker rmi $docker_image_name:$version-armv5

echo "ALL DONE"