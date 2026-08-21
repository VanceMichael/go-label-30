# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

两名操作员几乎同时启动不同的饲喂计划，两条审计都通过了授权，也都出现在日志中；随后合规校验却报告 sequence 1 重复、前驱链断裂，整段审计无法验签。任意一条单独提交都正常。请修复并发追加的链尾协调，让授权等待不会固定住过期的前驱，所有成功事件最终形成唯一连续的哈希链，并确保竞态检测通过。

## 含 Bug 版本

- 仓库：VanceMichael/go-label-30
- 仓库地址：https://github.com/VanceMichael/go-label-30.git
- parent SHA：aaa3db5e5271b3904c09d149e8dd95aa65444b23

## 复现步骤

```bash
git clone -- https://github.com/VanceMichael/go-label-30.git bug-repro
cd bug-repro
git checkout --detach aaa3db5e5271b3904c09d149e8dd95aa65444b23
go test ./internal/audit -run ^TestConcurrentAppendKeepsSingleAuditChain$ -count=1
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/audit -run ^TestConcurrentAppendKeepsSingleAuditChain$ -count=1
--- FAIL: TestConcurrentAppendKeepsSingleAuditChain (0.02s)
    ledger_test.go:45: concurrent append forked audit chain: state conflict: audit sequence 1; events=[{ID:event-a TenantID:farm ActorID:operator-a Action:feed.start ObjectType:plan ObjectID:plan-a Outcome:ok RequestID:request-a Details:map[] OccurredAt:2026-08-21 07:28:00.910846381 +0000 UTC Sequence:1 PreviousHash: Hash:3fc153fba4dde8285a72a8d7d855bfdb24de97e6ddbc08f3ab61285895232bf3} {ID:event-b TenantID:farm ActorID:operator-b Action:feed.start ObjectType:plan ObjectID:plan-b Outcome:ok RequestID:request-b Details:map[] OccurredAt:2026-08-21 07:28:00.910846381 +0000 UTC Sequence:1 PreviousHash: Hash:c41486e36f3600eb96ba9854fccf090af2042b911bd755137084d512bf710845}]
FAIL
FAIL	go-base/internal/audit	0.053s
FAIL

```

stderr：

```text
(empty)
```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/audit -run ^TestConcurrentAppendKeepsSingleAuditChain$ -count=1
--- FAIL: TestConcurrentAppendKeepsSingleAuditChain (0.00s)
    ledger_test.go:45: concurrent append forked audit chain: state conflict: audit sequence 1; events=[{ID:event-b TenantID:farm ActorID:operator-b Action:feed.start ObjectType:plan ObjectID:plan-b Outcome:ok RequestID:request-b Details:map[] OccurredAt:2026-08-21 07:28:33.991538632 +0000 UTC Sequence:1 PreviousHash: Hash:7316c7ceaeab7ac276aebc0b8148698deb7e290971a80dd485d09805e2725809} {ID:event-a TenantID:farm ActorID:operator-a Action:feed.start ObjectType:plan ObjectID:plan-a Outcome:ok RequestID:request-a Details:map[] OccurredAt:2026-08-21 07:28:33.991538632 +0000 UTC Sequence:1 PreviousHash: Hash:cb66410cfc0e5a3e9a9d074b4ae114103c0d3969e1f33d68d342c85ad30dd25f}]
FAIL
FAIL	go-base/internal/audit	0.001s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

两个 Append 在同一个授权屏障后同时继续时，Ledger 必须保存两条均已授权的事件，最终 Sequence 连续为 1、2，第二条 PreviousHash 指向第一条且 Verify 校验完整链成功；单条追加语义和详情副本隔离不变。TestConcurrentAppendKeepsSingleAuditChain 在 -race 下要由红转绿，其他包回归和 go build ./... 继续通过，不得丢弃任一成功事件、串行持锁调用外部授权器或放宽哈希链验证规则。
