# splitbandwidth

将 CSV 中的汇总带宽按域名随机拆分为多条独立带宽曲线，同时生成可交互的离线 HTML 图表。

Split total bandwidth in a CSV file into random per-domain bandwidth curves, with an interactive offline HTML chart.

## Features / 功能

- 支持两种拆分模式：`profile`（各域名保持相对稳定的占比曲线）和 `independent`（每行独立随机）
- 输出单文件 CSV（所有域名合并为列）或多文件 CSV（每个域名一个文件）
- 自动生成 Grafana 风格的离线 HTML 图表（内嵌 ECharts，无需网络）
- 图表支持深色/浅色主题切换、图例多选独显/隐藏、时间轴缩放
- 可设置随机种子以复现结果

---

- Two split modes: `profile` (stable per-domain share curves) and `independent` (fully random per row)
- Output as a single merged CSV or multiple per-domain CSV files
- Auto-generates a Grafana-style offline HTML chart (ECharts embedded, no network needed)
- Chart supports dark/light theme toggle, legend multi-select isolate/hide, time-axis zoom
- Reproducible results via random seed

## Install / 安装

```bash
go install github.com/user/splitbandwidth@latest
```

Or build from source:

```bash
git clone https://github.com/user/splitbandwidth.git
cd splitbandwidth
go build -o splitbandwidth .
```

Cross-compile for macOS Apple Silicon:

```bash
GOOS=darwin GOARCH=arm64 go build -o splitbandwidth_darwin_arm64 .
```

## Usage / 用法

```bash
splitbandwidth <source.csv> <domains.txt> [flags]
```

### Examples / 示例

```bash
# 输出单文件 CSV + 图表
splitbandwidth traffic.csv domains.txt -o result.csv

# 指定随机种子和小数位数
splitbandwidth traffic.csv domains.txt -o result.csv --seed 42 --decimal-places 2

# 输出多文件（每个域名一个 CSV）
splitbandwidth traffic.csv domains.txt --output-dir split_output

# 不生成图表
splitbandwidth traffic.csv domains.txt -o result.csv --no-chart
```

### Flags / 参数

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output-file` | | 单文件输出路径 / Single output CSV path |
| `--output-dir` | `split_output` | 多文件输出目录 / Multi-file output directory |
| `--chart-file` | | HTML 图表路径 / HTML chart output path |
| `--no-chart` | `false` | 不生成图表 / Skip chart generation |
| `--bandwidth-col` | `B` | 带宽列（列名、列号或字母）/ Bandwidth column |
| `--seed` | random | 随机种子 / Random seed |
| `--decimal-places` | `0` | 输出小数位数 / Decimal places |
| `--mode` | `profile` | 拆分模式 / Split mode: `profile` or `independent` |
| `--domain-spread` | `1.2` | 域名间差距（仅 profile）/ Domain spread |
| `--volatility` | `0.18` | 时间波动（仅 profile）/ Time volatility |
| `--smoothness` | `0.98` | 曲线平滑度（仅 profile，< 1）/ Smoothness |

### Input Format / 输入格式

**source.csv** — 第一行为表头，需包含时间列和带宽列（默认第 B 列）：

```csv
Time,Bandwidth(bps)
2025-01-01 00:00:00,1000000000
2025-01-01 00:05:00,1200000000
```

**domains.txt** — 域名列表，空白/逗号/分号分隔：

```
cdn1.example.com
cdn2.example.com
cdn3.example.com
```

### Chart / 图表

生成的 HTML 图表完全离线可用，特性：

- 🌗 深色/浅色主题一键切换
- 🔍 Isolate 模式：单击图例选中域名（可多选），只高亮选中线，tooltip 只显示选中数据
- 🚫 Hide 模式：单击图例隐藏/显示指定线
- 📊 底部时间轴滑块 + 鼠标滚轮缩放
- 📐 Y 轴自动换算单位（bps → Kbps → Mbps → Gbps → Tbps，1000 进制）

## License

MIT
