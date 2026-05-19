# splitbandwidth

将 CSV 中的汇总带宽按域名随机拆分为多条独立带宽曲线，同时生成可交互的离线 HTML 图表。

Split total bandwidth in a CSV file into random per-domain bandwidth curves, with an interactive offline HTML chart.

## What it does / 做了什么

假设你有一份汇总带宽数据（只有一条总线）：

Say you have a total bandwidth CSV (single aggregated line):

**输入 / Input: `traffic.csv`**

```csv
Time,Bandwidth(bps)
2025-01-01 00:00:00,10000000000
2025-01-01 00:05:00,12000000000
2025-01-01 00:10:00,9500000000
2025-01-01 00:15:00,11000000000
```

以及一份域名列表：

And a domain list:

**`domains.txt`**

```
cdn1.example.com
cdn2.example.com
cdn3.example.com
```

运行 / Run:

```bash
splitbandwidth traffic.csv domains.txt -o result.csv
```

**输出 / Output: `result.csv`**

```csv
Time,Bandwidth(bps),cdn1.example.com,cdn2.example.com,cdn3.example.com
2025-01-01 00:00:00,10000000000,4521000000,3287000000,2192000000
2025-01-01 00:05:00,12000000000,5394000000,3978000000,2628000000
2025-01-01 00:10:00,9500000000,4298000000,3105000000,2097000000
2025-01-01 00:15:00,11000000000,4987000000,3612000000,2401000000
```

每行各域名之和 = 原始 Total，且各域名的占比曲线在时间维度上保持相对稳定（`profile` 模式）。

Each row's per-domain values sum to the original Total. In `profile` mode, each domain maintains a relatively stable share over time.

同时自动生成 `result.html` 交互图表：

An interactive `result.html` chart is also generated:

![Dark Theme](screenshots/dark.png)
![Light Theme](screenshots/light.png)

## Chart Features / 图表特性

- 🌗 深色/浅色主题切换 (Dark/Light theme toggle)
- 🔍 Isolate 模式：单击图例多选独显，tooltip 只显示选中线 (Multi-select isolate)
- 🚫 Hide 模式：单击图例隐藏指定线 (Click legend to hide lines)
- 📊 时间轴缩放 (Time-axis zoom with slider & mouse wheel)
- 📐 Y 轴自动换算单位 (Auto unit: bps/Kbps/Mbps/Gbps/Tbps, 1000-based)
- 📡 完全离线，无需网络 (Fully offline, no network needed)

## Install / 安装

### Download Binary / 下载可执行文件

从 [Releases](https://github.com/stanhui/splitbandwidth/releases) 下载对应平台的可执行文件：

| Platform | File |
|----------|------|
| Linux x86_64 | `splitbandwidth_linux_amd64` |
| Linux ARM64 | `splitbandwidth_linux_arm64` |
| macOS Intel | `splitbandwidth_darwin_amd64` |
| macOS Apple Silicon | `splitbandwidth_darwin_arm64` |
| Windows x86_64 | `splitbandwidth_windows_amd64.exe` |

### Build from Source / 从源码编译

```bash
go install github.com/stanhui/splitbandwidth@latest
```

Or:

```bash
git clone https://github.com/stanhui/splitbandwidth.git
cd splitbandwidth
go build -o splitbandwidth .
```

## Usage / 用法

```bash
splitbandwidth <source.csv> <domains.txt> [flags]
```

### More Examples / 更多示例

```bash
# 指定随机种子（可复现）+ 保留2位小数
splitbandwidth traffic.csv domains.txt -o result.csv --seed 42 --decimal-places 2

# 输出多文件（每个域名一个 CSV）
splitbandwidth traffic.csv domains.txt --output-dir split_output/

# 不生成图表
splitbandwidth traffic.csv domains.txt -o result.csv --no-chart

# 独立随机模式（每行独立分配，不保持占比稳定）
splitbandwidth traffic.csv domains.txt -o result.csv --mode independent
```

### Flags / 参数

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output-file` | | 单文件输出路径 / Single output CSV path |
| `--output-dir` | `split_output` | 多文件输出目录 / Multi-file output directory |
| `--chart-file` | | HTML 图表路径 / HTML chart output path |
| `--no-chart` | `false` | 不生成图表 / Skip chart generation |
| `--bandwidth-col` | `B` | 带宽列（列名、列号或字母）/ Bandwidth column |
| `--seed` | random | 随机种子 / Random seed for reproducibility |
| `--decimal-places` | `0` | 输出小数位数 / Decimal places |
| `--mode` | `profile` | 拆分模式: `profile` or `independent` |
| `--domain-spread` | `1.2` | 域名间差距（仅 profile）/ Domain spread |
| `--volatility` | `0.18` | 时间波动（仅 profile）/ Time volatility |
| `--smoothness` | `0.98` | 曲线平滑度（仅 profile，< 1）/ Smoothness |

### Split Modes / 拆分模式

| Mode | Description |
|------|-------------|
| `profile` | 各域名保持相对稳定的占比，曲线平滑，模拟真实 CDN 流量分布 / Each domain maintains a stable share over time, simulating real CDN traffic |
| `independent` | 每行完全独立随机分配，占比波动大 / Fully random per row, high variance |

## Release / 发布

打 tag 后 GitHub Actions 自动编译并发布所有平台的可执行文件：

```bash
git tag v1.0.0
git push origin v1.0.0
```

## License

MIT
