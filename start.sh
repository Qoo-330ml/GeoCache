#!/bin/bash
# 本地启动脚本（无 Docker）

set -e

cd "$(dirname "$0")"

# 创建数据目录
mkdir -p data

# 安装依赖（如果需要）
if [ ! -d "venv" ]; then
    echo "创建虚拟环境..."
    python3 -m venv venv
fi

echo "激活虚拟环境..."
source venv/bin/activate

echo "安装依赖..."
pip install -q -r requirements.txt

echo "启动 GeoCache 服务（端口 18080）..."
uvicorn app.main:app --host 0.0.0.0 --port 18080 --reload