# Thrash

Goals:

- Mimic apache benchmark
- Use yml to specify a list of URLs with a percentage for frequency
- Measurements:
  - Min/Max/Avg/StdDev time per request
  - Requests/sec
  - endpoint distribution (for multiple endpoints)
  


```yml
endpoints:
  -
    url: http://localhost:9001/
    freq: 30.0
  -
    url: http://localhost:9001/tech
    freq: 10.0
  -
    url: http://localhost:9001/science
    freq: 10.0
  -
    url: http://localhost:9001/pop-culture
    freq: 10.0
  -
    url: http://localhost:9001/news
    freq: 10.0
  -
    url: http://localhost:9001/business
    freq: 10.0
```
