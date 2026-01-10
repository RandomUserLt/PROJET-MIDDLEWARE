Ces screens permettent d'illustrer un test sur la réception de mails sur un agenda factice. 

Cet agenda a été au préalable via l'api config, ainsi qu'une alerte en lien. 

Il s'agit de l'agenda n°9 visible dans la liste des agendas (VOIR DOSSIER CREATION AGENDAS/LISTE_DES_AGENDAS.png)

On écoute les canaux EVENTS et ALERTS dans différents terminaux, et on poste des alertes avec curl pour chacun des cas suivants :

Nouvel Evenement

Changement de salle

Changement d'horaire

Changement de titre (ce n'est pas peut-etre pas pertinent irl mais c'est interessant à tester)


Remarque :
Le choix de considérer les nouveaux évènements crée du spam mais a été laissé à titre illustratif car en terme de modifications en temps réel, il semblait ne pas y avoir assez d'activité sur les vrais agendas. 
