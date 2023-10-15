### 编译：
````
go build -o bc_server register_server.go
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


### 运行：
```
chmod +x ./bc_server
nohup ./bc_server > service.log 2>&1 & echo $! > pidfile
```

### 停止：
```
kill $(cat pidfile)
```