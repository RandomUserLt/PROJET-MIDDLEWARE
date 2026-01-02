Lancement :

go run ./cmd/scheduler/main.go

Lancement du NATS : 

docker rm -f nats-server
docker run -d --name nats-server -p 4222:4222 nats -js


Subscribe au canal EVENTS :

nats sub "EVENTS.new" --raw



