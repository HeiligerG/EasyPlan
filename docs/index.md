# EasyPlan

EasyPlan ist eine self-hosted Mini-App in Go, die Fixkosten zusammenrechnet. Sie beantwortet drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel muss pro Konto in den nächsten N Tagen mindestens vorhanden sein?
3. Welche Zahlungen kommen als Nächstes?

Erfasst werden nur Ausgaben. Es gibt keine Einnahmen, keine Kontostände und keine Bank-Anbindung. Die Währung ist fest CHF.

## Status

Phase 3 von 3 ist abgeschlossen: die Berechnung, der Datenspeicher, die Web-Oberfläche und der Betrieb stehen und sind getestet.

| Bereich | Paket | Stand |
|---|---|---|
| Berechnung | `internal/plan` | fertig, getestet |
| Datenspeicher | `internal/store` | fertig, getestet |
| Benutzeroberfläche | `internal/web` | fertig, getestet |
| Server und Konfiguration | `cmd/easyplan` | fertig, getestet |
| Betrieb, Backup, Installation | `Dockerfile`, `compose.yaml` | fertig, getestet |

## Leitplanken

- Ein Binary, eine SQLite-Datei, keine weiteren Dienste.
- Go mit Standardbibliothek, SQLite über `modernc.org/sqlite` ohne CGO, HTMX und CSS als eingebettete Dateien. Kein Node, kein Build-Schritt fürs Frontend, kein ORM.
- Konfiguration über Umgebungsvariablen, keine Einstellungsseite.
- Keine eigene Authentifizierung, der Zugriff regelt ein Reverse Proxy.
- Berechnungen entstehen bei Bedarf live, es werden keine abgeleiteten Daten gespeichert.

## Schnellstart mit Docker

```bash
git clone https://github.com/HeiligerG/EasyPlan.git
cd EasyPlan

# Der Datenordner gehört dem Nutzer 65532 (nonroot) des Containers.
mkdir -p ./data
sudo chown 65532:65532 ./data

docker compose up -d
curl -fsS http://127.0.0.1:8080/health
```

```text
ok
```

Die Oberfläche ist danach im Browser unter `http://127.0.0.1:8080` erreichbar. Die Adresse hört nur auf dem Loopback, von aussen braucht es einen Reverse Proxy.

Die ganze Anleitung steht unter [Betrieb](betrieb.md).

## Schnellstart für die Entwicklung

Go 1.27.3 oder neuer ist nötig (`go.mod`).

```bash
export EASYPLAN_DATA=./data
go run ./cmd/easyplan
```

Der Server hört danach auf `EASYPLAN_ADDR` und legt die Datenbank `easyplan.db` im Datenordner an. Im Log steht:

```text
time=2026-10-31T19:00:00.000+01:00 level=INFO msg="easyplan started" addr=127.0.0.1:8080 data=./data database=data/easyplan.db tz=Europe/Zurich
```

Ausserhalb von Docker hört der Server auf dem Loopback, weil die App keine eigene Authentifizierung hat. Wer sie breiter hören lassen will, setzt `EASYPLAN_ADDR` ausdrücklich, etwa `EASYPLAN_ADDR=:8080`.

## Die Umgebungsvariablen

| Variable | Standard ausserhalb von Docker | Wert im Image | Bedeutung |
|---|---|---|---|
| `EASYPLAN_ADDR` | `127.0.0.1:8080` | `:8080` | Die Adresse, auf der der Server hört |
| `EASYPLAN_DATA` | `/data` | `/data` | Der Ordner mit der SQLite-Datei |
| `TZ` | `Europe/Zurich` | `Europe/Zurich` | Die Zeitzone der Anwendung |

## Weiterführend

- [Architektur](architecture.md): Bausteine, Datenfluss und Datumskonvention
- [Oberfläche](features/oberflaeche.md): die vier Seiten und das Frontend
- [Dashboard](features/dashboard.md): Bedarf, Reserve und nächste Zahlung
- [Einträge](features/eintraege.md): Liste, Formular und Löschbestätigung
- [Konten](features/konten.md): anlegen, umbenennen, sortieren, deaktivieren
- [Timeline](features/timeline.md): die Zahlungen der nächsten zwölf Monate
- [Formate](features/formate.md): Beträge und Daten in Eingabe und Anzeige
- [Server und Betrieb](features/server.md): Start, Konfiguration, `/health` und Beenden
- [Sicherheit](features/sicherheit.md): Origin-Prüfung, Header und Reverse Proxy
- [Berechnung](features/berechnung.md): Monat, Woche, Tag, Reserve und Rundung
- [Wiederholungen](features/wiederholungen.md): Einheiten, Intervalle und Monatsende
- [Datenspeicher](features/datenspeicher.md): Schema, Migrationen und Fehler
- [Betrieb](betrieb.md): Container, Compose-Datei, Healthcheck und Update
- [Backup](backup.md): die Datensicherung und die Wiederherstellung
- [CI und Dependabot](ci.md): die automatischen Prüfungen und Updates
- [Architekturentscheidungen](adr/index.md): die ADRs mit Kontext und Folgen
- [Beitragen](contributing.md): Branch, Prüfungen und Sprachregeln
- [Changelog](changelog.md): was sich geändert hat
