# Thrash

Goals:

- Mimic apache benchmark
- Use yml to specify a list of URLs with a percentage for frequency
- Measurements:
  - Min/Max/Avg/StdDev time per request
  - Requests/sec
  - endpoint distribution (for multiple endpoints)

```
Usage of ./thrash:
  -c int
    	The number of concurrent requests to execute. (default 1)
  -config string
    	File path to a configuration file (yml) (default "./config.yml")
  -n int
    	The total number of requests to make.
```

## Example

Create a simple HTTP server for you to run some traffic to:

```
const http = require('http');

const hostname = '127.0.0.1';
const port = process.env.PORT || 3000;

const server = http.createServer((req, res) => {
  // console.log(`${req.method} ${req.url}`);
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/plain');
  res.setHeader('Connection', 'close');
  res.end('Hello World\n');
});

server.listen(port, hostname, () => {
  console.log(`Server running at http://${hostname}:${port}/`);
});
```

This server will respond to any path from the URL. The following config will cause thrash to generate requests for a few paths on the server:

```
root: "http://localhost:8000"
endpoints:
  - path: "/"
    freq: 30.0
  - path: "/foo"
    freq: -1.0
  - path: "/bar"
    freq: -1.0
  - path: "/qaz"
    freq: -1.0
  - path: "/qux"
    freq: -1.0
  - path: "/foobar"
    freq: -1.0
```

Build thrash:

```
go build
```

Run thrash:

```
thrash -n 100000 -c 100
```

The default behavior is to read configuration from `config.yml` in the current directory, and you might get some output like this:

```
 ./thrash -n 100000 -c 100
Starting thrashing: 2019-01-30 22:11:33.049620102 -0500 EST m=+0.002806439
Completed requests: 10000	791.157283ms	Req/Sec: 12644.986
Completed requests: 20000	1.555646669s	Req/Sec: 13080.576
Completed requests: 30000	2.328958511s	Req/Sec: 12931.615
Completed requests: 40000	3.106834258s	Req/Sec: 12855.687
Completed requests: 50000	3.885333387s	Req/Sec: 12845.311
Completed requests: 60000	4.663884083s	Req/Sec: 12843.918
Completed requests: 70000	5.433084021s	Req/Sec: 13001.056
Completed requests: 80000	6.196982304s	Req/Sec: 13091.259
Completed requests: 90000	6.964151104s	Req/Sec: 13036.144
Completed requests: 100000	7.727599332s	Req/Sec: 13098.913

Request Summaries
=================================

http://localhost:8000/
  Count:   29752
  Success: 100 %
  Min:     0.9151600000000001 ms
  Max:     25.194368 ms
  Avg:     0s ms
  Freq:    29.999998 %

http://localhost:8000/foo
  Count:   14003
  Success: 100 %
  Min:     0.470042 ms
  Max:     27.246095 ms
  Avg:     0s ms
  Freq:    14.000002 %

http://localhost:8000/bar
  Count:   13988
  Success: 100 %
  Min:     0.62174 ms
  Max:     24.585356 ms
  Avg:     0s ms
  Freq:    14.000002 %

http://localhost:8000/qaz
  Count:   14070
  Success: 100 %
  Min:     0.236469 ms
  Max:     21.162565 ms
  Avg:     0s ms
  Freq:    14.000002 %

http://localhost:8000/qux
  Count:   13823
  Success: 100 %
  Min:     1.026119 ms
  Max:     25.167557 ms
  Avg:     0s ms
  Freq:    14.000002 %

http://localhost:8000/foobar
  Count:   14364
  Success: 100 %
  Min:     0.676328 ms
  Max:     23.483106 ms
  Avg:     0s ms
  Freq:    14.000002 %

Result Summary
=================================
  Req Count:   100000
  Concurrency: 100
  Duration:    7.727616414s
  Success:     100 %
  Req/Sec:     12940.600910111376
```