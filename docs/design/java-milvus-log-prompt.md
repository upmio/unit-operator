# Java 端 Milvus 日志获取功能开发 Prompt

## 需求背景

在 unit-operator 的 agent 端新增了 Milvus 日志转发功能（`milvus-logtail` DaemonApp）。该功能会将 Milvus 的日志文件（`unit_app.out.log` 和 `unit_app.err.log`）的内容实时输出到容器标准输出（stdout/stderr）。

Java 端需要新增接口，让前端页面能够获取这些实时日志流进行展示。

## 当前系统架构

### 现有日志获取方式
位置：`upm-common/upm-kubernetes-client/src/main/java/io/syntropycloud/upm/kubernetes/service/K8sPodService.java`

```java
public String getLog(String clusterName, String namespaceName, Map<String, String> labels, Integer sinceSeconds) {
    List<Pod> pods = getMixedOperation(clusterName).inNamespace(namespaceName).withLabels(labels).list().getItems();
    if (CollectionUtils.isNotEmpty(pods)) {
        PodResource podResource = getMixedOperation(clusterName).inNamespace(namespaceName).resource(pods.get(0));
        if (sinceSeconds != null) {
            return podResource.sinceSeconds(sinceSeconds).getLog();
        } else {
            return podResource.getLog();
        }
    }
    return null;
}
```

**问题**：该方法只支持获取历史日志，不支持实时流式获取。

### Agent 与 Java 端的通信机制
Java 端通过创建 `GrpcCall` CR（Custom Resource）与 agent 通信：

1. Java 端创建 `GrpcCall` 资源
2. Unit Operator 监控 `GrpcCall` 资源
3. Operator 通过 gRPC 与 Pod 中的 agent 通信
4. Agent 执行操作并更新 `GrpcCallStatus`

## 现有相关文件

| 文件路径 | 用途 |
|---------|------|
| `upm-common/upm-kubernetes-client/src/main/java/io/syntropycloud/upm/kubernetes/service/K8sPodService.java` | Pod 日志获取 |
| `upm-common/upm-kubernetes-client/src/main/java/io/syntropycloud/upm/kubernetes/service/K8sGrpcCallCrdService.java` | GrpcCall CRUD 操作 |
| `upm-common/upm-kubernetes-client/src/main/java/io/syntropycloud/upm/kubernetes/model/operator/grpccall/GrpcCallActionEnum.java` | gRPC 可用操作类型 |
| `upm-service/upm-service-milvus-ms/src/main/java/io/syntropycloud/upm/service/milvus/ms/service/impl/KubeResourceService.java` | Milvus K8s 资源管理 |
| `upm-service/upm-service-milvus-ms/src/main/java/io/syntropycloud/upm/service/milvus/ms/controller/` | Milvus API 控制器 |

## 需要开发的功能

### 方案一：使用 Kubernetes Watch Log（推荐）

Fabric8 Kubernetes Client 支持 `watchLog()` 方法，可以实时获取容器日志。

**参考实现**：
```java
// 伪代码示例
try (LogWatch logWatch = podResource.watchLog()) {
    InputStream in = logWatch.getOutput();
    // 读取并处理日志流
    BufferedReader reader = new BufferedReader(new InputStreamReader(in));
    String line;
    while ((line = reader.readLine()) != null) {
        // 处理每一行日志
        // 可以通过 WebSocket 推送到前端
    }
}
```

**需要新增的内容**：
1. 在 `K8sPodService` 中新增 `watchLog()` 方法
2. 新增 API 端点（REST Controller）
3. 可选：通过 WebSocket 或 SSE (Server-Sent Events) 推送到前端

### 方案二：通过 GrpcCall 触发日志转发

如果方案一不可行，可以新增一个 `GrpcCall` action（如 `STREAM_LOG`），让 agent 通过某种方式（如临时文件、Socket）将日志传输给 Java 端。

**注意**：这种方式复杂度较高，不推荐作为首选方案。

## 开发任务清单

### Task 1: 增强 K8sPodService
位置：`upm-common/upm-kubernetes-client/src/main/java/io/syntropycloud/upm/kubernetes/service/K8sPodService.java`

