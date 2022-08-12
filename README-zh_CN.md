# 游戏存档同步工具

通过 S3 对象存储在多台计算机之间同步 Windows 游戏存档

## 目前支持的游戏

* 鬼泣5
* 巫师3：狂猎
* 真三国无双7 猛将传繁体中文版
* 上古卷轴5 天际
* 新仙剑奇侠传 单机版
* 仙剑奇侠传2
* 风色幻想2 aLive
* 风色幻想3 罪与罚的镇魂歌
* 风色幻想4 圣战的终焉
* 风色幻想5 赤月战争
* 风色幻想6 冒险奏鸣
* 风色幻想XX 交错的轨迹

## 支持的系统

* Windows 10

## 如何使用

* 填充 config.ini
* 使用 go build 编译 gamesavesyncing.exe
```
$ cd cmd\gamesave-syncing
$ go build
```
* 运行 gamesavesyncing.exe

### 转换时间格式
    
如果你有使用这个软件的早期版本，你可能需要使用 convert-time-format.exe 来帮助你
将已经上传的游戏存档的时间格式转换为新版本需要的格式
    
* 填充 config.ini
* 使用 go build 编译 convert-time-format.exe
```
$ cd cmd\convert-time-format
$ go build
```
* 运行 convert-time-format.exe

## 待完成

* 支持通过 FTP 服务端同步游戏存档
* 支持通过监控游戏存档所在的目录和游戏进程，当游戏存档发生修改并且探测到游戏进程退出后同步游戏存档
