# Kronos Stock Predictor

基于 Kronos Foundation Model 的股票预测系统，集成 gotdx 实时行情数据。

## 功能特性

- 全A股股票列表展示
- 基于 Kronos 的 K 线预测（120/30/10/5 根日线）
- 实时行情数据获取（gotdx）
- 本地 K 线数据存储
- 增量更新与全量强制更新
- 数据完整性验证

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (React)                        │
│  - 股票列表展示                                               │
│  - 预测结果展示                                                │
│  - 数据管理界面                                                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Backend API (Go)                          │
│  - gotdx 实时行情获取                                          │
│  - K 线数据存储 (SQLite)                                      │
│  - 增量/全量更新                                              │
│  - 数据验证                                                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Prediction Service (Python)                  │
│  - Kronos 模型推理                                            │
│  - 多周期预测（120/30/10/5根）                                 │
│  - 预测分数计算                                               │
└─────────────────────────────────────────────────────────────┘
```

## 快速开始

### 方式一：使用启动脚本

```bash
cd kronos-stock-predict
./start.sh
```

### 方式二：手动启动

#### 1. 编译后端

```bash
cd backend
go build -o server ./cmd/server/
```

#### 2. 安装预测服务依赖

```bash
cd prediction
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

#### 3. 安装前端依赖

```bash
cd frontend
npm install
```

#### 4. 启动服务

```bash
# 启动后端 (端口 8080)
cd backend && ./server

# 启动预测服务 (端口 8081) - 新开终端
cd prediction && source venv/bin/activate && python app.py

# 启动前端 (端口 3000) - 新开终端
cd frontend && npm run dev
```

## 访问地址

| 服务 | 地址 |
|------|------|
| 前端界面 | http://localhost:3000 |
| 后端 API | http://localhost:8080 |
| 预测服务 | http://localhost:8081 |

## API 接口

### 后端 API

| 接口 | 方法 | 描述 |
|------|------|------|
| `/api/stocks` | GET | 获取股票列表 |
| `/api/stock/:code` | GET | 获取单只股票信息 |
| `/api/kline/:code` | GET | 获取K线数据 |
| `/api/kline/:code/update` | POST | 增量更新K线 |
| `/api/kline/:code/full` | POST | 全量更新K线 |
| `/api/kline/:code/verify` | POST | 验证K线数据 |
| `/api/predict` | POST | 获取预测结果 |
| `/api/health` | GET | 健康检查 |

### 预测服务 API

| 接口 | 方法 | 描述 |
|------|------|------|
| `/predict` | POST | 预测下一根K线 |
| `/health` | GET | 健康检查 |

## 数据格式

### K线数据 (Kronos 格式)

```csv
timestamps,open,high,low,close,volume,amount
2024-01-02 09:30:00,100.0,101.0,99.5,100.5,1000000,100500000
```

### 预测请求

```json
{
  "code": "600519",
  "lookbacks": [120, 30, 10, 5]
}
```

### 预测响应

```json
{
  "code": "600519",
  "name": "贵州茅台",
  "predictions": {
    "120": {
      "next_kline": {"open": 1700, "high": 1720, "low": 1680, "close": 1710},
      "direction": "up",
      "change_pct": 0.58,
      "score": 0.72
    }
  }
}
```

## 配置文件

### backend/config.yaml

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "../data/stock.db"

gotdx:
  timeout: 10
  retry: 3
  market_sh: "上海主板"
  market_sz: "深圳主板"

prediction:
  url: "http://localhost:8081"
```

### prediction/config.yaml

```yaml
kronos:
  model_name: "NeoQuasar/Kronos-small"
  device: "cuda"  # or "cpu"
  max_context: 512

server:
  host: "0.0.0.0"
  port: 8081
```

## 数据管理

系统支持以下数据管理功能：

1. **增量更新**: 仅更新最新K线数据，适合日常维护
2. **全量更新**: 重新获取所有历史K线数据，适合首次导入或数据修复
3. **数据验证**: 检查本地数据的完整性和准确性

## 技术栈

- **后端**: Go 1.21+, gotdx
- **预测服务**: Python 3.10+, Kronos, PyTorch
- **数据库**: SQLite
- **前端**: React, Vite, TailwindCSS

## 目录结构

```
kronos-stock-predict/
├── backend/              # Go 后端服务
│   ├── cmd/server/       # 服务入口
│   ├── internal/         # 内部包
│   │   ├── api/          # API 处理器
│   │   ├── data/          # 数据库层
│   │   ├── gotdx/         # gotdx 封装
│   │   └── models/        # 数据模型
│   └── config.yaml        # 配置文件
├── prediction/           # Python 预测服务
│   ├── app.py            # Flask 应用
│   ├── kronos_predictor.py  # Kronos 封装
│   └── config.yaml       # 配置文件
├── frontend/             # React 前端
│   ├── src/
│   │   ├── components/   # React 组件
│   │   └── App.jsx       # 主应用
│   └── package.json
├── data/                 # 数据存储目录
├── start.sh             # 启动脚本
└── README.md
```

## 注意事项

1. **网络要求**: gotdx 需要连接通达信行情服务器，确保网络畅通
2. **首次运行**: 首次运行需要获取全量股票列表和K线数据，可能需要较长时间
3. **预测模型**: Kronos 模型会自动下载，首次使用时需要网络连接
4. **数据库**: 数据库文件较大(150MB+),建议启动服务后运行 `POST /api/sync/trigger` 自动同步数据

## 数据库说明

数据库文件 (`data/stock.db`) 包含:
- A股股票列表 (60xxx/00xxx/30xxx/68xxx)
- 日K线数据 (每只股票150根)
- 预测结果 (120/30/10/5根日线预测)

首次运行后端服务后,调用 `POST /api/sync/trigger` 自动获取所有数据。

## 许可证

MIT
