#!/bin/sh

# how to use
# for macOS & linux, run this command in shell
# ./build.sh v0.1.0

name="aliyunpan"
version=$1

if [ "$1" = "" ]; then
  version=v1.0.0
fi

output="out"

build_dir=`dirname $0`
if [ ! -d ${build_dir}/${output} ];then
  mkdir -p ${build_dir}/${output}
fi

default_golang() {
  export GOROOT=/usr/local/go
  go=$GOROOT/bin/go
}

Build() {
  default_golang
  goarm=$4
  if [ "$4" = "" ]; then
    goarm=7
  fi

  echo "Building $1..."
  export GOOS=$2 GOARCH=$3 GO386=sse2 CGO_ENABLED=0 GOARM=$4
  if [ $2 = "windows" ]; then
    goversioninfo -o=resource_windows_386.syso
    goversioninfo -64 -o=resource_windows_amd64.syso
    $go build -ldflags "-X main.Version=$version -s -w" -o "$output/$1/$name.exe"
    RicePack $1 $name.exe
  else
    $go build -ldflags "-X main.Version=$version -s -w" -o "$output/$1/$name"
    RicePack $1 $name
  fi

  Pack $1 $2
}

AndroidBuild() {
  default_golang
  echo "Building $1..."
  export GOOS=$2 GOARCH=$3 GOARM=$4 CGO_ENABLED=1
  $go build -ldflags "-X main.Version=$version -s -w -linkmode=external -extldflags=-pie" -o "$output/$1/$name"

  RicePack $1 $name
  Pack $1 $2
}

IOSBuild() {
  default_golang
  echo "Building $1..."
  mkdir -p "$output/$1"
  cd "$output/$1"
  export CC=/usr/local/go/misc/ios/clangwrap.sh GOOS=ios GOARCH=arm64 GOARM=7 CGO_ENABLED=1
  $go build -ldflags "-X main.Version=$version -s -w" -o $name github.com/tickstep/aliyunpan
  jtool2 --sign --inplace --ent ../../entitlements.xml $name
  cd ../..
  RicePack $1 $name
  Pack $1 "ios"
}

# zip 打包
Pack() {
  if [ $2 != "windows" ]; then
      chmod +x "$output/$1/$name"
  fi

  cp README.md "$output/$1"
  cp docs/manual.md "$output/$1"
  cp docs/plugin_manual.md "$output/$1"
  cp -rf assets/scripts/* "$output/$1"
  cp -rf assets/plugin "$output/$1"
  cp -rf assets/sync_drive "$output/$1"

  cd $output
  zip -q -r "$1.zip" "$1"

  # 删除
  rm -rf "$1"

  cd ..
}

# rice 打包静态资源
RicePack() {
  return # 已取消web功能
}

############### Android ###############
export ANDROID_NDK_ROOT=/Users/tickstep/Applications/android_ndk/android-ndk-r23-darwin
CC=$ANDROID_NDK_ROOT/bin/arm-linux-androideabi/bin/clang AndroidBuild $name-$version"-android-api16-armv7" android arm 7
CC=$ANDROID_NDK_ROOT/bin/aarch64-linux-android/bin/clang AndroidBuild $name-$version"-android-api21-arm64" android arm64 7
CC=$ANDROID_NDK_ROOT/bin/i686-linux-android/bin/clang    AndroidBuild $name-$version"-android-api16-386" android 386 7
CC=$ANDROID_NDK_ROOT/bin/x86_64-linux-android/bin/clang  AndroidBuild $name-$version"-android-api21-amd64" android amd64 7

############### iOS ###############
IOSBuild $name-$version"-ios-arm64"

############### macOS ###############
Build $name-$version"-darwin-macos-amd64" darwin amd64
# Build $name-$version"-darwin-macos-386" darwin 386
Build $name-$version"-darwin-macos-arm64" darwin arm64

############### Windows ###############
Build $name-$version"-windows-x86" windows 386
Build $name-$version"-windows-x64" windows amd64
Build $name-$version"-windows-arm64" windows arm64

############### Linux ###############
# x64/x86
Build $name-$version"-linux-386" linux 386
Build $name-$version"-linux-amd64" linux amd64

# 龙芯 LoongArch
Build $name-$version"-linux-loong64" linux loong64

# arm
Build $name-$version"-linux-armv5" linux arm 5
Build $name-$version"-linux-armv7" linux arm 7
Build $name-$version"-linux-arm64" linux arm64

# mips
GOMIPS=softfloat Build $name-$version"-linux-mips64" linux mips64
GOMIPS=softfloat Build $name-$version"-linux-mips64le" linux mips64le
GOMIPS=hardfloat Build $name-$version"-linux-mips64hf" linux mips64
GOMIPS=hardfloat Build $name-$version"-linux-mips64lehf" linux mips64le
GOMIPS=softfloat Build $name-$version"-linux-mips" linux mips
GOMIPS=softfloat Build $name-$version"-linux-mipsle" linux mipsle
GOMIPS=hardfloat Build $name-$version"-linux-mipshf" linux mips
GOMIPS=hardfloat Build $name-$version"-linux-mipslehf" linux mipsle

# freebsd
Build $name-$version"-freebsd-386" freebsd 386
Build $name-$version"-freebsd-amd64" freebsd amd64
# Build $name-$version"-freebsd-arm" freebsd arm

# Others
# Build $name-$version"-linux-ppc64" linux ppc64
# Build $name-$version"-linux-ppc64le" linux ppc64le
# Build $name-$version"-linux-s390x" linux s390x
# Build $name-$version"-solaris-amd64" solaris amd64
# Build $name-$version"-netbsd-386" netbsd	386
# Build $name-$version"-netbsd-amd64" netbsd amd64
# Build $name-$version"-netbsd-arm" netbsd	arm
# Build $name-$version"-openbsd-386" openbsd 386
# Build $name-$version"-openbsd-amd64" openbsd	amd64
# Build $name-$version"-openbsd-arm" openbsd arm
# Build $name-$version"-plan9-386" plan9 386
# Build $name-$version"-plan9-amd64" plan9 amd64
# Build $name-$version"-plan9-arm" plan9 arm
# Build $name-$version"-nacl-386" nacl 386
# Build $name-$version"-nacl-amd64p32" nacl amd64p32
# Build $name-$version"-nacl-arm" nacl arm
# Build $name-$version"-dragonflybsd-amd64" dragonfly amd64
