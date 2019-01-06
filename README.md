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
2019-01-02 13:17:31 NOTICE Saving results to DB: total 32767, pinged 0, maxIP 243.225.18.153 (4091613849)
2019-01-02 13:17:32 NOTICE Goroutines: 375785 (12348300)
2019-01-02 13:17:34 NOTICE DB Goroutines: 1
2019-01-02 13:17:34 NOTICE Saving results to DB: total 32767, pinged 0, maxIP 243.225.156.113 (4091649137)
2019-01-02 13:17:46 NOTICE Goroutines: 371429 (12348600)
2019-01-02 13:17:48 NOTICE DB Goroutines: 1
2019-01-02 13:17:48 NOTICE Saving results to DB: total 32767, pinged 0, maxIP 243.225.233.215 (4091668951)
2019-01-02 13:17:51 NOTICE DB Goroutines: 1
2019-01-02 13:17:51 NOTICE Saving results to DB: total 32767, pinged 0, maxIP 243.226.72.93 (4091693149)
```

## Related links

* https://en.wikipedia.org/wiki/Hilbert_curve
* https://blog.benjojo.co.uk/post/scan-ping-the-internet-hilbert-curve (and Russian https://habr.com/post/353986/)
* https://github.com/measurement-factory/ipv4-heatmap
