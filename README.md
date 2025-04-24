# pprofsvr - Go语言实现的Profile文件服务器

## 项目简介
pprofsvr是一个基于Go语言开发的轻量级Profile文件服务器，主要用于：
- 提供HTTP接口浏览和分析pprof生成的性能分析文件
- 支持文件目录浏览和性能分析可视化

## 功能特性
- 支持pprof格式的性能分析文件展示
- 内置文件浏览器，支持目录结构展示
- 自动缓存管理，优化性能分析体验
- 简洁的Web界面，方便查看分析结果

## 快速开始

### 安装
```bash
go get github.com/xiateng/pprofsvr
```

### 运行
```bash
pprofsvr -p /path/to/profiles -addr :8080
 ```

### 参数说明
- -p : 指定profile文件存储路径（默认为当前目录）
- -addr : 指定监听地址（默认为:26817）
## 使用示例
1. 生成pprof文件到pprofsvr指定目录：
```bash
go tool pprof -http=:8080 cpu1.prof
go tool pprof -http=:8080 cpu2.prof
 ```

2. 通过pprofsvr查看：
```plaintext
# 查看profile文件列表
http://localhost:26817/ 
# 查看指定profile文件
http://localhost:26817/cpu1.prof/ui/
 ```
