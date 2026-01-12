package redis

// luaPeekAndRem 实现了分布式延时队列的“消费并删除”原子操作。
// @Logic
// 1. ZRANGEBYSCORE: 基于当前系统时间戳，在有序集合(ZSet)中检索所有已到期的任务 ID。
// 2. ZREM: 同步从 ZSet 中剔除上述命中的任务，防止任务被并发节点重复拉取。
// 3. Return: 将命中的任务 ID 列表返回给调用方进行后续的业务处理。
//
// @Constraints
// - 原子性保障：通过 Lua 脚本执行，确保读取与删除之间不被其他命令插入。
// - 性能限制：调用方需合理控制 ARGV[2] (limit)，避免大批量删除导致 Redis 阻塞。
//
// @Parameters
// KEYS[1] - string: 延时队列的 ZSet 键名 (e.g., "dq_tasks_zset")
// ARGV[1] - int64 : 当前 Unix 时间戳 (Score)，用于判定任务是否到期
// ARGV[2] - int   : 单词拉取的最大任务数量 (Limit)，用于流量削峰
//
// @Returns
// table: 返回包含任务 Payload (ID) 的数组；若无到期任务则返回空 Table。
const luaFetchAndHold = `
local pending_key = KEYS[1]
local running_key = KEYS[2]
local max_score = ARGV[1]
local limit = ARGV[2]
local now = ARGV[3]

-- 1. 检索所有 Score 小于等于当前时间戳的任务
local raw_tasks = redis.call('ZRANGEBYSCORE', pending_key, 0, max_score, 'LIMIT', 0, limit)

if #raw_tasks > 0 then
    for i, raw_json in ipairs(raw_tasks) do
        -- 2. 解析 TaskID (Redis 内置 cjson 库)
        -- 注意：这里假设 raw_json 是合法的 JSON 字符串
        local task = cjson.decode(raw_json)
        local id = task.id

        -- 3. 从 Pending 移除
        redis.call('ZREM', pending_key, raw_json)

        -- 4. 构造 Running 记录 (包装一下，记录开始时间)
        -- 格式: {"start": 1700000000, "task": {...}}
        local running_data = cjson.encode({start = tonumber(now), task = task})
        
        -- 5. 写入 Running Hash
        redis.call('HSET', running_key, id, running_data)
    end
    return raw_tasks
else
    return {}
end
`

// luaAck 确认任务完成
// KEYS[1]: Running Hash (ddq:running)
// ARGV[1]: TaskID
const luaAck = `
return redis.call('HDEL', KEYS[1], ARGV[1])
`

// luaNack 任务失败重试
// @Logic
// 1. 从 Running 移除
// 2. 判断是否超过最大重试次数
// 3. 没超过 -> 更新 retry_count -> ZADD 回 Pending
// 4. 超过了 -> LPUSH 到 DLQ (死信队列)
//
// @Parameters
// KEYS[1]: Running Hash (ddq:running)
// KEYS[2]: Pending ZSet (ddq:tasks)
// KEYS[3]: Dead Letter Queue (ddq:dlq)
// ARGV[1]: TaskID
// ARGV[2]: JSON Payload (包含更新后的 retry_count 的完整 task 结构)
// ARGV[3]: Next Execute Time (重试的执行时间，通常是现在)
// ARGV[4]: Is Dead (1=进死信, 0=重试)
const luaNack = `
local running_key = KEYS[1]
local pending_key = KEYS[2]
local dlq_key = KEYS[3]

local id = ARGV[1]
local task_json = ARGV[2]
local score = ARGV[3]
local is_dead = tonumber(ARGV[4])

-- 1. 无论如何，先从正在运行列表移除
redis.call('HDEL', running_key, id)

if is_dead == 1 then
    -- 2. 超过重试次数，进死信队列
    redis.call('LPUSH', dlq_key, task_json)
else
    -- 3. 没超过，放回等待队列重试
    redis.call('ZADD', pending_key, score, task_json)
end

return 1
`
