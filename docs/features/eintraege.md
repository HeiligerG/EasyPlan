# Einträge

Ein Eintrag ist eine Ausgabe, die zu einem Konto gehört und sich nach einem Rhythmus wiederholt. Die Seite `/entries` zeigt alle Einträge, gruppiert nach Konto, und bietet das Anlegen, das Bearbeiten und das Löschen an.

## Die Liste

Die Liste zeigt für jedes Konto eine Gruppe. Die Reihenfolge der Gruppen ist die Reihenfolge der Konten, also die Reihenfolge, die auf der Seite [Konten](konten.md) eingestellt ist.

- Ein deaktiviertes Konto bleibt in der Liste und ist mit dem Merkmal „deaktiviert“ gekennzeichnet. Seine Einträge bleiben lesbar.
- Ein Konto ohne Einträge bekommt eine leere Gruppe mit dem Hinweis „Dieses Konto hat noch keine Einträge.“.
- Innerhalb einer Gruppe stehen die Einträge nach Startdatum und Kennung.

Jede Zeile zeigt Name, Betrag, Rhythmus, Startdatum, Enddatum und den Status. Ein Klick auf den Namen öffnet das Formular zum Bearbeiten; der Link überdeckt die ganze Zeile, ein Klick irgendwo in der Zeile öffnet dasselbe Formular.

### Der Rhythmus

Der Rhythmus wird in Worten angezeigt:

| Eingabe | Anzeige |
|---|---|
| `once` | einmalig am 12.11.2026 |
| `week`, Intervall 1 | wöchentlich |
| `week`, Intervall 2 | alle 2 Wochen |
| `month`, Intervall 1 | monatlich |
| `month`, Intervall 3 | alle 3 Monate |
| `year`, Intervall 1 | jährlich |
| `year`, Intervall 2 | alle 2 Jahre |

Die Regeln für die Einheiten und Intervalle stehen unter [Wiederholungen](wiederholungen.md).

### Die Merkmale erledigt und beendet

Ein Eintrag kann zwei Merkmale tragen:

- **erledigt**: Ein einmaliger Eintrag, dessen Datum in der Vergangenheit liegt, oder ein wiederkehrender Eintrag, dessen Enddatum vor dem heutigen Tag liegt.
- **beendet**: Der Eintrag kommt nicht mehr vor. Das ist der Fall, wenn sein Enddatum vor dem heutigen Tag liegt.

Ein einmaliger Eintrag in der Vergangenheit ist also erledigt, aber nicht beendet. Beendete Einträge zählen nicht im Dashboard, in der Reserve und in der Timeline, sie bleiben aber in dieser Liste sichtbar.

## Das Formular

Das Formular liegt auf eigener Seite, `/entries/new` für einen neuen Eintrag und `/entries/{id}` für das Bearbeiten. Es funktioniert ohne JavaScript.

| Feld | Bedeutung | Vorgabe |
|---|---|---|
| `account_id` | Das Konto des Eintrags | das erste aktive Konto |
| `name` | Die Bezeichnung des Eintrags | leer |
| `amount` | Der Betrag einer Fälligkeit in Franken | leer |
| `start_date` | Der erste Termin des Eintrags | der heutige Tag |
| `unit` | Die Einheit des Rhythmus | `month` |
| `interval` | Die Anzahl Einheiten zwischen zwei Terminen | 1 |
| `ends_on` | Der letzte Termin des Eintrags | leer |
| `notes` | Eine Notiz zum Eintrag | leer |

### Die Wahl des Kontos

- Beim Anlegen werden nur aktive Konten angeboten.
- Beim Bearbeiten kommt das aktuelle Konto des Eintrags dazu, auch wenn es deaktiviert ist. So bleibt der Eintrag auf seinem Konto, ohne dass ein deaktiviertes Konto neu gewählt werden muss.
- Ein Konto, das nicht existiert, wird mit „Dieses Konto gibt es nicht.“ abgelehnt.
- Ein deaktiviertes Konto, das nicht angeboten wird, wird mit „Dieses Konto ist deaktiviert.“ abgelehnt. Die Prüfung passiert serverseitig, auch bei einem präparierten Formular.

### Der Rhythmus einmalig

Bei der Einheit `once` ignoriert der Plan das Intervall und das Enddatum. Die Web-Schicht setzt deshalb das Intervall auf 1 und leert das Enddatum, bevor der Eintrag gespeichert wird.

## Die Prüfung der Eingaben

Die Prüfung passiert vollständig serverseitig. Eine Eingabe mit Fehlern wird mit dem Status 422 beantwortet, das Formular erscheint erneut mit den Werten, die eingegeben wurden, und mit einer Fehlermeldung beim betroffenen Feld.

| Grenze | Wert | Meldung |
|---|---|---|
| Name | höchstens 100 Zeichen | Der Name darf höchstens 100 Zeichen lang sein. |
| Notiz | höchstens 1000 Zeichen | Die Notiz darf höchstens 1000 Zeichen lang sein. |
| Betrag | grösser als 0, höchstens 10 Milliarden Franken | Der Betrag muss grösser als 0 sein. / Der Betrag ist zu gross. |
| Intervall | mindestens 1, höchstens 10'000 | Das Intervall muss mindestens 1 sein. |
| Startdatum | Jahr 1 bis 9999, Format `JJJJ-MM-TT` | Das Startdatum muss im Format JJJJ-MM-TT stehen. |
| Enddatum | nicht vor dem Startdatum | Das Enddatum darf nicht vor dem Startdatum liegen. |

Die Attribute `maxlength` und `min`/`max` des Formulars nennen dem Browser die Grenzen für Name, Notiz und Intervall. Sie sind nur eine Hilfe, die Prüfung des Servers gilt. Für den Betrag, das Startdatum und das Enddatum prüft nur der Server.

Ein Formular, dessen Body grösser als 64 KiB ist, wird mit dem Status 413 abgelehnt. Die Oberfläche liest keine Daten aus der Query-Zeichenkette, sondern nur aus dem Body.

## Das Speichern

Nach dem Speichern antwortet der Server mit dem Status 303 und der Adresse der Liste. Das Muster heisst Post/Redirect/Get: der Browser folgt der Weiterleitung und sendet das Formular nicht noch einmal. Nach einem Neuladen ist die Formularansicht weg, das Formular wird also nicht noch einmal gesendet.

## Das Löschen

Das Löschen ist zweistufig und kommt ohne JavaScript aus:

1. Auf der Seite zum Bearbeiten führt der Link „Eintrag löschen“ zur Bestätigungsseite `/entries/{id}/delete`.
2. Die Bestätigungsseite nennt den Namen des Eintrags. Erst der Button „Ja, löschen“ löscht ihn.

Die Bestätigungsseite ist ein normales Formular mit dem Ziel `/entries/{id}/delete`. Nach dem Löschen antwortet der Server wieder mit 303 und der Adresse der Liste.

## Weiterführend

- [Wiederholungen](wiederholungen.md): Einheiten, Intervalle und Monatsende
- [Konten](konten.md): die Konten der Einträge
- [Formate](formate.md): Beträge und Daten in Eingabe und Anzeige
