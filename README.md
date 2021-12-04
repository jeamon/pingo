# pingo

As a network champion from designing and implementing to troubleshooting large scale networks - I know that is usually not easy for administrators to quickly check basics reachability
statistics when troubleshooting or monitoring their IP-based infrastructures. Pingo is a standalone & lightweight & feature-rich tool to assist on Ping-based & Traceroute-based reachability checking tasks.
You can pipe input or provide a list of IP addresses at the program startup and by just pressing <Enter> or <P> or <CTRL+P> key spin up a customized Ping command on a given IP address then observe in real-time the statistics.
The same is acheived for Traceroute / Tracert feature. Use <CTRL+T> or <T> to spin up a traceroute toward a given IP address.



## Features / Goals

* pipe input a list of IP addresses or provide files to load from.
* auto filter and remove duplicated IP addresses from data provided.

* use keyboard (CTRL+A) to add & save new IP address to the list.
* use keyboard (CTRL+D) to delete an IP address from the list.
* use keyboard (CTRL+E) to edit a given IP address configs. 
* use keyboard (CTRL+F) to search an IP address and move focus on it.
* use keyboard (CTRL+L) to load and add IP addresses from files.
* use keyboard (CTRL+Q) to close help details or stop ongoing process.
* use keyboard (CTRL+P) to initiate a Ping on the focused IP address.
* use keyboard (CTRL+R) to clear the content of the outputs view.
* use keyboard (CTRL+T) to initiate a Traceroute on the focused IP.
* use keyboard (CTRL+C) to close immediately the whole program.

* use keyboard F1 & Esc to display Help and close it respectively.
* Press Enter key to initiate a Ping on the focused IP address.
* Press P key to initiate a Ping toward the focused IP address.
* Press T key to initiate a Traceroute toward the focused IP address.
* use Tab key to move focus between different views/sessions.
* use ↕ and ↔ to navigate into the list of IP or line of outputs.

* view in real-time the statistics of the ongoing Ping process.
* view any IP configuration when scrolling over the list of IPs. 
* per-IP config option to stream (on disk file) the ping outputs. 


## Demo

Preview on my youtube channel. coming soon.


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