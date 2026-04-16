#!/bin/bash

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_ROOT"

echo "=== Kronos 股票预测系统启动脚本 ==="

# 创建数据目录
mkdir -p data

# 编译后端
echo "[1/4] 编译后端..."
cd backend
go build -o server ./cmd/server/
cd ..

# 安装预测服务依赖
echo "[2/4] 安装预测服务依赖..."
cd prediction
if [ ! -d "venv" ]; then
    python3 -m venv venv
fi
source venv/bin/activate
pip install -q -r requirements.txt
cd ..

# 安装前端依赖
echo "[3/4] 安装前端依赖..."
cd frontend
if [ ! -d "node_modules" ]; then
    npm install
fi
cd ..

echo "[4/4] 启动服务..."

# 启动后端服务
echo "启动后端服务 (端口 8080)..."
cd backend
./server &
BACKEND_PID=$!
cd ..

# 等待后端启动
sleep 2

# 启动预测服务
echo "启动预测服务 (端口 8081)..."
cd prediction
source venv/bin/activate
python app.py &
PREDICT_PID=$!
cd ..

# 等待预测服务启动
sleep 2

# 启动前端
echo "启动前端服务 (端口 3000)..."
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

echo ""
echo "=== 服务已启动 ==="
echo "后端 API:     http://localhost:8080"
echo "预测服务:     http://localhost:8081"
echo "前端界面:     http://localhost:3000"
echo ""
echo "按 Ctrl+C 停止所有服务"

# 等待信号
wait
