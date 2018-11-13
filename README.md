Worldping is educational project to scan Internet IPv4 addresses and check services availability.

Hosts checker is written in Go and implemented in distributed manner. 
It publishes scan data to central database.
Currently only ping check available.

Visualization of the results of scanning could be done on top of it. For example, using [Hiblert curve](https://en.wikipedia.org/wiki/Hilbert_curve).

## Technical Features

* Dynamically evaluated concurrency level based on Load Average
* Graceful shutdown (for saving unsubmitted results, closing connections)
* Dependencies managed by 'go mod' (https://github.com/golang/go/wiki/Modules)

## Related links

* https://en.wikipedia.org/wiki/Hilbert_curve
* https://blog.benjojo.co.uk/post/scan-ping-the-internet-hilbert-curve (and Russian https://habr.com/post/353986/)
* https://github.com/measurement-factory/ipv4-heatmap
