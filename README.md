# GoRead

GoRead 是一个用 Golang 编写的终端/CLI EPUB 阅读器，受 [epr](https://github.com/wustho/epr) 启发。

## 特性

- 记住上次阅读的文件（直接运行 `goread` 无需参数）
- 记住每个文件的最后阅读状态（每个文件的状态保存在 `$HOME/.config/goread/config` 或 `$HOME/.goread`）
- 可调整文本区域宽度
- 适应终端大小调整
- 支持 EPUB3（不支持音频）
- 支持 vim 风格的按键绑定
- 支持打开图片（使用系统默认图片查看器）
- 深色/浅色配色方案（取决于终端颜色能力）
- 代码高亮

## Roadmap

- 支持 mobi

## 安装
🚧

## 使用方法

```
goread             读取上次阅读的 epub
goread EPUBFILE    读取指定的 EPUBFILE
goread STRINGS     从历史记录中读取匹配 STRINGS 的文件
goread NUMBER      从历史记录中读取编号为 NUMBER 的文件
```

## 选项

```
-r              打印阅读历史
-d              导出 epub 内容
-h, --help      打印帮助信息
```

## 按键绑定

```
帮助             : ?
退出             : q
向下滚动         : DOWN      j
向上滚动         : UP        k
向上半屏         : C-u
向下半屏         : C-d
向下翻页         : PGDN      RIGHT   SPC
向上翻页         : PGUP      LEFT
下一章           : n
上一章           : p
章节开始         : HOME      g
章节结束         : END       G
打开图片         : o
搜索             : /
下一个匹配项     : n
上一个匹配项     : N
切换宽度         : =
缩小             : -
放大             : +
目录             : TAB       t
元数据           : m
标记位置到 n     : b[n]
跳转到位置 n     : `[n]
切换配色方案     : [默认=0, 深色=1, 浅色=2]c
```

## 项目结构

```
goread/
├── cmd/
│   └── goread/       # 主程序入口
├── pkg/
│   ├── config/       # 配置管理
│   ├── epub/         # EPUB 解析
│   ├── parser/       # HTML 解析
│   └── ui/           # 用户界面
├── go.mod            # Go 模块定义
├── go.sum            # 依赖校验和
├── Makefile          # 构建脚本
└── README.md         # 项目说明
```

## 开发

### 依赖

- Go 1.16 或更高版本
- [tcell](https://github.com/gdamore/tcell) - 终端处理库
- [tview](https://github.com/rivo/tview) - 终端 UI 库

### 构建

🚧

## 许可证

MIT
