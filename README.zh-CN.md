# URL Bridge

[English](README.md) | [简体中文](README.zh-CN.md)

URL Bridge 可以把 Windows 10/11 虚拟机变成一个轻量级的浏览器桥接器：当虚拟机里的代码工具、聊天工具、文档或其他应用打开 `http://` 或 `https://` 链接时，URL 会被转发给宿主机上的服务，再由宿主机的默认浏览器打开。

## 仓库内容

- `urlbridge-host`：运行在宿主机上的跨平台服务，支持 Windows、Linux 和 macOS。
- `urlbridge-browser.exe`：Windows URL 处理器，用于注册 `http` 和 `https`。
- `urlbridge-guestctl.exe`：Windows 侧辅助工具，用于安装、状态检查和卸载访客端集成。

## 为什么有两个 Windows 二进制

真正接管链接的浏览器处理器使用 GUI 进程构建，这样在虚拟机里点击链接时不会弹出控制台窗口。控制器则保留为普通命令行程序，方便安装和诊断。

## 工作方式

1. 在宿主机上运行 `urlbridge-host`。
2. 在 Windows 虚拟机内运行 `urlbridge-guestctl.exe install`，可以显式传入 `--host-url`，如果宿主机开启了鉴权，也可以额外传入 `--token`。
3. Windows 会打开默认应用设置页。
4. 将 `HTTP` 和 `HTTPS` 的处理程序设置为 `URL Bridge`。
5. 之后在虚拟机内点击 URL 时，请求会发送给宿主机服务，并由宿主机浏览器打开。

## 自动发现

URL Bridge 支持访客端自动发现宿主机：

- 宿主机可以在 UDP `38496` 端口响应发现请求。
- Windows 访客端还会探测常见虚拟机宿主地址，例如 `10.0.2.2`、`10.0.3.2`，以及当前默认网关。

## Windows 侧的重要行为说明

在 Windows 10/11 上，第三方应用不能静默替换用户的默认浏览器设置。因此 URL Bridge 会先把自己注册为候选处理程序，再打开系统默认应用设置页，让用户显式把 `HTTP` 和 `HTTPS` 指向它。

这个行为参考了微软文档：

- <https://learn.microsoft.com/en-us/windows/win32/shell/default-programs>
- <https://learn.microsoft.com/en-us/windows/apps/develop/launch/launch-default-apps-settings>

## 构建

```bash
make fmt
make test
make build-all
```

本地构建产物会输出到 `dist/`。

## 配置文件

URL Bridge 现在支持宿主端和 Windows 访客端都使用 `config.yaml`。在没有显式传入 `--config PATH` 时，会按下面顺序查找配置文件：

1. 当前工作目录
2. 可执行文件所在目录
3. 平台默认路径

默认路径如下：

- Windows 宿主端：`%LOCALAPPDATA%\URLBridgeHost\config.yaml`
- Windows 访客端：`%LOCALAPPDATA%\URLBridge\config.yaml`
- Linux 宿主端：`$XDG_CONFIG_HOME/urlbridge/config.yaml`，未设置时回退到 `~/.config/urlbridge/config.yaml`，最后再查 `/etc/urlbridge/config.yaml`
- macOS 宿主端：`~/Library/Application Support/URLBridge/config.yaml`，然后是 `/etc/urlbridge/config.yaml`

配置优先级固定为：内建默认值，其次 `config.yaml`，最后是命令行里显式传入的参数。

宿主端配置示例：

```yaml
listen_addr: "0.0.0.0:38495"
token: "YOUR_TOKEN"
discovery: true
log_path: "/path/to/host.log" # 可选；设为 "" 可关闭文件日志
```

访客端配置示例：

```yaml
host_base_url: "http://10.0.2.2:38495/"
token: "YOUR_TOKEN"
request_timeout_seconds: 3
browser_path: "C:/Program Files/Google/Chrome/Application/chrome.exe" # 可选
```

## 宿主机使用方式

如果你希望虚拟机到宿主机的转发过程带鉴权，可以先生成一个 token。下面以 Linux x64 为例：

```bash
./dist/urlbridge-host-linux-amd64 token
```

用与你宿主机平台匹配的二进制启动服务：

```bash
./dist/urlbridge-host-linux-amd64 --listen 0.0.0.0:38495 --token YOUR_TOKEN
```

如果你已经写好了配置文件，也可以直接这样启动：

```bash
./dist/urlbridge-host-linux-amd64 --config ./config.yaml
```

说明：

