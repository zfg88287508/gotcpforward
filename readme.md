# Go tcp forward

大家不要被这个简单的名字糊弄了。什么tcpforward, 它还是一个威力强大的连接池子。
因为驱动tcp转发的机制，是基于连接池的。

因为是tcp协议通吃，所以只要是基于tcp的应用，比如postgres数据库，mysql数据库，redis数据库，等等各种tcp协议的应用，甚至可以是一个什么tcp游戏服务器，http协议服务器。
所有的只要是基于tcp的。它都能处理。

连接池，默认不开启。你可以手动加参数开启。

高级版有更多特性。【保活，主动链接，连接池动态扩容缩容，分布式等等】

Buy me a cup of coffee for $3

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/M4M54KKIF)

This application can forward tcp connection to upstream.

I use it to forward traffic to my kubernetes .

I use linux systemd to manage the tcp forward.

User request  => gotcpforward  =>   my http service api hosted inside k8s

## build

```
CGO_ENABLED=0 go build -v -a -ldflags ' -s -w  -extldflags "-static"' .
```

## use

```
  -l string
    	listen host:port
  -r string
    	remote host:port

```

Example use, forward local port to remote server port

```
./gotcpforward -l :3306 -r 10.1.23.43:3316
```


## License

```
    Copyright (C) 2000-2022 cnmade

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published
    by the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
