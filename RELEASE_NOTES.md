# PheebKit 0.2.0-rc.6

- Windows 媒体子进程先挂起创建，加入 Job Object 后才运行，封住启动与绑定之间的后代进程逃逸窗口；提前取消会被保留到绑定阶段。
- 队列事件缺口同步现在合并并发请求，并在 IPC 临时失败后按 0.5–30 秒退避重试；恢复成功或组件卸载后停止重试。
- 最终任务历史写入失败会阻止静默退出，支持重试保存或明确选择放弃本次历史后退出。退出磁盘写入不阻塞原生窗口线程；输出视频文件不受影响。
- 定期安全审计同时扫描源码与真实 Wails 生产 EXE，覆盖仅在生产构建标签下链接的依赖。发布作业继续校验制品来源、SHA-256 和标签一致性。
- 本候选包含 rc.5 发布之后的进程回收、内置 FFmpeg 缓存完整性、后台历史保存、事件顺序、预览能力释放、队列分页与损坏媒体粗细扫描优化，以及 x/net v0.62.0、x/crypto v0.58.0、x/text v0.43.0 安全修复。
- 当前仍为 Windows x64 验收候选。普通 main CI 只验证源码和生产二进制；包体、安装器生命周期与制品来源证明由独立候选打包运行验收，不将它们混同。
- 升级验收改为下载并校验已发布旧版的真实包体，覆盖 GitHub 制品摘要、清单、来源与标签提交；不再为此自动安装历史 Wails v3 Beta 工具，也不以重建旧源码的结果冒充用户正在使用的安装包。
- 配置文件写入增加按文件的进程内排队，避免高并发和慢磁盘耗尽跨进程文件锁的等待预算；不同文件继续并行，空闲锁自动回收。

## English — rc.6

- Create Windows media processes suspended, attach them to a kill-on-close Job Object, then resume. This closes the descendant escape window and retains cancellation received before attachment.
- Coalesce queue snapshot recovery requests and retry transient IPC failures with 0.5–30-second backoff. Stop retrying after recovery or disposal.
- Keep the app visible if its final history write fails, with explicit retry or exit-without-history actions. Flush off the native window thread; generated media files are unaffected.
- Periodic security audits inspect both source and the actual Wails production EXE, including production-tag dependencies. Publication retains checksum, provenance and tag checks.
- Includes the post-rc.5 process cleanup, trusted FFmpeg cache, background persistence, sequenced events, preview release, queue pagination and bounded recovery scanning changes, plus the x/net v0.62.0, x/crypto v0.58.0 and x/text v0.43.0 security fixes.
- Remains a Windows x64 verification candidate. Main-branch checks do not establish package or installer acceptance; use the separate candidate packaging and attestation run.
- CI upgrade acceptance now verifies the actual published baseline assets, GitHub digests, checksum manifest and exact tag provenance before execution, without bootstrapping historical Wails v3 Beta tooling.
- Queue in-process configuration writers per file before starting the cross-process lock deadline. Slow local writes no longer consume that budget; unrelated files remain independent and idle locks are reclaimed.

## Previous candidate: 0.2.0-rc.5

- 新增轻量直跑 EXE、FFmpeg 内置单文件 EXE、内置 FFmpeg 便携 ZIP 和可选安装器四种发布形态；统一附带 SHA-256 清单。
- 内置运行时固定为 Gyan FFmpeg 9.0.2 full build，打包时验证来源归档、`ffmpeg.exe`、`ffprobe.exe` 和实际版本。单文件版首次启动时原子解压到用户缓存并校验；ZIP 直接使用相邻 `runtime/ffmpeg`。
- 环境发现顺序为显式/hosted 兼容环境变量 → 本机保存路径 → 内置运行时 → PATH；设置页区分内置运行时。可执行文件提供无窗口 `--environment-info` 验收入口。
- 内置 FFmpeg 不等于完全离线：所有版本仍需 Microsoft WebView2 Runtime。内置包不触发重复下载，提供 FFmpeg GPL v3 许可证、Gyan README、对应源码提交和哈希。

## Previous candidate: 0.2.0-rc.4

- 安全构建工具链更新为 Go 1.27.3，修复下载链路可达的标准库证书验证风险 GO-2026-6809；本地发布门槛与 CI 同步要求修复版。
- 前端锁文件将 ESLint Hooks 插件的间接 `zod` 依赖固定到 4.6.7，修复 GHSA-mrhx-c7xc-m3w5；Node 24 `npm audit` 当前为零已知问题。该依赖仅处于开发/校验链，不进入桌面运行时。

