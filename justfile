set allow-duplicate-variables := true

import? '.just/local.just'
import '.just/default.just'
import '.just/mod/go.just'

pkg-ubuntu2204-amd64:
    TARGET_ARCH=amd64 \
    TARGET_VERSION=22.04 \
      go run ./example/ubuntu/main.go

pkg-ubuntu1804-amd64:
    TARGET_ARCH=amd64 \
    TARGET_VERSION=18.04 \
      go run ./example/ubuntu/main.go

debug-openeuler:
    go run ./example/openeuler/main.go
