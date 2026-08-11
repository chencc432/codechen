#!/usr/bin/env python3
"""Convert 03-transformer.md to a self-contained HTML with Mermaid + paper figures."""

import re
from pathlib import Path

import markdown
from markdown.extensions.tables import TableExtension
from markdown.extensions.fenced_code import FencedCodeExtension

ROOT = Path(__file__).resolve().parent.parent
MD_PATH = ROOT / "03-transformer.md"
HTML_PATH = ROOT / "03-transformer.html"
IMG_BASE = "images/transformer-paper"

PAPER_SECTION = """
## 3.2 从论文图出发：Transformer 系统全景

> 原论文：[Attention Is All You Need](https://arxiv.org/abs/1706.03762)（Vaswani et al., 2017）  
> 本节用**一个翻译例子**，对照论文 **Figure 1 / Figure 2**，让数据从输入一路流到输出，逐组件、逐步骤讲清楚——**每一步都标出张量形状**，你能清楚看到数据长什么样、怎么变。

为了和论文图完全对齐，本节统一用论文的尺寸（后面讲 Llama 时再换成 4096 那套）：

| 符号 | 含义 | 论文取值 |
|------|------|----------|
| `d_model` | 每个 token 的向量维度 | 512 |
| `N` | Encoder / Decoder 各自的层数 | 6 |
| `h` | 注意力头数 | 8 |
| `d_k = d_v` | 每个头的维度 = d_model / h | 64 |
| `d_ff` | FFN 中间层维度 | 2048 |

### 3.2.1 先建立直觉：这台机器在做什么

论文里的 Transformer 是为**机器翻译**设计的，我们全程用这个例子：

```
源语言（输入）:  "Je suis étudiant"   （法语：我是学生）
目标语言（输出）: "I am a student"
```

整台机器分两半（看 Figure 1：左半 + 右半）：

- **Encoder（左）**：把整句源语言**一次性读懂**，压缩成一组"语义向量"（论文叫 memory）。
- **Decoder（右）**：拿着这组语义向量，**一个词一个词**地把译文写出来。

> 现代大模型（GPT、Llama）只保留了 **Decoder 半边**（3.2.12 会讲）。但完整版能把所有零件都讲到，理解完整版后砍半边非常容易。

### 3.2.2 Figure 1：先把整机地图看一遍

![Figure 1: The Transformer - model architecture（论文原图）]({img}/figure1-transformer-architecture.png)

下面这张图是上面论文图的"数据流版本"，**每条边都标了形状**，先扫一眼建立全局印象，下面再逐站详解：

```mermaid
graph TD
    SRC["源句 'Je suis étudiant'"] --> TOK1["Tokenize<br/>→ ID [3]"]
    TOK1 --> EMB1["Input Embedding<br/>→ [3, 512]"]
    EMB1 --> PE1["+ Positional Encoding<br/>→ [3, 512]"]
    PE1 --> ENC["Encoder × 6 层<br/>(Self-Attn + FFN)"]
    ENC --> MEM["Encoder 输出 memory<br/>[3, 512]"]

    TGT["已生成译文 '&lt;bos> I am'"] --> TOK2["Tokenize<br/>→ ID [3]"]
    TOK2 --> EMB2["Output Embedding<br/>→ [3, 512]"]
    EMB2 --> PE2["+ Positional Encoding<br/>→ [3, 512]"]
    PE2 --> DEC["Decoder × 6 层<br/>(Masked Self-Attn + Cross-Attn + FFN)"]
    MEM -->|"K,V 来自 Encoder"| DEC
    DEC --> LIN["Linear + Softmax<br/>[3, 512] → [3, 词表大小]"]
    LIN --> NEXT["取最后一个位置<br/>→ 下一个词 'a'"]

    style ENC fill:#e3f2fd
    style DEC fill:#fff3e0
    style MEM fill:#c8e6c9
    style NEXT fill:#ffcdd2
```

**读图口诀：左边 Encoder 读句子，右边 Decoder 写译文，中间一条线（memory）把两边连起来。**

下面 3.2.3 ~ 3.2.10 就沿着这张图，**从上到下、一站一站**走，每站说清三件事：**这个组件做什么 / 数据形状怎么变 / 为什么需要它**。

---

### 3.2.3 站点①　Input Embedding：把文字变成向量

**位置**：Figure 1 左下角第一个框（"Input Embedding"）。

**做什么**：源句先被切成 token，再查一张大表，把每个整数 ID 换成一个 512 维向量。

```
"Je suis étudiant"
  ↓ Tokenize（切词 + 查词表）
ID:        [ 1024,  88,  377 ]          形状 [3]
  ↓ Input Embedding（查 [词表大小, 512] 的表，取出对应行）
向量:      [[0.1, -0.3, ...],           形状 [3, 512]
            [0.7,  0.2, ...],
            [-0.4, 0.9, ...]]
```

**数据形状**：`[3]`（3 个整数）→ `[3, 512]`（3 个向量，每个 512 维）。

**为什么需要**：整数 1024 和 1025 没有数值关系，但向量能让"意思相近的词，向量也相近"。这张表是**训练出来的**（详见 3.4）。

---

### 3.2.4 站点②　Positional Encoding：告诉模型词的先后顺序

**位置**：Figure 1 里 Embedding 上方那个 `+` 号。

**做什么**：给每个位置算一个 512 维的"位置向量"，**加**到对应的词向量上。

```
位置 0 的词向量  +  PE(0)  →  带位置信息的向量
位置 1 的词向量  +  PE(1)  →  带位置信息的向量
位置 2 的词向量  +  PE(2)  →  带位置信息的向量
              （逐元素相加，形状不变）
```

**数据形状**：`[3, 512]` + `[3, 512]` → `[3, 512]`（形状不变）。

**为什么需要**：后面的 Attention 计算**本身不区分顺序**（打乱 token 顺序，注意力结果一样）。不加位置信息，"我爱你"和"你爱我"在模型眼里就一样了。论文用 sin/cos 公式，Llama 改用 RoPE（详见 3.5）。

到这里，源句已经变成 `[3, 512]` 的矩阵，准备进入 Encoder。

---

### 3.2.5 站点③　Encoder Layer：读懂整句（重复 6 次）

**位置**：Figure 1 左侧那个大框（`Nx` 表示堆叠 6 层）。

一层 Encoder 内部只有 **2 个子模块**，每个后面跟一个 **Add & Norm**：

```mermaid
graph TD
    X["输入 [3, 512]"] --> SA["① Multi-Head<br/>Self-Attention"]
    SA --> AN1["Add & Norm<br/>x + Attn(x)"]
    X --> AN1
    AN1 --> FF["② Feed-Forward<br/>512→2048→512"]
    FF --> AN2["Add & Norm<br/>x + FFN(x)"]
    AN1 --> AN2
    AN2 --> OUT["输出 [3, 512]<br/>形状不变"]

    style SA fill:#e1f5fe
    style FF fill:#fff9c4
    style AN1 fill:#c8e6c9
    style AN2 fill:#c8e6c9
```

#### 子模块①：Multi-Head Self-Attention（让词互相交流）

- **Self（自）**：Q、K、V 全部来自源句自己。
- **没有 Mask**：每个词能看到**整句所有词**（前后都行），因为 Encoder 的任务是"读懂"，不是"预测下一个"。

```
"étudiant" 的新向量 ≈ 0.6·V(suis) + 0.3·V(Je) + 0.1·V(étudiant)
→ "étudiant" 吸收了"主语是 Je、动词是 suis"的上下文
```

形状：输入 `[3, 512]` → 输出 `[3, 512]`（每个词换成"看过全句后"的新表示）。内部机制见 3.2.7。

#### Add & Norm（残差 + 归一化）

```
输出 = LayerNorm( x + SelfAttention(x) )
       └─ 残差：把原始 x 加回来，防止信息丢失、梯度好传
       └─ LayerNorm：把数值幅度拉回稳定范围
```

形状不变：`[3, 512]` → `[3, 512]`。

#### 子模块②：Feed-Forward Network（每个词独立深加工）

对**每个位置独立**做一次"先扩维再缩回"：

```
[512] --W1--> [2048] --ReLU--> [2048] --W2--> [512]
       先放大，让模型有空间做复杂非线性变换，再压回原大小
```

形状：`[3, 512]` → `[3, 512]`（注意：位置之间不交互，3 个词各算各的）。

#### 一层的总结

输入 `[3, 512]` → 输出 `[3, 512]`，**形状完全不变**，所以能直接堆 6 层。
6 层走完，得到 **memory `[3, 512]`**——源句的最终语义表示，交给 Decoder 用。

---

### 3.2.6 站点④　Decoder Layer：逐词生成译文（重复 6 次）

**位置**：Figure 1 右侧大框。比 Encoder **多一个 Cross-Attention**，所以有 **3 个子模块**：

```mermaid
graph TD
    X["输入 [t, 512]<br/>已生成的译文"] --> MSA["① Masked<br/>Self-Attention"]
    MSA --> AN1["Add & Norm"]
    X --> AN1
    MEM["Encoder memory<br/>[3, 512]"] --> CA["② Cross-Attention<br/>Q=译文, K/V=memory"]
    AN1 --> CA
    CA --> AN2["Add & Norm"]
    AN1 --> AN2
    AN2 --> FF["③ Feed-Forward"]
    FF --> AN3["Add & Norm"]
    AN2 --> AN3
    AN3 --> OUT["输出 [t, 512]"]

    style MSA fill:#ffcdd2
    style CA fill:#e1f5fe
    style FF fill:#fff9c4
    style AN1 fill:#c8e6c9
    style AN2 fill:#c8e6c9
    style AN3 fill:#c8e6c9
```

#### 子模块①：Masked Self-Attention（只能看已经写出来的词）

和 Encoder 的 Self-Attention 一样，但**加了因果 Mask**：

```
正在写第 t 个词时，只允许看 第 0 ~ t 个位置，未来的词被遮成 -∞
原因：生成时未来的词还不存在，不能偷看答案
```

形状：`[t, 512]` → `[t, 512]`。

#### 子模块②：Cross-Attention（Encoder 和 Decoder 之间的桥）⭐

**这就是 Figure 1 中间那条线**，也是整张图最容易忽略、却最关键的一步：

| 角色 | 来自哪里 | 直觉 |
|------|----------|------|
| **Q（查询）** | Decoder 自己（当前译文状态） | "我接下来想表达什么？" |
| **K（键）** | **Encoder memory** | 源句每个词的"索引标签" |
| **V（值）** | **Encoder memory** | 源句每个词的"实际内容" |

```
Decoder 当前位置的 Q  ·  Encoder 三个词的 K  →  谁最相关？
  写 "am" 时，Q 和 "suis" 的 K 最匹配
  → 取出 "suis" 的 V，混进当前向量
  → 完成 "法语 suis → 英语 am" 的对齐
```

形状：Q 是 `[t, 512]`，K/V 是 `[3, 512]`（来自 memory），输出 `[t, 512]`。
**没有 Mask**——译文可以参考源句的全部词。

#### 子模块③：Feed-Forward

和 Encoder 的 FFN 完全一样，逐位置 `512→2048→512`。

6 层 Decoder 走完，输出 `[t, 512]`，交给最后的输出层。

---

### 3.2.7 Figure 2：拆开 Attention 内部（所有 Attention 框的通用零件）

上面反复出现"Multi-Head Attention"。它内部到底怎么算？这就是论文 **Figure 2** 的两张图——**左图是单个零件，右图是把零件并联**。

**左图 — Scaled Dot-Product Attention（一次注意力）**

![Figure 2 left]({img}/figure2-scaled-dot-product-attention.png)

一句公式，五个步骤：

```
Attention(Q, K, V) = softmax( Q·Kᵀ / √d_k ) · V

① MatMul   : Q·Kᵀ          算每对词的相似度    [t,64]·[64,3] = [t,3]
② Scale    : ÷ √d_k(=8)    防止数值过大        [t,3]
③ Mask     : 未来位置→ -∞  仅 Decoder 自注意力  [t,3]
④ Softmax  : 每行归一化     变成 0~1 的权重      [t,3]
⑤ MatMul   : 权重 · V       按权重混合内容       [t,3]·[3,64] = [t,64]
```

**右图 — Multi-Head Attention（8 个零件并联）**

![Figure 2 right]({img}/figure2-multi-head-attention.png)

```
输入 [t, 512]
  ↓ 用 8 套不同的小矩阵投影，切成 8 个头，每头 64 维
  Q,K,V 各 → [8 头, t, 64]
  ↓ 8 个头各自做一次上面的 Scaled Dot-Product Attention（并行）
  → [8 头, t, 64]
  ↓ Concat 拼回 [t, 512]
  ↓ 再过一个 W_O [512,512] 混合各头信息
输出 [t, 512]
```

**为什么要多头**：一个头只能学一种关系（比如指代）。8 个头能同时学语法、指代、位置等多种关系，最后汇总。详见 3.7。

> 记住：Figure 1 里**每一个**写着 Multi-Head Attention 的框，内部都是这张右图。区别只在于 **Q/K/V 从哪来、要不要 Mask**（见 3.2.9 对照表）。

---

### 3.2.8 站点⑤　Linear + Softmax：从向量选出下一个词

**位置**：Figure 1 最顶部（"Linear" + "Softmax"）。

```
Decoder 输出 [t, 512]
  ↓ Linear（W [512, 词表大小]）        → [t, 词表大小]   每个词一个分数
  ↓ 只取最后一个位置                    → [词表大小]
  ↓ Softmax                            → [词表大小] 概率分布
  ↓ 取概率最大的词（或采样）            → 下一个词，比如 "a"
```

**数据形状**：`[t, 512]` → `[t, 词表大小]`，推理时只关心最后一行。

---

### 3.2.9 数据怎么"流动"：自回归生成的循环 ⭐

前面是"一遍前向"，但 Decoder 真正生成译文是**一个循环**：每生成一个词，把它接到已生成序列后面，再跑一遍 Decoder，直到输出结束符 `<eos>`。

```mermaid
graph LR
    A["输入 '&lt;bos>'"] --> B["Decoder + 输出层"]
    B --> C["生成 'I'"]
    C --> D["输入 '&lt;bos> I'"]
    D --> E["Decoder + 输出层"]
    E --> F["生成 'am'"]
    F --> G["输入 '&lt;bos> I am'"]
    G --> H["..."]
    H --> I["生成 '&lt;eos>' 停止"]

    style C fill:#ffcdd2
    style F fill:#ffcdd2
    style I fill:#c8e6c9
```

**完整数据流（Encoder 只跑一次，Decoder 循环跑）**：

| 轮次 | Decoder 输入 | Cross-Attn 看的 memory | 输出层选出 |
|------|--------------|------------------------|-----------|
| 1 | `<bos>` | `[3,512]`（"Je suis étudiant"） | `I` |
| 2 | `<bos> I` | 同上（不变） | `am` |
| 3 | `<bos> I am` | 同上 | `a` |
| 4 | `<bos> I am a` | 同上 | `student` |
| 5 | `<bos> I am a student` | 同上 | `<eos>` → 停 |

**关键观察**：
- **Encoder 的 memory 只算一次**，整个生成过程反复复用——这正是 Cross-Attention 的 K、V。
- **Decoder 每轮都把序列加长 1**，重复计算前面的词——这就是 **KV Cache 要优化的地方**（详见 3.11）。

---

### 3.2.10 三种 Attention 一张表彻底分清

整张图里出现了 3 处 Attention，名字像、算法同，区别只在 **Q/K/V 来源**和 **Mask**：

| 类型 | 出现位置 | Q 来自 | K / V 来自 | Mask？ | 作用 |
|------|----------|--------|-----------|--------|------|
| Encoder Self-Attn | Encoder ① | Encoder | Encoder | 无 | 源句内部互相理解 |
| Masked Self-Attn | Decoder ① | Decoder | Decoder | **有** | 已生成译文自看（不偷看未来）|
| Cross-Attn | Decoder ② | Decoder | **Encoder memory** | 无 | 译文对齐源句 |

记忆法：**Self = Q/K/V 同源；Cross = Q 是译文、K/V 是源句；带 Mask 的只有 Decoder 的自注意力。**

---

### 3.2.11 现代 LLM（GPT / Llama）对这张图做了什么改动

GPT、Llama 这类生成式模型**砍掉了整个 Encoder**，只留 Decoder，并去掉了 Cross-Attention：

| 论文原版（Figure 1） | GPT / Llama | 为什么 |
|----------------------|-------------|--------|
| Encoder + Decoder | **只有 Decoder** | 不做翻译，没有"源句"要编码 |
| Cross-Attention | **删除** | 没有 Encoder memory 可查 |
| sin/cos 位置编码 | **RoPE** | 长度外推更好 |
| Post-Norm | **Pre-Norm + RMSNorm** | 深层训练更稳 |
| ReLU FFN | **SwiGLU** | 效果更好 |
| MHA（KV 头 = Q 头） | **GQA**（KV 头更少） | 省 KV Cache 显存（3.8）|

所以一个 **Llama Block** 就是 Figure 1 右侧 Decoder Layer **去掉 Cross-Attention** 的简化版：

```mermaid
graph TD
    X["输入 [seq, 4096]"] --> N1["RMSNorm"]
    N1 --> A["Masked Multi-Head<br/>Self-Attention (GQA)"]
    A --> R1["+ 残差"]
    X --> R1
    R1 --> N2["RMSNorm"]
    N2 --> F["SwiGLU FFN<br/>4096→11008→4096"]
    F --> R2["+ 残差"]
    R1 --> R2
    R2 --> OUT["输出 [seq, 4096]"]

    style A fill:#e1f5fe
    style F fill:#fff9c4
    style R1 fill:#c8e6c9
    style R2 fill:#c8e6c9
```

> 这台简化机器就是后面 3.3 ~ 3.12 的主角。下面开始把每个零件拆到最细。

---

""".replace("{img}", IMG_BASE)


