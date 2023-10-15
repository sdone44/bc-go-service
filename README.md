### 编译：
````
go build -o bc_server register_server.go
go build -o bc_server_amd register_server.go
````


### 配置.env：
```
DB_PREFIX=
DB_ROOT_PASSWORD=
DB_DATABASE=
DB_USERNAME=
DB_PASSWORD=
DB_PORT=
DB_HOST=

PORT=

RPC_URL=
CONTRACT_ADDRESS=
CONTRACT_ABI=''
CONTRACT_METHOD_SHARADATE=
```

### 安装MariaDB数据库：
```
docker-compose up -d
```

### 运行：
```
chmod +x ./bc_server
nohup ./bc_server > service.log &
```

### 查看：
```
ps -ef | grep "bc_server"
```

### 停止：
```
ps -ef | grep "bc_server" | grep -v grep | awk '{ print $2 }' | xargs kill -9
```