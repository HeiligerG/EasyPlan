# Changelog

Alle Änderungen an EasyPlan stehen in diesem Dokument. Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/), die Versionierung folgt [SemVer](https://semver.org/lang/de/).

Diese Datei ist die einzige Quelle für den Changelog. Eine zweite Kopie im Repository-Root gibt es nicht.

## [Unreleased]

### Added

- Bedarf pro Konto und gesamt: Monat, Woche und Tag, abgeleitet aus den Jahreskosten der Einträge.
- Reserve pro Konto und gesamt für die nächsten N Tage. Jede Fälligkeit im Zeitraum zählt, auch wenn derselbe Eintrag mehrfach vorkommt.
- Nächste Zahlung pro Konto: der früheste Termin aller aktiven Einträge, der heutige Tag eingeschlossen.
- Wiederholungen in den Einheiten einmalig, wöchentlich, monatlich und jährlich, mit einem Intervall bis 10'000 Einheiten.
- Monatsende-Regel mit Schaltjahr: ein monatlicher Eintrag vom 31.01. läuft am 28.02. weiter und kommt am 31.03. zurück.
- Ende eines Eintrags über ein optionales Enddatum, das am Termin selbst noch gilt.
- SQLite-Datenspeicher mit eingebetteten Migrationen, Fremdschlüsseln und WAL-Modus.
- Standardkonten Lifestyle, Sparen und Steuern entstehen mit der ersten Migration. Ein gelöschtes Standardkonto kommt nicht zurück.
- Der Start bricht mit einer klaren Meldung ab, wenn die Datenbank ein neueres Schema hat als das Binary unterstützt.
- Zwei gleichzeitige Starts auf einer bereits angelegten Datenbankdatei legen die Tabellen nicht doppelt an. Beim ersten Start auf einer brandneuen Datei kann sporadisch `SQLITE_BUSY` auftreten.
- Die Eingaben werden geprüft: Beträge bis 10^12 Rappen (10 Milliarden Franken), Datumswerte in den Jahren 1 bis 9999. Das Enddatum liegt nie vor dem Startdatum.
- Kontoverwaltung im Datenspeicher: anlegen, umbenennen, sortieren, deaktivieren und löschen, solange das Konto keine Einträge hat. Die Oberfläche bietet nur das Deaktivieren an.
- Dokumentation mit MkDocs: Überblick, Architektur, Berechnung, Wiederholungen, Datenspeicher, Oberfläche, Dashboard, Einträge, Timeline, Konten, Formate, Server und Betrieb, Sicherheit, ADRs und Changelog.
- Web-Oberfläche mit vier Seiten: Dashboard, Einträge, Timeline und Konten.
- Dashboard mit einer Tabelle pro aktivem Konto und einem Total: Bedarf pro Monat, Woche und Tag, Reserve und nächste Zahlung.
- Die Reserve lässt sich über 7, 30, 60 oder 90 Tage wählen. Die Auswahl steht in der Adresse (Query-Parameter `n`) und wird nicht gespeichert.
- Das Dashboard zeigt die nächste Zahlung pro Konto: den frühesten Termin aller aktiven Einträge ab dem heutigen Tag. Fällt der Betrag auf den gleichen Tag, gewinnt der kleinere Name, bei gleichen Namen die kleinere Kennung.
- Einträge: Liste nach Konto, Anlegen, Bearbeiten mit einem Klick auf die Zeile und Löschen mit Bestätigungsseite. Alles funktioniert ohne JavaScript.
- Konten: anlegen, umbenennen, in der Reihenfolge verschieben, aktivieren und deaktivieren.
- Timeline: die Zahlungen der nächsten zwölf Monate, nach Monat gruppiert, mit Summe je Monat und einem Filter auf das Konto.
- Die Prüfung der Formulare passiert auf dem Server. Ein Fehler erscheint als Meldung beim betroffenen Feld, die Antwort hat den Status 422.
- Das Löschen eines Eintrags ist zweistufig: erst der Link auf der Seite, dann die Bestätigung.
- Nach jedem Speichern antwortet der Server mit Status 303 und der Adresse der Liste, damit ein Neuladen die Eingabe nicht wiederholt.
- Beträge werden in ganzen Franken angezeigt, kaufmännisch gerundet, mit dem Apostroph als Tausendertrenner.
- In der Eingabe sind `'` und Leerzeichen Tausendertrenner, `.` und `,` sind Dezimaltrenner. `1'200` und `1.200,50` sind gültig.
- Der Server startet mit `EASYPLAN_DATA=./data go run ./cmd/easyplan` und legt die Datei `easyplan.db` im Datenordner an.
- Die Zeitzone kommt aus `TZ` (Standard `Europe/Zurich`). Die Daten der Zeitzonenbank sind im Binary eingebettet, `TZ` wirkt also auch im Container.
- Die Antworten tragen die Header `Content-Security-Policy`, `X-Content-Type-Options` und `Referrer-Policy`.
- CSS und HTMX liegen als eingebettete Dateien im Binary. Es gibt keinen Build-Schritt fürs Frontend.

### Changed

- Das Total des Dashboards wird aus den ungerundeten Werten berechnet und kann deshalb um 1 CHF von der Summe der Zeilen abweichen.
- Die Monatsgruppen der Timeline decken den ganzen Zeitraum ab: 12 Gruppen am 1. eines Monats, an jedem anderen Tag 13.
- `go run ./cmd/easyplan` startet jetzt den Server, statt nur die Konfiguration zu melden.
- Eine unbekannte Adresse und eine bekannte Adresse mit der falschen Methode liefern beide die Seite für 404.

### Fixed

- Keine.

### Removed

- Keine.

### Security

- Schreibende Anfragen (POST, PUT, PATCH, DELETE) werden mit Status 403 abgelehnt, wenn der `Origin`-Header fehlt oder nicht zum `Host` der Anfrage passt. Ein Reverse Proxy muss den `Host`-Header erhalten, sonst sind alle Schreibzugriffe gesperrt.

### Noch nicht vorhanden

Der Betrieb folgt mit Phase 3: Container, Compose-Datei, `/health` und der Unterbefehl `healthcheck`, der Backup-Button, die Wiederherstellung und der GitHub-Actions-Workflow. Bis dahin gibt es diese Anleitungen nicht, siehe [Status](index.md).
