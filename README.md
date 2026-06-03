# Account System · 金融账户系统

> Go + Gin + MySQL + Redis"。

完整支持开户、充值、提现、转账、对账核心流程,基于数据库事务 + 状态机 CAS 实现并发幂等,内置 JWT 鉴权与 Redis 黑名单失效控制。

---

## 快速开始

依赖:`Docker` 和 `Docker Compose`,无需本地安装 MySQL / Redis / Go。

```bash
git clone https://github.com/HSUJUI111/account-system.git
cd account-system
docker-compose up -d --build

# 等 30 秒,服务在 http://localhost:8080
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"123456"}'
```

完全重置(删数据):
```bash
docker-compose down -v
```

---

## 项目结构

```
account-system/
├── cmd/main.go                     # 入口
├── config/                         # 配置加载(支持环境变量覆盖)
├── internal/
│   ├── handler/                    # HTTP 层(按业务模块拆分)
│   ├── service/                    # 业务层(事务、幂等、状态机)
│   ├── middleware/                 # JWT 鉴权中间件
│   ├── repository/                 # 数据访问层(MySQL + Redis)
│   └── model/                      # GORM 数据模型
├── pkg/idgen/                      # 雪花 ID 生成
├── Dockerfile
└── docker-compose.yml
```

---

## 核心模块

### 1. 资金账户(`/api/accounts`)
- 双余额模型:可用余额(`available_balance`) + 冻结余额(`frozen_balance`)
- 一个用户 + 一种币种唯一一个账户(联合唯一索引)
- 对外账户号(`account_no`)用雪花 ID 生成,内部主键(`id`)自增

### 2. 充值(`/api/deposit`)
- 异步两阶段:`create` 创建订单(`status=1 处理中`)→ `confirm` 模拟支付回调入账(`status=2 成功`)
- 入账事务原子完成"改状态 + 加余额 + 记流水"三件事,任一步失败全部回滚

### 3. 提现(`/api/withdraw`)
- 冻结/解冻机制:申请时把可用余额转入冻结余额,避免代付期间被重复提取
- 双路径:代付成功扣冻结(`status=2`),代付失败解冻退回(`status=3`)

### 4. 转账(`/api/transfer`)
- 同步原子转账,事务内完成"A 扣 + B 加 + 双方流水"四个写操作
- **按 account_id 升序加悲观锁**,从根本上防止双向并发死锁
- 客户端可传 `transfer_no` 作为幂等键,服务端建唯一索引兜底

### 5. 对账(`/api/reconcile`)
- 流水累加自检:断言 `sum(收入流水) - sum(支出流水) == account.available_balance`
- 不一致项写入 `reconcile_alerts` 表,支持单账户或全量对账

### 6. JWT 鉴权(`/api/auth`)
- access token + refresh token 双 token 机制
- bcrypt 密码哈希、防用户名枚举、HMAC 算法显式校验
- Redis 黑名单实现登出即时失效

---

## 关键设计决策

### CAS 状态机:用一条带条件的 UPDATE 实现幂等

```sql
UPDATE deposit_order
SET status = 2, ...
WHERE order_no = ? AND status = 1;
```

支付回调重复到达时,第二次 `RowsAffected = 0`,直接当成功返回,不重复加钱。这是用"数据库行锁的原子性"代替"应用层 mutex 或分布式锁"——单条 UPDATE 即可同时完成"比较状态"和"切换状态",并发安全且零开销。

### 按 account_id 升序加锁规避转账死锁

A 转 B 和 B 转 A 并发发生时,如果按"转出方先锁"会形成循环等待。解法:**所有事务一律按 account_id 升序加锁**——`min(from_id, to_id)` 先锁,`max(...)` 后锁。两个方向的并发请求拿锁顺序完全相同,死锁从根本上不可能发生。已通过双向 100+ 并发压测验证无死锁、余额守恒。

### 双层幂等防线

每个写操作都有两层防御:
- **第一层**:业务订单号唯一索引(`order_no`、`transfer_no`)防重复提交
- **第二层**:状态机 CAS(`WHERE status = ?`)防重复处理
- **流水表兜底**:`(biz_order_no, biz_type, direction)` 联合唯一,防重复记账

### 提现先冻结再调外部 API

提现的钱要离开系统,顺序错了会资损。设计:
1. 事务内:可用余额 → 冻结余额(`WHERE available_balance >= amount` 内嵌校验)
2. 提交事务后再调外部代付 API(API 调用绝不能放在 DB 事务内)
3. 异步回调:成功扣冻结、失败解冻退回

---

## 实战中发现的问题