def insert_paper_section(md_text: str) -> str:
    """Insert paper architecture section after 3.1, renumber following sections."""
    marker = "接下来我们一步步详解。\n\n---\n\n## 3.2 第一步"
    if marker not in md_text:
        return md_text

    head, tail = md_text.split(marker, 1)
    tail = tail.replace("## 3.2 第一步", "## 3.3 第一步", 1)
    # Renumber sections 3.3->3.4 ... 3.11->3.12 (high to low)
    for old, new in [
        ("### 3.11.", "### 3.12."),
        ("## 3.11 ", "## 3.12 "),
        ("### 3.10.", "### 3.11."),
        ("## 3.10 ", "## 3.11 "),
        ("## 3.9 ", "## 3.10 "),
        ("## 3.8 ", "## 3.9 "),
        ("## 3.7 ", "## 3.8 "),
        ("## 3.6 ", "## 3.7 "),
        ("### 3.5.", "### 3.6."),
        ("## 3.5 ", "## 3.6 "),
        ("## 3.4 ", "## 3.5 "),
        ("## 3.3 第二步", "## 3.4 第二步"),
    ]:
        tail = tail.replace(old, new)

    return head + "接下来我们一步步详解。**建议先读 3.2 论文架构全景，再拆零件。**\n\n---\n\n" + PAPER_SECTION + "---\n\n## 3.3 第一步" + tail[len("## 3.3 第一步"):]


