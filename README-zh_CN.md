# 游戏存档同步工具

通过 S3 对象存储在多台计算机之间同步 Windows 游戏存档

## 目前支持的游戏

* 巫师3：狂猎
* 上古卷轴5：天际
* 新仙剑奇侠传 单机版
* 风色幻想3 罪与罚的镇魂歌
* 风色幻想4 圣战的终焉

## 支持的系统

* Windows 10

## 如何使用

* 在 config.ini 中填入你的 S3 存储配置
* 使用 go build 编译 gamesavesyncing.exe
* 运行 gamesavesyncing.exe