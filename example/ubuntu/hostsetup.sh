#!/bin/bash

# 1. 添加本地离线源配置函数
setup_offline_repo() {
    echo "正在配置本地离线源: /opt/offline-repo..."

    # 备份现有的 sources.list (可选)
    [ -f /etc/apt/sources.list ] && mv /etc/apt/sources.list /etc/apt/sources.list.bak

    # 创建本地源配置文件
    # [trusted=yes] 允许跳过 GPG 签名检查，这在离线环境中非常实用
    echo "deb [trusted=yes] file:///opt/offline-repo ./" > /etc/apt/sources.list.d/offline.list

    # 清理并更新索引
    apt-get clean
    apt-get update
}

# 定义判断函数
check_nft_needs_update() {
    local ver=$(dpkg-query -W -f='${Version}' nftables 2>/dev/null | cut -d: -f2)
    [[ -z "$ver" ]] || dpkg --compare-versions "$ver" lt "1.0.1"
}

get_nvidia_version() {
    local os_ver=$(grep "VERSION_ID" /etc/os-release | cut -d'"' -f2)
    [ "$os_ver" = "18.04" ] && echo "525" || echo "580"
}

# 执行主逻辑
main() {
    # 第一步：初始化离线源
    setup_offline_repo

    if check_nft_needs_update; then
        echo "正在更新 nftables..."
        apt-get install -y nftables-custom
    fi

    if lspci | grep -qi nvidia; then
        echo "检测到 NVIDIA 显卡，正在安装驱动..."
        local nv_ver=$(get_nvidia_version)
        apt-get install -y nvidia-headless-$nv_ver nvidia-utils-$nv_ver nvidia-container-toolkit
    fi
}

main "$@"