def preprocess_mermaid(md_text: str) -> str:
    """Replace mermaid fenced blocks with placeholders, then restore as HTML divs."""
    blocks = []
    pattern = re.compile(r"```mermaid\n(.*?)```", re.DOTALL)

    def replacer(match):
        blocks.append(match.group(1).strip())
        return f"@@MERMAID_{len(blocks) - 1}@@"

    processed = pattern.sub(replacer, md_text)
    return processed, blocks


def restore_mermaid(html: str, blocks: list[str]) -> str:
    # Replace from high index first to avoid MERMAID_1 matching inside MERMAID_10
    for i in range(len(blocks) - 1, -1, -1):
        code = blocks[i].replace("&", "&amp;").replace("<", "&lt;")
        div = f'<div class="mermaid">{code}</div>'
        token = f"@@MERMAID_{i}@@"
        html = html.replace(f"<p>{token}</p>", div)
        html = html.replace(token, div)
    return html


HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>模块3：Transformer 架构详解</title>
  <style>
    :root {
      --bg: #fafafa;
      --card: #ffffff;
      --text: #1a1a2e;
      --muted: #5c5c7a;
      --accent: #2563eb;
      --accent-light: #e8f0fe;
      --border: #e2e8f0;
      --code-bg: #f1f5f9;
    }
    * { box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
        "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
      line-height: 1.75;
      color: var(--text);
      background: var(--bg);
      margin: 0;
      padding: 0;
    }
    .layout {
      display: grid;
      grid-template-columns: 260px 1fr;
      min-height: 100vh;
    }
    nav#toc {
      background: var(--card);
      border-right: 1px solid var(--border);
      padding: 1.5rem 1rem;
      position: sticky;
      top: 0;
      height: 100vh;
      overflow-y: auto;
      font-size: 0.85rem;
    }
    nav#toc h2 {
      font-size: 0.95rem;
      margin: 0 0 1rem;
      color: var(--muted);
    }
    nav#toc a {
      display: block;
      color: var(--text);
      text-decoration: none;
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
      line-height: 1.4;
    }
    nav#toc a:hover { background: var(--accent-light); color: var(--accent); }
    nav#toc .toc-h3 { padding-left: 1rem; font-size: 0.8rem; color: var(--muted); }
    main {
      max-width: 900px;
      padding: 2rem 2.5rem 4rem;
    }
    h1 { font-size: 1.85rem; border-bottom: 2px solid var(--accent); padding-bottom: 0.5rem; }
    h2 {
      font-size: 1.4rem;
      margin-top: 2.5rem;
      padding-top: 1rem;
      border-top: 1px solid var(--border);
      color: #0f172a;
    }
    h3 { font-size: 1.15rem; margin-top: 1.5rem; color: #334155; }
    blockquote {
      background: var(--accent-light);
      border-left: 4px solid var(--accent);
      margin: 1rem 0;
      padding: 0.75rem 1rem;
      border-radius: 0 6px 6px 0;
      color: #1e3a5f;
    }
    code {
      background: var(--code-bg);
      padding: 0.15em 0.4em;
      border-radius: 4px;
      font-size: 0.9em;
      font-family: "SF Mono", "Fira Code", Consolas, monospace;
    }
    pre {
      background: #1e293b;
      color: #e2e8f0;
      padding: 1rem 1.25rem;
      border-radius: 8px;
      overflow-x: auto;
      line-height: 1.5;
    }
    pre code {
      background: none;
      color: inherit;
      padding: 0;
      font-size: 0.85rem;
    }
    table {
      border-collapse: collapse;
      width: 100%;
      margin: 1rem 0;
      font-size: 0.92rem;
    }
    th, td {
      border: 1px solid var(--border);
      padding: 0.5rem 0.75rem;
      text-align: left;
    }
    th { background: #f8fafc; font-weight: 600; }
    tr:nth-child(even) { background: #f8fafc; }
    img {
      max-width: 100%;
      height: auto;
      display: block;
      margin: 1rem auto;
      border: 1px solid var(--border);
      border-radius: 8px;
      background: white;
      padding: 8px;
    }
    .figure-caption {
      text-align: center;
      color: var(--muted);
      font-size: 0.9rem;
      margin-bottom: 1.5rem;
    }
    .mermaid {
      background: white;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1rem;
      margin: 1rem 0;
      text-align: center;
    }
    hr { border: none; border-top: 1px solid var(--border); margin: 2rem 0; }
    ul, ol { padding-left: 1.5rem; }
    li { margin: 0.35rem 0; }
    @media (max-width: 900px) {
      .layout { grid-template-columns: 1fr; }
      nav#toc { position: relative; height: auto; max-height: 40vh; }
      main { padding: 1.5rem; }
    }
  </style>
</head>
<body>
  <div class="layout">
    <nav id="toc"><h2>目录</h2>{toc}</nav>
    <main>{content}</main>
  </div>
  <script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
    mermaid.initialize({ startOnLoad: true, theme: 'neutral', securityLevel: 'loose' });
  </script>
</body>
</html>
"""


def build_toc(html: str) -> str:
    items = []
    for m in re.finditer(r"<h([23]) id=\"([^\"]+)\">(.*?)</h[23]>", html):
        level, id_, title = m.group(1), m.group(2), re.sub(r"<[^>]+>", "", m.group(3))
        cls = "toc-h3" if level == "3" else ""
        items.append(f'<a class="{cls}" href="#{id_}">{title}</a>')
    return "\n".join(items)


def main():
    md_raw = MD_PATH.read_text(encoding="utf-8")
    md_text = insert_paper_section(md_raw)
    md_text, mermaid_blocks = preprocess_mermaid(md_text)

    md = markdown.Markdown(
        extensions=[TableExtension(), FencedCodeExtension()],
        extension_configs={"markdown.extensions.fenced_code": {"lang_prefix": ""}},
    )
    body_html = md.convert(md_text)
    body_html = restore_mermaid(body_html, mermaid_blocks)

    # Markdown may wrap block elements in <p> — fix invalid nesting
    body_html = re.sub(
        r"<p>\s*(<div class=\"mermaid\">.*?</div>)\s*</p>",
        r"\1",
        body_html,
        flags=re.DOTALL,
    )
    body_html = re.sub(
        r"<p>\s*(<figure>.*?</figure>)\s*</p>",
        r"\1",
        body_html,
        flags=re.DOTALL,
    )

    # Add ids to headings for TOC
    def add_heading_id(match):
        tag, title = match.group(1), match.group(2)
        id_ = re.sub(r"[^\w\u4e00-\u9fff]+", "-", title).strip("-").lower()
        id_ = re.sub(r"-+", "-", id_)
        return f'<h{tag} id="{id_}">{title}</h{tag}>'

    body_html = re.sub(r"<h([23])>(.*?)</h[23]>", add_heading_id, body_html)

    # Wrap img alt as caption hint
    body_html = re.sub(
        r"<img alt=\"([^\"]+)\" src=\"([^\"]+)\" />",
        r'<figure><img alt="\1" src="\2" /><figcaption class="figure-caption">\1</figcaption></figure>',
        body_html,
    )
    # Unwrap invalid <p><figure>...</figure></p> produced by the step above
    body_html = re.sub(
        r"<p>\s*(<figure>.*?</figure>)\s*</p>",
        r"\1",
        body_html,
        flags=re.DOTALL,
    )

    toc = build_toc(body_html)
    html = HTML_TEMPLATE.replace("{toc}", toc).replace("{content}", body_html)
    HTML_PATH.write_text(html, encoding="utf-8")
    print(f"Written: {HTML_PATH} ({HTML_PATH.stat().st_size // 1024} KB)")


if __name__ == "__main__":
    main()