完成所有模块后跑全量对账,**1001 账户出现 70.5 元差额**。排查发现:对账公式 `sum(收入流水) - sum(支出流水) = 余额` 隐含假设"所有流水都动可用余额",但**提现成功扣冻结**这类流水只动冻结余额、不动可用,被错误地算成了可用余额支出。

**修复**:对账 SQL 排除 `biz_type = 4`(提现成功)的流水后,余额完全守恒。

**更深的认知**:这暴露了流水表设计的局限——单一流水表混记了两种余额变动。**生产级方案**应给流水表加 `balance_type` 字段(1=可用 2=冻结)或拆分冻结流水表,对账逻辑不再需要枚举 biz_type。

---

## 技术栈

- **语言**:Go 1.26
- **Web 框架**:Gin
- **ORM**:GORM v2(开启 TranslateError 翻译数据库错误)
- **数据库**:MySQL 8.0
- **缓存**:Redis 7(JWT 黑名单)
- **JWT**:golang-jwt/jwt v5
- **金额计算**:shopspring/decimal(避免浮点精度丢失)
- **ID 生成**:雪花算法(bwmarrin/snowflake)
- **密码哈希**:bcrypt


## API 速查表

> 所有金额字段以**字符串**形式传输和返回(避免 JSON 浮点精度丢失),前端需用 BigNumber 类库处理。

### 公开接口(无需鉴权)

| Method | 路径 | 说明 |
|---|---|---|
| POST | `/api/auth/register` | 用户注册 |
| POST | `/api/auth/login` | 登录,签发 access + refresh token |
| POST | `/api/auth/refresh` | 用 refresh token 换新 access |
| POST | `/api/auth/logout` | 登出,access token 拉黑 |

### 业务接口(需鉴权:Header `Authorization: Bearer <access_token>`)

| Method | 路径 | 说明 |
|---|---|---|
| POST | `/api/accounts/create` | 为当前用户开账户 |
| GET | `/api/accounts?currency=USD` | 查询当前用户的指定币种账户 |
| POST | `/api/deposit/create` | 创建充值订单 |
| POST | `/api/withdraw/create` | 申请提现(冻结余额) |
| POST | `/api/transfer` | 转账给其他用户 |

### 内部接口(模拟外部回调,生产环境应使用 HMAC 签名或内网隔离)

| Method | 路径 | 说明 |
|---|---|---|
| POST | `/api/deposit/confirm` | 模拟支付渠道回调,入账 |
| POST | `/api/withdraw/confirm` | 模拟代付渠道回调,确认成功或失败退回 |
| GET | `/api/reconcile/account/:id` | 单账户对账 |
| GET | `/api/reconcile/all` | 全量对账,返回所有不一致项 |

---

## 调用示例

```bash
# 1. 注册
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"123456"}'

# 2. 登录,拿到 access_token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456"}'

# 后面所有请求都用这个 token
export TOKEN=""

# 3. 开 USD 账户
curl -X POST http://localhost:8080/api/accounts/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"currency":"USD"}'

# 4. 充值 100
curl -X POST http://localhost:8080/api/deposit/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"currency":"USD","amount":"100"}'

# 5. 模拟支付回调(内部接口,无需 token)
curl -X POST http://localhost:8080/api/deposit/confirm \
  -H "Content-Type: application/json" \
  -d '{"order_no":""}'

# 6. 查余额(应该是 100)
curl "http://localhost:8080/api/accounts?currency=USD" \
  -H "Authorization: Bearer $TOKEN"

# 7. 申请提现 30
curl -X POST http://localhost:8080/api/withdraw/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"currency":"USD","amount":"30","payee_account":"6228****1234","payee_name":"张三","payee_bank":"工商银行"}'

# 8. 模拟代付成功回调
curl -X POST http://localhost:8080/api/withdraw/confirm \
  -H "Content-Type: application/json" \
  -d '{"order_no":"","success":true}'

# 9. 转账 10 给另一个用户(假设 to_user_id = 2)
curl -X POST http://localhost:8080/api/transfer \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"to_user_id":2,"currency":"USD","amount":"10"}'

# 10. 全量对账,验证账务一致
curl http://localhost:8080/api/reconcile/all

# 11. 登出
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```
### 错误码约定

| HTTP | 业务含义 | 触发场景 |
|---|---|---|
| 200 | 成功 | 业务正常返回 |
| 400 | 请求格式错 | JSON 解析失败、参数缺失、金额格式错 |
| 401 | 未鉴权 | 无 token、token 无效、token 已过期、token 被拉黑、密码错 |
| 404 | 资源不存在 | 账户/订单不存在 |
| 409 | 资源冲突 | 用户名/邮箱已注册、账户已存在 |
| 422 | 业务规则拒绝 | 余额不足 |
| 500 | 服务端故障 | 数据库连接异常等 |

---

## License

MIT