# ADR-0003: Kalendertage als UTC-Mitternacht, Termine nur aus dem Startdatum

Status: accepted
Datum: 2026-10-30

## Kontext

Fälligkeiten sind Kalendertage ohne Uhrzeit: Die Miete fällig am 1. Oktober bedeutet einen Tag, keinen Zeitpunkt mit Uhrzeit. Werden solche Werte als Zeitpunkte mit Zeitzone gespeichert, entstehen Fehler an den Tagesgrenzen und bei Sommerzeitwechseln.

Zwei Fehlerquellen sind zu vermeiden:

- Der erste: Ein Termin, der aus dem vorherigen Termin abgeleitet wird, verschiebt sich mit jedem Monatsende. Ein monatlicher Eintrag vom 31.01. liefe so auf den 28.02. und dann auf den 28.03., statt zurück auf den 31.03. zu kommen.
- Der zweite: Die Differenz zweier Kalendertage hängt von der Zeitzone ab, wenn beide Zeitpunkte eine Uhrzeit tragen.

## Entscheidung

- Ein Kalendertag ist im Code ein `time.Time` auf 00:00:00 UTC ohne Zeitzonenzuordnung.
- In der Datenbank steht ein Kalendertag als Text im Format `YYYY-MM-DD`. `plan.ParseDate` und `plan.FormatDate` sind die einzigen Wege zwischen Text und Zeitstempel, der Store nutzt sie für alle Datumswerte.
- Jedes Datumsargument der Berechnung gilt als der Kalendertag seines Zeitstempels und wird auf 00:00:00 UTC normalisiert. Zwischen zwei Kalendertagen liegen damit immer genau 24 Stunden.
- Termine werden immer direkt aus dem Startdatum berechnet: der n-te Termin ist `Startdatum + n * Intervall`. `plan.Occurrences` und `plan.Next` teilen sich den Kern `nth`, in dem das passiert; `Next` greift direkt darauf zu. `plan.Reserve` und `plan.Timeline` rufen `plan.Occurrences` auf.
- Gibt es den Tag im Zielmonat nicht, wird der letzte Tag des Monats genommen. Das gilt auch im Schaltjahr.
- Unterstützt sind die Jahre 1 bis 9999. Die Berechnung braucht diese Grenze, damit Monats- und Tagesrechnung nirgends überlaufen.

## Konsequenzen

- Die Berechnung ist unabhängig von der Zeitzone und vom Aufrufzeitpunkt. Gleiche Eingabe, gleiches Ergebnis.
- Ein monatlicher Eintrag bleibt am selben Tag des Monats: 31.01.2026, 28.02.2026, 31.03.2026, 30.04.2026.
- Ein jährlicher Eintrag vom 29.02.2028 läuft am 28.02. der Folgejahre weiter und kommt am 29.02.2032 zurück, weil der Tag jedes Mal neu aus dem Startdatum abgeleitet wird.
- Das Plan-Paket bleibt rein: kein I/O, kein `time.Now()`. Ein Test im Paket prüft das.
- Wer Termine anzeigt, muss sie in der lokalen Zeitzone (`TZ`) formatieren. Das folgt mit der Oberfläche in Phase 2.
