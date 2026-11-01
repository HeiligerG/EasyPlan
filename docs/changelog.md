# Changelog

Alle Änderungen an EasyPlan stehen in diesem Dokument. Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/), die Versionierung folgt [SemVer](https://semver.org/lang/de/).

Diese Datei ist die einzige Quelle für den Changelog. Eine zweite Kopie im Repository-Root gibt es nicht.

## [Unreleased]

### Added

- Healthcheck unter `GET /health`: Die Antwort ist `ok` mit dem Status 200, wenn die Datenbank erreichbar ist, und `unhealthy` mit dem Status 503, wenn sie es nicht ist.
- Unterbefehl `healthcheck` für das Binary: Er fragt die laufende Anwendung und beendet sich mit 0 bei einer gesunden und mit 1 bei einer kranken Antwort.
- Container-Image auf Basis von distroless: statisches Binary ohne CGO, ohne Shell, als Nutzer 65532, geschrieben wird nur `/data`.
- Compose-Datei für den Betrieb: unbeschreibbares Root-Dateisystem, Port nur auf `127.0.0.1:8080` des Hosts, Daten im Ordner `./data`.
- Der Healthcheck des Containers läuft alle dreissig Sekunden und meldet den Zustand des Dienstes nach aussen.
- Backup als Download: Der Knopf „Backup herunterladen“ im Seitenfuss jeder Seite liefert eine Kopie der Datenbank im Format `easyplan-JJJJMMTT-HHMMSS.db`.
- Wiederherstellung von Hand: Container stoppen, Datei ersetzen, Container starten. Die Anleitung steht in der README.
- CI-Workflow mit vier Jobs: `go` (Formatierung, `vet`, Build ohne CGO, Tests mit `-race`), `vulncheck` (Suche nach bekannten Schwachstellen), `docs` (`mkdocs build --strict`) und `docker` (Build des Images).
- Die CI sucht mit `govulncheck` nach bekannten Schwachstellen im Code und in der Standardbibliothek. Der Job braucht Netzzugriff auf die Schwachstellendatenbank von Go; lokal wurde auf dem aktuellen Stand nichts gefunden.
- Beim Start räumt die Anwendung Verzeichnisse `easyplan-backup-*` weg, die ein abgebrochenes Backup liegen gelassen hat. Ein Fehler dabei wird als Warnung geloggt und verhindert den Start nicht.
- Die Backup-Route hat eine eigene Schreibfrist von zehn Minuten. Die allgemeine Frist des Servers von 60 Sekunden würde eine grosse Datenbank abschneiden, weil sie schon vor dem Kopieren beginnt.
- Dependabot meldet neue Versionen der Aktionen, der Go-Module, der Basis-Images und der Dokumentationswerkzeuge wöchentlich am Montag um 06:00.
- Die Dokumentation hat die Seiten [Betrieb](betrieb.md), [Backup](backup.md) und [CI und Dependabot](ci.md) und die ADRs 0007 bis 0009.
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

- Der Standardwert von `EASYPLAN_ADDR` ist ausserhalb von Docker jetzt `127.0.0.1:8080`. Der Server hört damit per Default nur auf dem Loopback. Im Container-Image steht weiterhin `:8080`, dort bindet die Compose-Datei nur `127.0.0.1:8080:8080` auf dem Host.
- Der Fehler des Healthchecks steht mit dem Level `Warn` im Log statt mit `Error`. Die Container-Runtime fragt alle dreissig Sekunden, ein Fehler käme sonst im Sekundentakt als Fehler im Log.
- Die Healthcheck-URL maskiert die Zonen-ID einer IPv6-Adresse (`%` wird zu `%25`), sonst wäre die URL ungültig.
- Die Basis-Images sind an einen Digest gepinnt, die Aktionen der CI an einen Commit-SHA. Ein Tag kann verschoben werden, ein Commit nicht.
- Die Compose-Datei rotiert das Container-Log (`json-file`, `max-size: 10m`, `max-file: 3`) und gibt dem Container beim Stoppen 15 Sekunden, mehr als die zehn Sekunden, die das Binary für das Beenden braucht.
- Alle Knöpfe und alle Knöpfe als Link sind mindestens 44 Pixel hoch, damit sie mit dem Finger gut zu treffen sind.
- Das Total des Dashboards wird aus den ungerundeten Werten berechnet und kann deshalb um 1 CHF von der Summe der Zeilen abweichen.
- Die Monatsgruppen der Timeline decken den ganzen Zeitraum ab: 12 Gruppen am 1. eines Monats, an jedem anderen Tag 13.
- `go run ./cmd/easyplan` startet jetzt den Server, statt nur die Konfiguration zu melden.
- Eine unbekannte Adresse und eine bekannte Adresse mit der falschen Methode liefern beide die Seite für 404.

### Fixed

- Die Inhaltsliste der README nennt alle 28 Seiten der Dokumentation, auch die neun ADRs und die Seite Beitragen.
- Sprachfehler auf der Seite Server und Betrieb korrigiert: das Wort Container-Runtime braucht den Artikel „eine“.

### Removed

- Keine.

### Security

- Das Backup ist nur über `POST /backup` erreichbar. Ein `GET` muss nach RFC 9110 §9.2.1 sicher sein, ein Backup ist es nicht: Es kopiert die ganze Datenbank und erzeugt Last, ein Link, ein Prefetch oder ein Suchroboter könnte es auslösen. Weil `POST` eine schreibende Methode ist, greift zusätzlich die Prüfung der Herkunft: ohne passenden `Origin` antwortet die Adresse mit 403.
- Die temporäre Kopie des Backups entsteht in einem Ordner mit den Rechten 0700, die Datei hat die Rechte 0600, und der Ordner wird in jedem normalen Weg wieder gelöscht. Was nach einem harten Abbruch liegen bleibt, entfernt die Anwendung beim nächsten Start.
- Der Container läuft ohne Root-Rechte, ohne besondere Linux-Rechte (`cap_drop: [ALL]`) und mit `no-new-privileges`. Das Root-Dateisystem ist unbeschreibbar, geschrieben wird nur `/data`.
- Der Download trägt `Cache-Control: no-store`, damit keine Kopie der Datenbank in einem Zwischencache liegen bleibt.
- Schreibende Anfragen (POST, PUT, PATCH, DELETE) werden mit Status 403 abgelehnt, wenn der `Origin`-Header fehlt oder nicht zum `Host` der Anfrage passt. Ein Reverse Proxy muss den `Host`-Header erhalten, sonst sind alle Schreibzugriffe gesperrt.

### Noch nicht vorhanden

- Es gibt noch keinen Upload für die Wiederherstellung und keinen Export in andere Formate. Beides ist nicht in Version 1 vorgesehen.
- Es gibt noch keine automatische Sicherung. Der Betrieb kann einen Cron-Job mit `sqlite3 easyplan.db ".backup …"` einrichten, siehe [Backup](backup.md).
