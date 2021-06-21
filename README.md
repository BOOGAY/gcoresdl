# gcoredl

用于批量下载用户已经购买的机核有声书 MP3 文件，并添加正确的 ID3 信息和封面

## 安装方法

### 下载编译好的二进制文件

访问 https://github.com/go-guoyk/gcoredl/releases 下载编译好的二进制文件，请依照当前操作系统选择正确的文件

### 自己编译

1. 安装 Golang
2. 执行 `git clone https://github.com/go-guoyk/gcoredl.git` 克隆代码
3. 执行 `go build` 进行编译

## 使用方法

1. 在电脑网页端打开机核网，并登录自己的账户
2. 点开计划下载的有声书，比如《克苏鲁神话III》，此时可以在浏览器链接栏看到该有声书的编号为 `64`
3. 打开浏览器的开发者工具，找到 `Cookie` 中，`auth_token` 的值，比如 `xxxxxxxxx`

   **注意，这个值代表着你的用户身份，任何情况下请勿告诉其他人**
   
4. 执行该命令行工具

   ```shell
   # Linux 和 macOS
   ./gcoredl -album 64 -auth xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   
   # Windows
   .\gcoredl.exe -album 64 -auth xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

最后，你会得到一个文件夹 `output-64`，里面有完整有声书的 `MP3` 文件，并且每个 `MP3` 文件都包含正确的 ID3 标签，直接放入播放器，即可获得完整的专辑，标题，封面展示。

## 免责声明

1. 本方法基于未公开的 API，未来官方 API 变动可能导致该方法失效

2. 本方法无法获取到用户未购买的有声书，如果你想听某个有声书，请乖乖掏钱

## Credits

Guo Y.K., MIT License