- Material 3 / Material You 工作台：使用官方 HCT 算法生成种子色主题，支持跟随 Windows 强调色、系统／浅色／深色、预设和自定义主题色、舒适／紧凑密度及本机持久保存。新增 `Ctrl+K` 快速操作、指针波纹、减少动效与高对比兼容；桌面端直接在稳定 Wails v2 + React 中落实 Compose/Material 设计原则。未来 Android 端可使用原生 Jetpack Compose。
- 参数分为输出、轨道、调整和工具四组，切换保留状态；长桌面使用固定参数面板，底部输出检查栏适配窄屏、放大字体与多条错误。毛玻璃仅用于应用框架和浮层。
- 源视频支持完整预览、所选范围循环、近似按帧步进、最多八张显式加载的缩略图、有界单声道波形窗口，以及片段按钮／键盘排序。浏览预览不会改变输出范围。
- 样片支持带原片偏移的同步播放、单路试听、并排静帧和分屏揭示；不支持的播放编码可回退到 FFmpeg 静帧。修复首次按需加载及媒体信息到达后的比较区定位，并在关闭后恢复入口焦点。
- 批量处理复用已保存的通用配方，支持逐文件轨道、名称、实际编码器和命令预检。全部通过后显式整批入队；设置变化使预检失效，提交时再次原子检查且不覆盖已有结果。停止预检在当前检查结束后生效。
- 原生实测已覆盖中文路径双文件导入、GPU 样片／批量编码、同步及静帧比较；独立完整解码通过。开发实测截图和准确边界见 docs/uiux-iteration.md 与 docs/images/README.md。
- 当前为新的验收候选；打包、托管 CI 和干净 Windows 客户端门槛按各自证据记录。Android 继续暂缓，不将交互同步声称为逐解码帧锁定，也不将单个样片视为全片质量评估。

## English

- Adds four release forms: lightweight direct EXE, embedded single-file EXE, portable ZIP with adjacent FFmpeg, and an optional installer, all with SHA-256 sums.
- Pins the bundled runtime to the Gyan FFmpeg 9.0.2 full build and verifies its source archive, executables and reported version during packaging. Embedded editions atomically materialize a checked cache on first launch; the ZIP uses `runtime/ffmpeg`.
- Discovery order is explicit/compatibility environment variables → saved local paths → bundled runtime → PATH. Settings identifies the bundled source, and `--environment-info` provides a windowless acceptance probe.
- Bundled FFmpeg is not a claim of complete offline operation because WebView2 remains required. Bundled artifacts include FFmpeg's GPL v3 license, Gyan README, source commit and hashes.

### Previous candidate: 0.2.0-rc.4

- Pin Go 1.27.3 and require it for release builds/CI to address the reachable standard-library certificate-validation advisory GO-2026-6809.
- Refresh the transitive `zod` development dependency used by the ESLint Hooks plugin to 4.6.7, addressing GHSA-mrhx-c7xc-m3w5. `npm audit` is clean under Node 24; the package is not part of the desktop runtime.

- Material HCT seed palettes now include Windows Material You dynamic color, preset/custom seeds, system/light/dark modes and density in the stable Wails v2 / React app. A `Ctrl+K` command palette, pointer ripples, reduced motion and forced-colour support join the grouped, retained parameter panels and responsive output dock. The desktop shell applies Compose/Material principles; a future Android client can use native Jetpack Compose.
- Source looping, nominal frame steps, bounded filmstrip/waveform extraction and accessible segment ordering. Preview navigation does not change output intent.
- Offset-aware synchronized sample playback with one audible audio source, static side-by-side/split comparison and an explicit codec fallback. First-open lazy/metadata navigation and return focus are fixed.
- Reusable portable batch recipes and sequential per-file preflight with actual encoder/track/command decisions. All files must pass before explicit atomic admission; edits invalidate review and existing outputs remain protected.
- Actual native sample/batch encoding and independent full-output decoding passed on the current workstation. This is a verification candidate; package/CI provenance and clean-client acceptance are recorded separately. Android remains deferred.

## Previous candidate: 0.2.0-rc.3

- 统一公开命名：环境变量、集成测试、CI、构建常量和 Go 模块使用 PHEEBKIT / pheebkit。旧环境变量、数据目录、主题键和安装运行保护仅在隔离兼容层中保留；不删除、搬移或重写旧用户配置。
- 新增简体中文 / English 界面切换，首次按系统语言选择，之后保存本机偏好；切换不重载窗口、不重置项目、轨道、剪辑或参数。原生文件选择器标题及文件类型同步语言；Windows 自有按钮遵循系统语言。
- 界面与应用提示使用共享的本地翻译目录，不调用在线翻译。用户文件名、路径、字体、预设名称、FFmpeg 原始日志和帮助保持原样。存储不可用时保留本次切换并说明无法持久保存。
- 新增翻译覆盖、参数占位符和旧公开命名检查，并加入前端测试入口。设置页按需加载，控制新增界面功能的主包体积。
- 当前仍为稳定化候选；不将本机或托管 Windows Server 回归等同于干净 Windows 10/11 客户端全面验收。Android 暂缓。
- 轨道语言标签仅查询自有映射键；异常媒体标签（如 constructor）按原文显示，避免继承对象属性导致错误显示或界面异常。

## Previous candidate: 0.2.0-rc.2

