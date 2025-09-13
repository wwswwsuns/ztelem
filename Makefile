SHELL := /bin/bash
PROTOC ?= protoc
PROTO_DIRS := proto
PROTO_FILES := $(shell find $(PROTO_DIRS) -name '*.proto')
GO_OUT_DIR := proto
BINARY_NAME := telemetry
DOCKER_IMAGE := telemetry:latest

.PHONY: tools proto build run clean fmt test docker-build docker-up docker-down db-init db-reset

tools:
	@which protoc >/dev/null || (echo "missing protoc" && exit 1)
	@which protoc-gen-go >/dev/null || (echo "missing protoc-gen-go" && exit 1)
	@which protoc-gen-go-grpc >/dev/null || (echo "missing protoc-gen-go-grpc" && exit 1)

proto: 
	$(PROTOC) -I proto \
	--go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative \
	--go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
	$(PROTO_FILES)

build: proto
	go mod tidy
	go build -o bin/$(BINARY_NAME) .

run: build
	./bin/$(BINARY_NAME) -config=config.yaml

run-debug: build
	./bin/$(BINARY_NAME) -config=config.yaml -debug=true

test:
	go test -v ./...

test-integration:
	go test -v -tags=integration ./...

fmt:
	gofmt -s -w .
	go mod tidy

clean:
	rm -rf bin

# Docker 相关命令
docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# 数据库相关命令
db-init:
	@echo "初始化TimescaleDB数据库..."
	psql -h localhost -p 5432 -U telemetry_app -d telemetry_db -f scripts/init_timescaledb.sql

db-reset:
	@echo "重置数据库..."
	psql -h localhost -p 5432 -U telemetry_app -d telemetry_db -c "DROP SCHEMA IF EXISTS telemetry CASCADE;"
	$(MAKE) db-init

db-status:
	@echo "检查数据库状态..."
	psql -h localhost -p 5432 -U telemetry_app -d telemetry_db -c "SELECT schemaname, tablename FROM pg_tables WHERE schemaname = 'telemetry';"

# 开发环境快速启动
dev-up: docker-up
	@echo "等待数据库启动..."
	sleep 10
	@echo "开发环境已启动！"
	@echo "gRPC服务: localhost:50051"
	@echo "数据库: localhost:5432"
	@echo "Grafana: localhost:3000 (admin/admin123)"

dev-down: docker-down

# 性能测试
bench:
	go test -bench=. -benchmem ./internal/storage/

# 代码检查
lint:
	golangci-lint run

# 生成依赖图
deps:
	go mod graph | dot -T png -o dependencies.png

# 帮助信息
help:
	@echo "可用命令:"
	@echo "  make build         - 构建应用"
	@echo "  make run           - 运行应用"
	@echo "  make test          - 运行测试"
	@echo "  make docker-up     - 启动Docker环境"
	@echo "  make dev-up        - 启动开发环境"
	@echo "  make db-init       - 初始化数据库"
	@echo "  make db-status     - 检查数据库状态"
	@echo "  make clean         - 清理构建文件"
