"# Middleware" 

Pour exécuter : Lancer chaque api depuis son repertoire de base 
______________
CONFIG    : /A_CONFIG_OK/cmd/main.go      ; SWAGGER UI : http://localhost:8080/swagger/index.html 
TIMETABLE : /B_TIMETABLE_OK/cmd/timetable/main.go ; SWAGGER UI : http://localhost:8081/swagger/index.html 
SCHEDULER : /C_SCHEDULER_OK/cmd/scheduler/main.go
CONSUMER  : /D_CONSUMER_OK/cmd/consumer/main.go
ALERTER   : /E_ALERTER_OK/cmd/alerter/main.go 

REMARQUE : 
______________
A titre démonstratif, il a été choisi de laisser les nouveaux cours générer des alertes.
On a donc les cas suivants qui génèrent des alertes : event_changed et event_new.
Les cours modifiés créent également des alertes (testés avec curl car peu de modifications en temps réel sur les agendas fournis dans le tp)
Un agenda factice a été créé pour illustrer le traitement de modifications


VOIR DOSSIER ANNEXES_DEMO
______________

Autre : 
______________
Quelques problèmes de persistance avec les db 
