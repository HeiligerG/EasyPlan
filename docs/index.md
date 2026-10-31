# EasyPlan

EasyPlan ist eine self-hosted Mini-App in Go, die Fixkosten zusammenrechnet. Sie beantwortet drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel muss pro Konto in den nächsten N Tagen mindestens vorhanden sein?
3. Welche Zahlungen kommen als Nächstes?

Erfasst werden nur Ausgaben. Es gibt keine Einnahmen, keine Kontostände und keine Bank-Anbindung. Die Währung ist fest CHF.

## Status

Phase 2 von 3 ist abgeschlossen: die Berechnung, der Datenspeicher und die Web-Oberfläche stehen und sind getestet. Der Betrieb mit Container, Backup und Installation folgt mit Phase 3.

| Bereich | Paket | Stand |
|---|---|---|
| Berechnung | `internal/plan` | fertig, getestet |
| Datenspeicher | `internal/store` | fertig, getestet |
| Benutzeroberfläche | `internal/web` | fertig, getestet |
| Server und Konfiguration | `cmd/easyplan` | fertig, getestet |
| Betrieb, Backup, Installation | – | folgt in Phase 3 |

!!! note "Die App ist benutzbar"
    `go run ./cmd/easyplan` startet den Server. Es gibt noch keinen Healthcheck, kein Container-Image und keine Anleitung für die Installation, das folgt mit Phase 3.

## Leitplanken

- Ein Binary, eine SQLite-Datei, keine weiteren Dienste.
- Go mit Standardbibliothek, SQLite über `modernc.org/sqlite` ohne CGO, HTMX und CSS als eingebettete Dateien. Kein Node, kein Build-Schritt fürs Frontend, kein ORM.
- Konfiguration über Umgebungsvariablen, keine Einstellungsseite.
- Keine eigene Authentifizierung, der Zugriff regelt ein Reverse Proxy.
- Berechnungen entstehen bei Bedarf live, es werden keine abgeleiteten Daten gespeichert.

## Schnellstart

Go 1.27.3 oder neuer ist nötig (`go.mod`).

```bash
export EASYPLAN_DATA=./data
go run ./cmd/easyplan
```

Der Server hört danach auf `EASYPLAN_ADDR` und legt die Datenbank `easyplan.db` im Datenordner an. Im Log steht:

```text
time=2026-10-31T19:00:00.000+01:00 level=INFO msg="easyplan started" addr=:8080 data=./data database=data/easyplan.db tz=Europe/Zurich
```

Die Oberfläche ist danach im Browser unter `http://127.0.0.1:8080` erreichbar.

!!! warning "Der Standardwert `:8080` hört auf allen Schnittstellen"
    EasyPlan hat keine eigene Authentifizierung. Für die lokale Entwicklung gehört der Server deshalb nur auf den Loopback:

    ```bash
    EASYPLAN_ADDR=127.0.0.1:8080 EASYPLAN_DATA=./data go run ./cmd/easyplan
    ```

    Die Einzelheiten stehen unter [Server und Betrieb](features/server.md).

Die Umgebungsvariablen mit ihren Standardwerten:

| Variable | Standard | Bedeutung |
|---|---|---|
| `EASYPLAN_ADDR` | `:8080` | Listen-Adresse |
| `EASYPLAN_DATA` | `/data` | Ordner der SQLite-Datei |
| `TZ` | `Europe/Zurich` | Zeitzone |

## Weiterführend

- [Architektur](architecture.md): Bausteine, Datenfluss und Datumskonvention
- [Oberfläche](features/oberflaeche.md): die vier Seiten und das Frontend
- [Dashboard](features/dashboard.md): Bedarf, Reserve und nächste Zahlung
- [Einträge](features/eintraege.md): Liste, Formular und Löschbestätigung
- [Konten](features/konten.md): anlegen, umbenennen, sortieren, deaktivieren
- [Timeline](features/timeline.md): die Zahlungen der nächsten zwölf Monate
- [Formate](features/formate.md): Beträge und Daten in Eingabe und Anzeige
- [Server und Betrieb](features/server.md): Start, Konfiguration und Beenden
- [Sicherheit](features/sicherheit.md): Origin-Prüfung, Header und Reverse Proxy
- [Berechnung](features/berechnung.md): Monat, Woche, Tag, Reserve und Rundung
- [Wiederholungen](features/wiederholungen.md): Einheiten, Intervalle und Monatsende
- [Datenspeicher](features/datenspeicher.md): Schema, Migrationen und Fehler
- [Architekturentscheidungen](adr/index.md): die ADRs mit Kontext und Folgen
- [Beitragen](contributing.md): Branch, Prüfungen und Sprachregeln
- [Changelog](changelog.md): was sich geändert hat