- 统一 FFmpeg 环境提示、下载来源记录中的用户可见品牌名；保留当时的历史数据与环境变量键和安装标识以兼容旧配置与升级（rc.3 后由隔离层承接）。新增中英文 README 入口及真实 Windows 桌面运行截图。
- 环境验证保留取消/超时的真实原因，并显示最多 1 KiB 的工具错误详情；首次启动的有限验证预算由 8 秒调整为 30 秒。继续通过真实共享配置回归验证新路径下的启动行为，不跳过失败检查。

- 修复桌面最终二进制审计发现的 HTTP/2 间接依赖漏洞：使用官方修复版 `golang.org/x/net v0.60.0` 及其兼容依赖；最低 Go 语言版本随依赖升至 1.26，正式构建仍要求 Go 1.27.2+。打包和 checks-only CI 均对最终 EXE 运行漏洞扫描，阻止遗漏桌面构建标签依赖。未发布的旧 rc.2 本地产物移出发布目录保留，不作为可分发候选。

菲比工具箱 Windows x64 测试版；Android 暂缓。

- 修复首次托管 CI 发现的问题：PATH 专项测试明确使用已核验的测试工具；4 MiB 以内的源文件完整哈希，检测粗粒度文件时间戳下的同大小修改。大文件仍使用有界采样，不声称全内容完整性校验；旧指纹保持兼容，新任务使用新版指纹。
- 新增 `--build-info` 无窗口查询，打包与安装验收直接核对程序内的版本、源码修订、修改状态和框架版本；查询不会读取用户配置或启动媒体任务。rc.1 保留为被后续候选替代的本地产物，不作为稳定版发布。
- 稳定化候选：经用户明确批准，窗口框架从 Wails v3 Beta 迁移到固定稳定版 Wails v2.17.0，移除 v3 Go/npm 运行时。保留现有品牌界面、媒体核心、项目／草稿／配置格式与安装标识。
- 使用显式 58 方法桌面接口；内部生命周期与媒体资源处理器不暴露给界面。适配窗口控制、事件、文件拖放、原生对话框、媒体预览资源与关闭确认。记住的输出目录已丢失时仍可重新选择位置。
- 稳定版打包新增版本／源码修订匹配的验收记录门槛。RC 是验证候选；干净 Windows 10/11、无 WebView2、CPU-only、托管 CI 和公开源码／素材审查需要独立证据，详见 docs/stability.md。不会把缺少验收的构建标成稳定版。
- 修复设置页自动保存复选框被文本输入框样式撑大的问题；改为品牌配色的开关，文字与右侧控件对齐，支持点击整行、空格键和清晰的键盘焦点。标明保存编辑设置后生效，保存期间禁用开关。
- 文本输入样式排除复选框、单选框与滑块，避免同类污染。开关适配浅色／深色、窄屏、减少动画和系统高对比度模式。
- 修复降噪强度逐字输入小数时的交互：输入 0 后保持输入框，支持继续键入 0.5，失焦后规范显示。beta.10 为未交付的本地候选包，保留原产物。
- 新增可选声音处理：动态单遍响度归一化（目标 LUFS、真峰值、响度范围）、FFT 降噪（强度、噪声底）、最多 8 段均衡（频率、Q、增益）、淡入淡出、采样率与单声道／立体声转换。参数可自行调整，默认不启用。
- 压缩和音频提取均可使用。每条选中音轨独立处理，最多四轨；多片段拼接在拼接后统一处理，避免每段重复淡入淡出。复制／移除音频、非法参数、缺失滤镜或超过输出时长的淡入淡出会在执行前拒绝。
- 启用声音处理后默认输出 48 kHz，可调整；Opus 输出采用 48 kHz 解码时钟。保存前检查实际输出的采样率和请求的声道数。媒体信息同时显示已解析的源采样率与声道。
- 参数进入项目、草稿、预设、撤销／重做、批量和队列快照。样片使用自己的内容进行声音处理，不应用整片淡入淡出；原始整片参数仍会先校验。恢复损坏视频时清除声音处理，先恢复再处理声音。

响度归一化采用动态单遍模式，不等于两遍测量的线性归一化，也不保证编码后满足母带标准。大幅均衡／增益和强降噪可能降低听感，请试听确认。淡入淡出按所选输出时间线计时，不寻找最后一段可听声音；音轨短于视频／容器时间线时，淡出可能落在静音区间，可先提取音频再处理。

保留视频恢复、多片段导出、文字字幕烧录、GPU 检测与 CPU 回退、环境补全、诊断导出等能力。恢复输出仍须完整解码复检；普通声音处理任务使用结构及请求音频格式检查。

当前采用固定稳定 Wails v2.17.0；候选包未签名。本机已实际验证原生导入与 GPU 压缩；干净 Windows 10/11、CPU-only、托管 CI 及完整稳定版验收仍需独立证据。两遍响度归一化、混音／路由、图片／GIF、多源时间线、HDR、自由滤镜图、网络采集与 Android 不属于本次稳定化范围。

轻量版不打包 FFmpeg；内置版携带固定来源并通过校验的运行时。配置位于 AppData，卸载保留配置及媒体。源码仓库为 https://github.com/u-fw/PheebKit。
