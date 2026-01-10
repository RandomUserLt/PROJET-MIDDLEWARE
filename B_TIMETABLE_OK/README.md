# TP middleware example

## Run

Tidy / download modules :
```
go mod tidy
```
Build & run :
```
go run cmd/timetable/main.go
```
Swagger UI : 

swag init -g cmd/timetable/main.go
http://localhost:8081/swagger/index.html 
