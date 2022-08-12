# gamesave syncing

Syncing game save between PC via S3

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


## TODO

* Support sync via FTP server
* Support monitor game save directory and game process，sync game save while change