```java
/**
 * Watch pod log in real-time
 * @param clusterName cluster name
 * @param namespaceName namespace name
 * @param labels pod labels selector
 * @return LogWatch instance, caller should close it after use
 */
public LogWatch watchLog(String clusterName, String namespaceName, Map<String, String> labels) {
    List<Pod> pods = getMixedOperation(clusterName).inNamespace(namespaceName).withLabels(labels).list().getItems();
    if (CollectionUtils.isNotEmpty(pods)) {
        PodResource podResource = getMixedOperation(clusterName).inNamespace(namespaceName).resource(pods.get(0));
        return podResource.watchLog();
    }
    return null;
}

/**
 * Watch pod log with tail lines
 * @param clusterName cluster name
 * @param namespaceName namespace name
 * @param labels pod labels selector
 * @param tailLines number of lines to tail from end of log
 * @return LogWatch instance
 */
public LogWatch watchLog(String clusterName, String namespaceName, Map<String, String> labels, Integer tailLines) {
    List<Pod> pods = getMixedOperation(clusterName).inNamespace(namespaceName).withLabels(labels).list().getItems();
    if (CollectionUtils.isNotEmpty(pods)) {
        PodResource podResource = getMixedOperation(clusterName).inNamespace(namespaceName).resource(pods.get(0));
        if (tailLines != null) {
            return podResource.tailLines(tailLines).watchLog();
        }
        return podResource.watchLog();
    }
    return null;
}
```

### Task 2: 新增 Milvus 日志 API 端点
位置：`upm-service/upm-service-milvus-ms/src/main/java/io/syntropycloud/upm/service/milvus/ms/controller/`

参考现有的 Unit 相关端点，新增：

```java
/**
 * 获取 Milvus Unit 的实时日志
 * GET /units/{unitId}/logs/stream
 *
 * @param unitId Unit ID
 * @param tailLines 尾取的日志行数（可选，默认100）
 * @return 日志流（text/event-stream 或 chunked response）
 */
@GetMapping("/{unitId}/logs/stream")
public ResponseEntity<InputStream> streamUnitLog(
    @PathVariable String unitId,
    @RequestParam(required = false, defaultValue = "100") Integer tailLines) {

    // 1. 根据 unitId 获取 Unit 资源
    // 2. 获取 Unit 所属的 Pod labels
    // 3. 调用 K8sPodService.watchLog() 获取日志流
    // 4. 返回 InputStream
}
```

### Task 3: 前端对接（如果需要）
- 如果使用 SSE，需要新增 SSE 端点
- 如果使用 WebSocket，需要新增 WebSocket 配置
- 前端使用 EventSource 或 WebSocket 客户端接收日志流

## 关键依赖

确保以下依赖可用（Fabric8 Kubernetes Client）：
```xml
<dependency>
    <groupId>io.fabric8</groupId>
    <artifactId>kubernetes-client-api</artifactId>
    <version>${fabric8.version}</version>
</dependency>
```

## 测试要点

1. **单元测试**：测试 `K8sPodService.watchLog()` 方法
2. **集成测试**：
   - 部署带有新 agent 的 Milvus Pod
   - 调用日志流 API
   - 验证日志是否实时到达
3. **日志验证**：确认日志内容与 Milvus 容器内 `/dev/stdout` 输出一致

## Milvus 日志特点

根据 agent 实现，日志会同时从两个文件读取：
1. `${LOG_MOUNT}/unit_app.out.log` -> stdout
2. `${LOG_MOUNT}/unit_app.err.log` -> stderr

通过 Kubernetes 的 `getLog()` 或 `watchLog()` API，stdout 和 stderr 会合并到一起输出。

## 参考文档

- [Fabric8 Kubernetes Client - Log Watch](https://github.com/fabric8io/kubernetes-client#watching-logs)
- [Spring Boot WebFlux Streaming](https://spring.io/guides/gs/spring-boot-webflux/) - 如果需要响应式流

## 注意事项

1. **资源清理**：`LogWatch` 使用完毕后必须调用 `close()` 释放连接
2. **超时处理**：设置合理的超时时间，避免连接泄漏
3. **前端适配**：根据前端技术栈选择合适的推送方式（SSE/WebSocket/轮询）
4. **日志格式**：Milvus 日志为文本格式，前端可能需要解析时间戳、日志级别等

## 联调测试

完成 Java 端开发后，需要进行以下联调：

1. 部署新版 agent（包含 milvus-logtail）
2. 启动 Milvus Pod
3. 通过 Java API 获取日志流
4. 验证日志是否实时显示
5. 验证日志内容完整性
