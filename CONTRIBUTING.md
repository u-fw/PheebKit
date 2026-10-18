# 参与开发

保持 Go Core、固定稳定 Wails v2.17.0、React/TypeScript 和现有 lockfile。2026-10-18 用户批准从 v3 Beta 迁移到稳定 v2；当前产品不再依赖 v3。新增依赖前核验官方来源，避免无关升级。

使用 Go 1.27.2 安全工具链、Node 24+、固定 Wails CLI，按 README 安装。干净源码目录应先构建真实前端产物，再执行会加载根包的 Go 检查：

```powershell
npm --prefix frontend ci
npm --prefix frontend run build
go test ./...
go vet ./...
npm --prefix frontend run test
npm --prefix frontend run typecheck
npm --prefix frontend run lint
npm --prefix frontend run format:check
```

修改 Go 服务接口后，重新生成并提交 TypeScript bindings。`VERSION` 是版本来源，运行 `node scripts/sync-version.mjs` 同步前端、Windows 元信息及 Go 诊断版本。不要手改生成文件。

真实 FFmpeg 检查使用自备环境，或在原生 Windows x64 上运行 `scripts/setup-test-ffmpeg.ps1`。该脚本下载目录内已固定散列的发布者工具，仅放入 `.tools`，不修改系统 PATH 或发布包。随后设置 `KOMPEG_INTEGRATION=1` 并运行 Go 测试。强制终止恢复测试只结束它自己创建的无界面测试子进程。

提交应包含必要回归检查、用户可见变化和实际验证范围。不要提交二进制、媒体、AppData、个人路径、凭据或诊断样本。公开反馈前检查所有附件。项目/草稿/配置格式变动需说明兼容性，不能静默丢弃旧数据。

发布检查见 `docs/release-acceptance.md`。本地打包要求已提交的干净源码；创建候选包不代表已完成干净机器、原生界面或托管 CI 验收。发布、推送、签名和公开漏洞披露由维护者明确决定。
