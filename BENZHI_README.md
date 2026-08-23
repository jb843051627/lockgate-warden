# lockgate-warden 运行镜像

内河船闸监控调度服务。

## 构建

```bash
./build_benzhi_docker.sh lockgate-warden linux/amd64
./build_benzhi_docker.sh lockgate-warden linux/arm64
```

镜像名固定 `benzhi/lockgate-warden:latest`（后构建架构覆盖标签属规范口径）。

## 容器内验证

```bash
docker run --rm --platform linux/amd64 --entrypoint /bin/sh benzhi/lockgate-warden:latest -lc "go version && go build ./..."
```
