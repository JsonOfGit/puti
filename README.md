# gingob
Golang+Vue

# Build by Docker
## Using a MySQL in lcoal
运行一个mysql容器供本地使用
```
docker run --name go-mysql -p 3306:3306 -v E:/data/mysql:/var/lib/mysql -e MYSQL_ROOT_PASSWORD=123456 -d mysql
```

## Compile the gingob app inside the Docker container
容器内编译应用：
```
$ docker run --rm -v "$PWD":/go/src/gingob -w /go/src/gingob golang go build -v
```
会存在依赖包不存在的问题，所以把src整个挂载：
```
$ docker run --rm -v E:/GoPath/src:/go/src -w /go/src/gingob golang go build -v
```
## Cross-compile your app inside the Docker container
交叉编译，例如编译一个Windows平台二进制文件:
```
$ docker run --rm -v E:/GoPath/src:/go/src -w /go/src/gingob -e GOOS=windows -e GOARCH=amd64 golang go build -v
```

## Build by a Makefile (recommend)
容器内通过 Makefile 编译应用：
```
$ docker run --rm -v "$PWD":/go/src/gingob -w /go/src/gingob golang make
$ docker run --rm -v E:/GoPath/src:/go/src -w /go/src/gingob golang make
$ docker run --rm -v E:/GoPath/src:/go/src -w /go/src/gingob -e GOOS=windows -e GOARCH=amd64 golang make
```

## Start a Go instance in your app
如果要在容器内运行应用，通过运行 dockerfile 构建镜像来使用：
```
$ docker build -t golang-gingob .
$ docker run -it --rm --name gingob goalng-gingob
```
但是 go get 会失败，最好还是编排后使用。