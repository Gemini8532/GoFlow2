files in directories are generated with something like this:

 curl 'http://localhost:9090/flowuk?date=2025-11-14T13:00:00Z?num=20' | xargs cp -t 2025-11-14
