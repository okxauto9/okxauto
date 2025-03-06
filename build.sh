APP=okxauto
##############
##Author: xxxxx@xx.com
##Date: 2025-03-06 11:22:55
##LastEditors: xxxxx@xx.com
##LastEditTime: 2025-03-06 11:42:02
##FilePath: \aaa8\build.sh
##Description: 
##
##Copyright (c) 2022 by xxxxx@xx.com, All Rights Reserved. 
##############
LDFLAGS="-w -s"
GIT_VERSION="v1.0"

export CC=gcc
# Native build
CGO_ENABLED=1 go build -ldflags "-w -s" -o ${APP} main.go
export CC=gcc
# Linux amd64
echo "Linux amd64 building version ${GIT_VERSION}"
CC=gcc CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" -o ${APP}-amd64 main.go
export CC=aarch64-linux-gnu-gcc
# Linux arm64
echo "Linux arm64 building version ${GIT_VERSION}"
CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags "-w -s" -o ${APP}-arm64 main.go
export CC=x86_64-w64-mingw32-gcc
# Windows amd64
echo "Windows amd64 building version ${GIT_VERSION}"
CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -ldflags "-w -s" -o ${APP}-amd64.exe main.go
export CC=aarch64-w64-mingw32-gcc


echo "Build completed"