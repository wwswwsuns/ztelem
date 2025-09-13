#!/bin/bash

# Telemetry数据采集器部署脚本

set -e

echo "=== Telemetry数据采集器部署脚本 ==="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go 1.21+"
    exit 1
fi

# 检查protoc
if ! command -v protoc &> /dev/null; then
    echo "警告: 未找到protoc，如需重新生成proto文件请安装protoc"
fi

# 创建必要的目录
echo "创建目录结构..."
mkdir -p bin logs

# 编译程序
echo "编译程序..."
go mod tidy
go build -o bin/telemetry .

if [ $? -eq 0 ]; then
    echo "✓ 编译成功"
else
    echo "✗ 编译失败"
    exit 1
fi

# 检查配置文件
if [ ! -f "config.yaml" ]; then
    echo "创建默认配置文件..."
    cat > config.yaml << EOF
# Telemetry数据采集器配置文件

# 数据库配置
database:
  host: "localhost"
  port: 5432
  user: "telemetry_app"
  password: "your_password"
  dbname: "telemetrydb"
  schema: "telemetry"
  max_connections: 10
  
# gRPC服务配置
grpc:
  port: 50051
  
# 日志配置
logging:
  level: "info"
  file: "logs/telemetry.log"
  max_size: 100
  max_backups: 3
  max_age: 28
  
# 调试模式
debug: false
EOF
    echo "✓ 已创建默认配置文件 config.yaml，请根据实际环境修改"
fi

# 创建systemd服务文件
echo "创建systemd服务文件..."
cat > telemetry.service << EOF
[Unit]
Description=Telemetry Data Collector
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$(pwd)
ExecStart=$(pwd)/bin/telemetry -config $(pwd)/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

echo "✓ 已创建systemd服务文件 telemetry.service"

# 设置权限
chmod +x bin/telemetry

echo ""
echo "=== 部署完成 ==="
echo ""
echo "下一步操作："
echo "1. 修改 config.yaml 配置文件"
echo "2. 确保TimescaleDB数据库已准备就绪"
echo "3. 运行程序："
echo "   前台运行: ./bin/telemetry -config config.yaml"
echo "   后台运行: nohup ./bin/telemetry -config config.yaml > /dev/null 2>&1 &"
echo "   调试模式: ./bin/telemetry -config config.yaml -debug"
echo ""
echo "4. 安装为系统服务（可选）："
echo "   sudo cp telemetry.service /etc/systemd/system/"
echo "   sudo systemctl daemon-reload"
echo "   sudo systemctl enable telemetry"
echo "   sudo systemctl start telemetry"
echo ""
echo "5. 查看日志："
echo "   tail -f logs/telemetry.log"
echo "   journalctl -u telemetry -f"
echo ""