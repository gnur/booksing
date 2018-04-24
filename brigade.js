const { events, Job } = require("brigadier");

events.on("push", function(e, project) {


  console.log("received push for commit " + e.commit)

  var node = new Job("test-runner")

  node.image = "golang:1.9.4-alpine3.7"

  node.tasks = [
    "mkdir -p /go/src/github.com/gnur/",
    "cp -a /src/ /go/src/github.com/gnur/booksing/",
    "cd /go/src/github.com/gnur/booksing/",
    "go get github.com/jteeuwen/go-bindata/...",
    "go get github.com/elazarl/go-bindata-assetfs/...",
    "go-bindata-assetfs static/...",
    "RUN go build -o app *.go"
  ]

  // We're done configuring, so we run the job
  node.run()
})
