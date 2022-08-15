# gamesave syncing

Syncing game save between PC via S3 or FTP

[简体中文文档](README-zh_CN.md)

## Supported games

* Devil May Cry 5
* The Witcher 3
* Shin Sangokumusou 7 TC
* Skyrim
* New Legend of Sword and Fairy
* The Legend of Sword and Fairy 2
* Wind Fantasy 2 aLive
* Wind Fantasy 3
* Wind Fantasy 4
* Wind Fantasy 5
* Wind Fantasy 6
* Wind Fantasy XX

## Supported OS

* Windows 10

## How to use

* Fill config.ini
* Generate gamesavesyncing.exe
```
$ cd cmd\gamesave-syncing
$ go build
```
* Run gamesavesyncing.exe

### Convert time format

If you use the early version of gamesave-syncing, you may need convert-time-format
to help you convert the time format of uploaded gamesave.

* Fill config.ini
* Generate convert-time-format.exe
```
$ cd cmd\convert-time-format
$ go build
```
* Run convert-time-format.exe

### config.ini

You can config gamesavesyncing.exe to use S3 or FTP to sync gamesave

#### S3 example

```ini
[s3]
endpoint = oss-cn-guangzhou.aliyuncs.com
bucketName = yourBucketName
accessKeyID = yourAccessKeyID
secretAccessKey = yourSecretAccessKey
```

#### FTP example
```ini
[ftp]
addr = 127.0.0.1:21
user = yourUsername
password = userPassword
subDir = yourSubdirToStoreGamesave
```

### conf.d

If your game not in the [Supported games](https://github.com/chenjianlong/gamesave-sync#supported-games)

You can add your game search info under conf.d directory

## TODO

* Support monitor game save directory and game process，sync game save while change
