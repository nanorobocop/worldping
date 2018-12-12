Worldping is educational project to scan Internet IPv4 addresses and check services availability.

Hosts checker is written in Go and implemented in distributed manner. 
It publishes scan data to central database.
Currently only ping check available.

Visualization of the results of scanning could be done on top of it. For example, using [Hiblert curve](https://en.wikipedia.org/wiki/Hilbert_curve).

## Technical Features

* Dynamically evaluated concurrency level based on Load Average
* Graceful shutdown (for saving unsubmitted results, closing connections)
* Dependencies managed by 'go mod' (https://github.com/golang/go/wiki/Modules)

## Performance

Performance during scan - is a main feature of this project.
Current version scans 10000 hosts in 1 sec and spinning up thousands goroutines:

```bash
2018-12-12 05:30:16 NOTICE Saving results to DB: total 10000, pinged 0, maxIP 153.123.73.163 (2574993827)
2018-12-12 05:30:17 NOTICE Saving results to DB: total 10000, pinged 0, maxIP 153.123.112.180 (2575003828)
2018-12-12 05:30:18 NOTICE Goroutines: 8786 (70754)
2018-12-12 05:30:19 NOTICE Saving results to DB: total 10000, pinged 0, maxIP 153.123.151.195 (2575013827)
2018-12-12 05:30:20 NOTICE Saving results to DB: total 10000, pinged 0, maxIP 153.123.190.211 (2575023827)
```

## Related links

* https://en.wikipedia.org/wiki/Hilbert_curve
* https://blog.benjojo.co.uk/post/scan-ping-the-internet-hilbert-curve (and Russian https://habr.com/post/353986/)
* https://github.com/measurement-factory/ipv4-heatmap
