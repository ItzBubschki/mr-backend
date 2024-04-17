this is the backend for ScreenSociety
it acts as a wrapper around the move api i am using
it also handles the firebase cloud messaging notifications

it runs on my oracle cloud instance at backend.screensociety.de
it is built locally using docker and the pushed to the github registry

steps to deploy new version
```shell
# copy folder to oracle cloud instance
# connect to oci
docker build -t ghcr.io/itzbubschki/mr-backend/movie-rater-backend:latest .
docker push ghcr.io/itzbubschki/mr-backend/movie-rater-backend:latest
docker kill mr-backend
docker rm mr-backend
docker run -d --name mr-backend -p 8010:8080 --network mr-backend ghcr.io/itzbubschki/mr-backend/movie-rater-backend:latest
```

steps to run locally:
```shell
# start docker and mongo
go run main/main.go --mongoHost=localhost
```

future todos:
- maybe use go client library: https://github.com/movieofthenight/go-streaming-availability