- `0.0.0.0` 适合虚拟机通过虚拟网卡而不是本机回环访问宿主机服务的场景。
- 服务会根据宿主机的 IPv4 网卡打印可供访客端使用的候选 URL。
- 默认会启用 UDP `38496` 上的自动发现；如需关闭，可传 `--discovery=false`。
- 宿主端日志始终会输出到 stdout，默认还会额外写入文件；如需关闭文件日志，可设 `log_path: ""`。
- 宿主端默认日志路径分别是：Windows 的 `%LOCALAPPDATA%\URLBridgeHost\host.log`，Linux 的 `$XDG_STATE_HOME/urlbridge/host.log` 或 `~/.local/state/urlbridge/host.log`，macOS 的 `~/Library/Logs/URLBridge/host.log`。
- Linux 上会优先使用 `xdg-open`，然后回退到 `gio open`。
- macOS 上使用 `open`，Windows 上使用系统默认的 URL 处理器。

### 宿主机安装脚本

在仓库根目录执行：

```bash
./scripts/install-host.sh
```

如果宿主机是 Windows：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-host.ps1
```

这两个安装脚本都会：

- 自动识别正确的宿主机二进制
- 安装到当前用户目录
- 写入或复用 `config.yaml`
- 如果你没有显式提供 token，会自动生成
- 安装后立即启动服务
- 在支持的平台上尽量配置用户级自启动
- 通过 `--config <path>` 启动宿主端

## 访客端使用方式

将下面两个文件复制到 Windows 虚拟机：

- `dist/urlbridge-browser.exe`
- `dist/urlbridge-guestctl.exe`

如果访客虚拟机是 Windows on ARM，请改用 `dist/urlbridge-browser-arm64.exe` 和 `dist/urlbridge-guestctl-arm64.exe`。

显式安装示例：

```powershell
.\urlbridge-guestctl.exe install --host-url http://10.0.2.2:38495 --token YOUR_TOKEN
```

也可以让虚拟机自动发现宿主机后完成安装：

```powershell
.\urlbridge-guestctl.exe install
```

在正式安装前，你也可以先查看虚拟机当前能发现哪些宿主机：

```powershell
.\urlbridge-guestctl.exe discover
```

常见宿主机地址示例：

- VirtualBox NAT 模式下，宿主机通常可通过 `http://10.0.2.2:38495` 访问。
- Host-only 或 bridged 网络模式下，通常应使用宿主机真实的局域网 IP 或 host-only IP。
- 宿主机操作系统类型并不重要，关键是虚拟机能通过网络访问宿主机服务。

安装完成后：

- Windows 会打开通用的默认应用设置页，以兼容 Windows 10 和 Windows 11。
- 把 `HTTP` 和 `HTTPS` 都设置为 `URL Bridge`。
- 如果在配置超时内无法连接宿主机，访客端会回退到本地浏览器。`browser_path` 为空时，URL Bridge 会按 Chrome、Edge 的顺序自动探测，成功打开后把所选浏览器写回配置，后续直接复用。
- 如果已保存的 `browser_path` 之后无法启动，URL Bridge 会重新探测本地浏览器，并在替换配置前弹出确认提示。

常用访客端命令：

```powershell
.\urlbridge-guestctl.exe install --config .\config.yaml --host-url http://10.0.2.2:38495
.\urlbridge-guestctl.exe discover
.\urlbridge-guestctl.exe discover --config .\config.yaml
.\urlbridge-guestctl.exe status
.\urlbridge-guestctl.exe status --config .\config.yaml
.\urlbridge-guestctl.exe open-default-settings
.\urlbridge-guestctl.exe uninstall
```

### 访客端安装脚本

可以在仓库根目录运行，也可以在打包后的访客端 bundle 中运行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-guest.ps1
```

访客端安装脚本默认会把配置写到 `%LOCALAPPDATA%\URLBridge\config.yaml`，并在注册 URL handler 时显式带上 `--config`。它会保留已有的 `browser_path`，并默认使用 3 秒的宿主机请求超时；如果 `browser_path` 仍为空，第一次真正发生本地浏览器回退时会自动探测并缓存。如果你希望完全显式配置，也可以继续传 `-HostUrl`、`-Token` 或 `-ConfigPath`。

## 当前限制

- URL Bridge 只转发 `http` 和 `https`。
- 它不会代理虚拟机里的 cookies、会话状态或浏览器用户配置。
- 宿主机服务仍要求虚拟机能够通过所选虚拟网络访问宿主机。
- 在 Windows 10/11 上，最终的默认应用绑定仍需要用户手动确认。
- UDP 自动发现是否可用，取决于虚拟机网络模式；如果广播被阻断，请改用可访问的 NAT 地址，例如 `10.0.2.2`，并显式传入 `--host-url`。
