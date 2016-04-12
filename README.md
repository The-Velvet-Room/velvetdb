# velvetdb [![Build Status](https://travis-ci.org/The-Velvet-Room/velvetdb.svg?branch=master)](https://travis-ci.org/The-Velvet-Room/velvetdb)

A database for player information.

### Setup

Dockerfiles and a docker-compose file are available if you're interested. Else,
install [Go](https://golang.org/) and [RethinkDB](http://rethinkdb.com/docs/install/).

On Mac/Linux, it should be enough to install [Docker Toolbox](https://docs.docker.com/mac/), clone the repo,
and run `docker-compose run --service-ports --rm web bash`. That'll drop you into a shell inside a container that has Go
installed, with a volume mounted for your code. After adding a config and running the server, you can get the IP of your
Docker host by running `docker-machine ip`. Then, you should be able to connect to port 3000 on that IP.

On Windows, it might be easier to avoid Docker for now, at least for running the Go app. Issues with shell
paths, container binding, and lack of an interactive mode in docker-compose make the process take a few more commands.
However, it can be convenient to install Docker Toolbox and use Docker or Kitematic to spin up a RethinkDB container, rather than
having to install RethinkDB locally.

If you're interested in getting it to work, feel free to file an issue with your progress for any questions that come up.

### Config

You'll need to supply a RethinkDB connection to start the app.
You can either create `config.json` file in the root directory,
or set environment variables. The `config.json` should look like this:

```
{
  "rethinkConnection": "",
  "challongeApiKey": "",
  "challongeDevUsername": "",
  "cookieKey": ""
}
```

If you're setting environment variables, prefix the above keys with `VELVETDB_`.
For example: `VELVETDB_RETHINKCONNECTION`.

### Running

You should be able to run the app either by starting the Docker container or by running `go run *.go`
from the root directory. Make sure to provide a RethinkDB connection string or the server won't start.
