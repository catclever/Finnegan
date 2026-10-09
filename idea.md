# Finnegan v0.1: 赛博夜间意识流沙盒 (The Consciousness Stream Sandbox)

**定位**: CyberMemex 架构下的核心“卡片流转与分发”子系统
**核心理念**: Local-First (本地优先) / 打破目录暴政 / 无缝桥接联邦网络 (AT Protocol)

---

## 1. 核心哲学 (Philosophy)

传统的笔记软件是静态的“树”，而 Finnegan 是一片流动的“海”。它摒弃了强结构的文件夹分类，将信息划分为两种基本状态：
* **散落 (The Scattering)**：所有灵感、外来资讯均作为独立的原子碎片，悬浮在无层级的卡片箱中。
* **编织 (The Weaving)**：不移动物理文件，仅通过标签、元数据和虚拟视图，将散落的卡片动态串联成线性的长文或逻辑网。

## 2. 三层系统架构 (System Architecture)

### 2.1 数据源层 (Single Source of Truth)
* **载体**：完全基于物理硬盘的纯文本 Markdown 文件。
* **结构**：极致扁平化。所有文件堆叠于单一物理目录（如 `~/Finnegan/`），以时间戳或哈希值命名（例：`20260601_idea.md`）。
* **生产力终端**：在 M2 Max 主力机上，通过定制版的 VS Code 环境（极简亮灰排版 + Git 版本控制）直接对底层文件进行硬核重构。

### 2.2 响应式 GUI 层 (The Reactive Engine)
* **状态管理**：前端深度集成 **RxDB**，实现毫秒级的 Local-First 数据响应。
* **监听机制**：本地后台服务（File Watcher）实时监控物理目录。一旦 VS Code 或其他编辑器触发文件保存，变动瞬间推入 RxDB，GUI 界面完成无刷新的瀑布流更新。
* **视觉呈现**：弱化文件树，采用桌面平铺视角的“散落卡片”视图。支持多端无缝流转（如在 iPad 上通过离线 PWA 进行直觉化的卡片拖拽编织）。

### 2.3 网络吞吐层 (The AT Protocol Bridge)
将本地数字大脑与外部联邦网络（以 Bluesky 为代表）彻底打通。

* **📥 捕获流 (Inflow)**：
  * 在 PVE 边缘节点部署监听服务，订阅特定的 AT Protocol Feed 或关键词。
  * 将高价值网络内容自动“降维”抓取为纯文本，打上 `#External` 标签，静默空投至本地硬盘的 `/Inbox` 目录，化作 Finnegan 中的一张待处理卡片。
* **📤 分发流 (Outflow)**：
  * 卡片自带 YAML 头部元数据（Frontmatter）。
  * 当本地卡片被标记为 `publish: true` 时，CI/CD 推流脚本自动触发，调用 AT Protocol API，将该 Markdown 卡片静默发布至广域网。
  * **数据主权**：无论网络分发多远，该信息的“绝对本体”永远安全地锁在本地物理硬盘中。

### 2.4 全球意识流雷达：Bugu (不孤)
“德不孤，必有邻”。基于大语言模型与 SearxNG 构建的“跨文化/深网探针”，用于对本地卡片的观点进行全网回声捕获。

*   **原子化检索矩阵 (The Atomic Target Registry)**：摒弃层级分类，将全网的搜索目标拆解为扁平的“原子（Targets）”。
    *   **类型定义**：分为具体网站 (`site`)、特殊子引擎 (`engine`, 如 marginalia、wiby)、建站指纹 (`footprint`, 如 inurl:forum)。
    *   **配置属性**：每个原子拥有独立支持的语言区 (`languages`，如 `["ru", "en"]` 或 `["global"]`)，以及一组基于母语（中文）的语义标签 (`tags`，如 `["技术", "独立博客", "复古"]`)。
*   **语义与标签路由 (Semantic Tag Routing)**：
    *   **LLM 职责（降负）**：大模型仅需根据用户的卡片观点，输出“多语种搜索词（Semantic Queries）”和对应的“请求标签/语言（Requested Tags & Languages）”。
    *   **系统拼接（确定性）**：本地脚本根据大模型输出的 Tags，从原子库中精准匹配交集，动态拼装为 SearxNG 的最终检索语句。实现“零幻觉”的深网打捞。

## 3. 工作流演练 (The Workflow)

1. **游牧态 (Nomad Mode)**：在移动端或外部网络，随时捕捉信息，流入 `~/Finnegan/00_Inbox/`。
2. **意识流态 (Finnegan Mode)**：在 GUI 中，通过搜索和标签，将不同来源的卡片平铺在桌面，寻找神经元级的微弱关联。
3. **架构师态 (Ulysses Mode)**：打开 VS Code，使用分屏视图，将筛选出的散落卡片“织”成结构严密的长篇文档。
4. **广播态 (Broadcast Mode)**：修改头部元数据，一键将思考结晶推流至联邦宇宙。

---
*“在 Finnegan 的黄昏大雾中，卡片彻底松散、循环和流转，直至数字大脑的黎明到来。”*