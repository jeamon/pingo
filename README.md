# PinGO

As a network champion from designing and implementing to troubleshooting large scale networks - I know that is usually not easy for administrators to quickly check basics reachability
statistics when troubleshooting or monitoring their IP-based infrastructure devices. PinGO is a standalone & lightweight & feature-rich tool to assist on Ping-based & Traceroute-based reachability checking tasks.
You can pipe input or provide a list of IP addresses at the program startup. And by just pressing `Enter` or `P` or <CTRL+P> key you spin up a customized Ping command on a given IP address then observe in real-time the statistics.
The same is achieved for Traceroute / Tracert feature. Use <CTRL+T> or `T` to spin up a traceroute toward a given IP address.


![overview of PinGO](https://github.com/jeamon/pingo/blob/master/cover.png?raw=true)


## Features / Goals


![key controls of PinGO](https://github.com/jeamon/pingo/blob/master/help.PNG?raw=true)


* pipe input a list of IP addresses or provide files to load from.
* auto filter and remove duplicated IP addresses from data provided.
* view in real-time the statistics of the ongoing Ping process.
* view any IP configuration when scrolling over the list of IPs. 
* per-IP config option to stream (on disk file) the ping outputs.

| Command | Description |
|:------ | :-------------------------------------- |
| CTRL+A | add and save new IP address to the list |
| CTRL+D | delete an IP address from the list |
| CTRL+E | edit a given IP address configs |
| CTRL+F | search an IP address and move focus on it |
| CTRL+L | load and add IP addresses from files |
| CTRL+Q | close help details or stop ongoing process |
| CTRL+P | initiate a Ping on the focused IP address |
| CTRL+R | clear the content of the outputs view |
| CTRL+T | initiate a Traceroute on the focused IP |
| CTRL+C | close immediately the whole program |
| F1 & Esc | display Help and close it respectively |
| Enter | initiate a Ping on the focused IP address |
| P | initiate a Ping toward the focused IP address |
| T | initiate a Traceroute toward the focused IP address |
| Tab | move focus between different views/sessions |
| ↕ & ↔ | navigate into the list of IP or line of outputs |
 

## Demo

Live preview on my youtube channel. coming soon.


## Installation

* **Download executables files**

Please check later on [releases page](https://github.com/jeamon/pingo/releases)

* **From source on windows**

```shell
$ git clone https://github.com/jeamon/pingo.git
$ cd pingo
$ go build -o pingo.exe .
```
* **From source on linux/macos**

```shell
$ git clone https://github.com/jeamon/pingo.git
$ cd pingo
$ go build -o pingo .
$ chmod +x ./pingo
```

## Getting started

* Start the tool with any available files containing a list of ip addresses 

```
$ type ip-list-00.txt | pingo.exe ip-list-01.txt ip-list-02.txt ip-list-03.txt 
```

```
$ cat ip-list-00.txt | ./pingo ip-list-01.txt ip-list-02.txt ip-list-03.txt
```

```
$ echo 127.0.0.1 | ./pingo ip-list-01.txt ip-list-02.txt ip-list-03.txt
```

## License

Please check & read [the license details](https://github.com/jeamon/pingo/blob/master/LICENSE) 


## Contact

Feel free to [reach out to me](https://blog.cloudmentor-scale.com/contact) before any action. Feel free to connect on [Twitter](https://twitter.com/jerome_amon) or [linkedin](https://www.linkedin.com/in/jeromeamon/)