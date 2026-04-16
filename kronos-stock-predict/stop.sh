#!/bin/bash

echo "停止 Kronos 股票预测系统..."

# 停止所有相关服务
pkill -f "backend/server" 2>/dev/null || true
pkill -f "python.*app.py" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true

echo "所有服务已停止"
