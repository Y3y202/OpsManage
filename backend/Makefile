.PHONY: build run swagger clean

# 生成 Swagger 文档
swagger:
	swag init -g main.go -o docs --parseDependency --parseInternal

# 编译
build: swagger
	go build -o opsmanage .

# 运行
run: build
	./opsmanage

# 清理
clean:
	rm -f opsmanage
