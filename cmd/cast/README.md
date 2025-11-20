if the server is running in ../cmd/server/main.go
it is possible to run a test of the average generation like this:
  find ../../rainfall_data -name '*.png' | head -10 | xargs go run . -gridType=average -maxFeatures 1000 --maxAngle 0.3 --smoothness 0.1 -serverURL http://localhost:9093 -id=123

then there should be smoothed vectors avaiable via http://localhost:9093/average-flow-grid?id=123
these vectors are in the encoded png format

