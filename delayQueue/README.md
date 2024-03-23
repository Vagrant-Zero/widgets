# Delay Queue

## 事件抽象定义
```protobuf
message DelayEventDto{
   string key = 1; // 事件全局唯一key
   string topic = 2; // 发送到的topic
   bytes data = 3; // 事件包体
}
```

## redis中心化存储
1. 利用一个set来存储过期task，zset存储延期task；
2. 为避免redis大key的问题，redis中zset按照按照分钟进行划分，即每个分钟级时间片对应一个zset；
3. 采用hashTag的策略，保证`addTask`和`deleteTask`一定在同一个redis实例上执行；

## Example

```go
package main

import (
   "context"
   "fmt"
)

func main() {
   ctx := context.Background()
   event := &delay_queue.DelayEventDto{
      Key:   "xxx",
      Topic: "xxx",
      Data:  map[string]string{"test_key": "test_value"},
   }
   err := delay_queue.SendEvent(ctx, event)
   if err != nil {
      fmt.Printf("send delay event error, err=%v\n", err)
      return
   }
   return
}
```

## TODO
1. 异步发送事件失败，如果告知调用方？
    > 这里的发送失败是指发送到kafka失败，同步失败在调用api的时候已经就返回error了
2. 