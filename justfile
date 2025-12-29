set allow-duplicate-variables := true

import? '.just/local.just'
import '.just/default.just'
import '.just/mod/go.just'

pkg-ubuntu version arch:
    TARGET_ARCH={{ arch }}  TARGET_VERSION={{ version }} go run ./example/ubuntu/main.go

pkg-oe version arch:
    TARGET_ARCH={{ arch }}  TARGET_VERSION={{ version }} go run ./example/openeuler/main.go

pkg-ubuntu1804-amd64: (pkg-ubuntu "18.04" "amd64")

pkg-ubuntu2004-amd64: (pkg-ubuntu "20.04" "amd64")

pkg-ubuntu2204-amd64: (pkg-ubuntu "22.04" "amd64")

pkg-ubuntu2404-amd64: (pkg-ubuntu "24.04" "amd64")

pkg-oe2203-arm64: (pkg-oe "22.03" "arm64")

pkg-oe2203-amd64: (pkg-oe "22.03" "amd64")

pkg-oe2203sp2-amd64: (pkg-oe "22.03sp2" "amd64")

pkg-oe2403-amd64: (pkg-oe "24.03" "amd64")

