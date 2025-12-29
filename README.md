# Host setup

## 按需构建离线仓库

* Ubuntu
    - [x] 18.04/amd64 `just pkg-ubuntu1804-amd64`
    - [x] 20.04/amd64 `just pkg-ubuntu2004-amd64`
    - [x] 22.04/amd64 `just pkg-ubuntu2204-amd64`
    - [x] 24.04/amd64 `just pkg-ubuntu2404-amd64`
* OpenEuler
    - [x] 22.03/arm64 `just pkg-oe2203-arm64`
    - [x] 22.03/amd64 `just pkg-oe2203-amd64`

## 同步 `/opt/offline-repo`

在主机上执行：

```shell
bash /opt/offline-repo/setup.sh
```